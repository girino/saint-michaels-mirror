// Copyright (c) 2026 Girino Vey.
//
// This software is licensed under Girino's Anarchist License (GAL).
// See LICENSE file for full license text.
// License available at: https://license.girino.org/
package main

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fiatjaf/khatru"
	jsonlib "github.com/girino/nostr-lib/json"
	"github.com/girino/nostr-lib/logging"
	"github.com/nbd-wtf/go-nostr"
)

const (
	mirrorQueueSize        = 4096
	mirrorDropLogInterval  = 10 * time.Second
	mirrorIdlePollInterval = 250 * time.Millisecond
	mirrorListenerPoll     = 250 * time.Millisecond
)

type droppingMirror struct {
	mirrored    atomic.Int64
	dropped     atomic.Int64
	live        atomic.Int64
	dead        atomic.Int64
	fails       atomic.Int64
	listeners   atomic.Int64
	lastDropLog atomic.Int64 // unix seconds
	hub         *clientHub
}

func (d *droppingMirror) GetStatsName() string { return "mirror" }

func (d *droppingMirror) GetStats() jsonlib.JsonEntity {
	mirrored := d.mirrored.Load()
	dropped := d.dropped.Load()
	fails := d.fails.Load()
	state := HealthGreen
	if fails >= 10 {
		state = HealthRed
	} else if fails > 2 {
		state = HealthYellow
	}
	obj := jsonlib.NewJsonObject()
	obj.Set("mirrored_events", jsonlib.NewJsonValue(mirrored))
	obj.Set("mirror_successes", jsonlib.NewJsonValue(mirrored))
	obj.Set("mirror_failures", jsonlib.NewJsonValue(fails))
	obj.Set("consecutive_mirror_failures", jsonlib.NewJsonValue(fails))
	obj.Set("mirror_health_state", jsonlib.NewJsonValue(state))
	obj.Set("live_relays", jsonlib.NewJsonValue(d.live.Load()))
	obj.Set("dead_relays", jsonlib.NewJsonValue(d.dead.Load()))
	obj.Set("dropped_events", jsonlib.NewJsonValue(dropped))
	obj.Set("listening_filters", jsonlib.NewJsonValue(d.listeners.Load()))
	if d.hub != nil {
		obj.Set("client_queue_skips", jsonlib.NewJsonValue(d.hub.skipped.Load()))
		obj.Set("slow_writes", jsonlib.NewJsonValue(d.hub.slow.Load()))
		obj.Set("slow_disconnects", jsonlib.NewJsonValue(d.hub.kicked.Load()))
	}
	return obj
}

func (d *droppingMirror) noteDrop() {
	n := d.dropped.Add(1)
	now := time.Now().Unix()
	last := d.lastDropLog.Load()
	if last != 0 && now-last < int64(mirrorDropLogInterval.Seconds()) {
		return
	}
	if d.lastDropLog.CompareAndSwap(last, now) {
		logging.Warn("mirror dropped %d events total (broadcast busy or queue full, listeners=%d)",
			n, d.listeners.Load())
	}
}

// enqueueMirrorEvent is non-blocking. If the queue is full the event is dropped.
func enqueueMirrorEvent(queue chan *nostr.Event, evt *nostr.Event) bool {
	select {
	case queue <- evt:
		return true
	default:
		return false
	}
}

// subTracker counts allowed REQs per websocket. khatru.GetListeningFilters races
// (copies len then ranges a slice mutated under another goroutine) and panics.
type subTracker struct {
	mu    sync.Mutex
	perWS map[*khatru.WebSocket]int
	total atomic.Int64
}

func newSubTracker() *subTracker {
	return &subTracker{perWS: make(map[*khatru.WebSocket]int)}
}

func (s *subTracker) noteREQ(ws *khatru.WebSocket) {
	if ws == nil {
		return
	}
	s.mu.Lock()
	s.perWS[ws]++
	s.mu.Unlock()
	s.total.Add(1)
}

func (s *subTracker) removeWS(ws *khatru.WebSocket) {
	if ws == nil {
		return
	}
	s.mu.Lock()
	n := s.perWS[ws]
	delete(s.perWS, ws)
	s.mu.Unlock()
	if n > 0 {
		s.total.Add(-int64(n))
	}
}

func (s *subTracker) count() int64 {
	n := s.total.Load()
	if n < 0 {
		return 0
	}
	return n
}

func startDroppingMirror(ctx context.Context, queryURLs []string, relay *khatru.Relay) *droppingMirror {
	d := &droppingMirror{hub: newClientHub()}
	d.live.Store(int64(len(queryURLs)))

	pool := nostr.NewSimplePool(ctx, nostr.WithPenaltyBox())
	queue := make(chan *nostr.Event, mirrorQueueSize)

	// Last RejectFilter: only runs when earlier hooks allowed the REQ.
	relay.RejectFilter = append(relay.RejectFilter, func(ctx context.Context, filter nostr.Filter) (bool, string) {
		d.hub.noteREQ(ctx, filter)
		d.listeners.Store(d.hub.count())
		return false, ""
	})
	relay.OnDisconnect = append(relay.OnDisconnect, func(ctx context.Context) {
		d.hub.removeWS(khatru.GetConnection(ctx))
		d.listeners.Store(d.hub.count())
	})
	// Skip khatru's sequential WriteJSON for sockets we already write to
	// asynchronously, so a published EVENT cannot stall the handler either.
	relay.PreventBroadcast = append(relay.PreventBroadcast, func(ws *khatru.WebSocket, _ *nostr.Event) bool {
		return d.hub.shouldSkipSyncWrite(ws)
	})
	relay.OnEventSaved = append(relay.OnEventSaved, func(_ context.Context, evt *nostr.Event) {
		d.hub.fanout(evt)
	})

	go d.supervise(ctx, "ingest", func() { d.ingestLoop(ctx, queryURLs, pool, queue) })
	go d.supervise(ctx, "broadcast", func() { d.broadcastLoop(ctx, queue) })
	go d.stuckWriteWatchdog(ctx)
	go d.monitorQueryRelays(ctx, queryURLs, pool)

	logging.Info("dropping mirror started: %d query remotes, queue=%d (async per-client writes; subscribe only while clients are listening)",
		len(queryURLs), mirrorQueueSize)
	return d
}

func (d *droppingMirror) supervise(ctx context.Context, name string, fn func()) {
	for ctx.Err() == nil {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					logging.Error("mirror %s panic: %v", name, rec)
				}
			}()
			fn()
		}()
		if ctx.Err() != nil {
			return
		}
		time.Sleep(time.Second)
	}
}

// ingestLoop runs the firehose only while at least one client has an open REQ.
// That avoids pulling every event from every query remote into a queue nobody will read.
func (d *droppingMirror) ingestLoop(ctx context.Context, urls []string, pool *nostr.SimplePool, queue chan *nostr.Event) {
	for {
		if ctx.Err() != nil {
			return
		}
		n := d.hub.count()
		d.listeners.Store(n)
		if n == 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(mirrorIdlePollInterval):
				continue
			}
		}

		subCtx, cancel := context.WithCancel(ctx)
		now := nostr.Now()
		sub := pool.SubscribeMany(subCtx, urls, nostr.Filter{Since: &now})
		logging.Info("mirror firehose subscribed (%d listening filters)", n)

		stopWatch := make(chan struct{})
		go func() {
			t := time.NewTicker(mirrorListenerPoll)
			defer t.Stop()
			defer close(stopWatch)
			for {
				select {
				case <-subCtx.Done():
					return
				case <-t.C:
					cur := d.hub.count()
					d.listeners.Store(cur)
					if cur == 0 {
						logging.Info("mirror firehose stopping: no listening clients")
						cancel()
						return
					}
				}
			}
		}()

		for {
			select {
			case <-subCtx.Done():
				<-stopWatch
				cancel()
				goto next
			case ev, ok := <-sub:
				if !ok {
					cancel()
					<-stopWatch
					goto next
				}
				if ev.Event == nil {
					continue
				}
				if d.hub.count() == 0 {
					continue
				}
				if !enqueueMirrorEvent(queue, ev.Event) {
					d.noteDrop()
				}
			}
		}
	next:
		cancel()
	}
}

func (d *droppingMirror) broadcastLoop(ctx context.Context, queue chan *nostr.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-queue:
			if d.hub.count() == 0 {
				continue
			}
			start := time.Now()
			matched := d.hub.fanout(evt)
			d.mirrored.Add(1)
			if dur := time.Since(start); dur >= 50*time.Millisecond {
				logging.Warn("mirror fanout slow: dur=%s matched=%d listeners=%d client_skips=%d",
					dur.Round(time.Millisecond), matched, d.hub.count(), d.hub.skipped.Load())
			}
		}
	}
}

func (d *droppingMirror) stuckWriteWatchdog(ctx context.Context) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			d.hub.kickStuckWriters()
		}
	}
}

func (d *droppingMirror) monitorQueryRelays(ctx context.Context, urls []string, pool *nostr.SimplePool) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	d.checkQueryRelays(urls, pool)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.checkQueryRelays(urls, pool)
		}
	}
}

func (d *droppingMirror) checkQueryRelays(urls []string, pool *nostr.SimplePool) {
	dead := int64(0)
	for _, url := range urls {
		if _, err := pool.EnsureRelay(url); err != nil {
			dead++
		}
	}
	total := int64(len(urls))
	d.dead.Store(dead)
	d.live.Store(total - dead)
	if total > 0 && dead > total/2 {
		d.fails.Add(1)
	} else {
		d.fails.Store(0)
	}
	skips, slow, kicked := int64(0), int64(0), int64(0)
	if d.hub != nil {
		skips = d.hub.skipped.Load()
		slow = d.hub.slow.Load()
		kicked = d.hub.kicked.Load()
	}
	logging.Info("mirror stats: mirrored=%d dropped=%d live=%d dead=%d listeners=%d client_skips=%d slow_writes=%d slow_disconnects=%d",
		d.mirrored.Load(), d.dropped.Load(), d.live.Load(), d.dead.Load(), d.listeners.Load(), skips, slow, kicked)
}
