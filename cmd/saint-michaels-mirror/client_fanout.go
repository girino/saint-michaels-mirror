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
	"github.com/girino/nostr-lib/logging"
	"github.com/nbd-wtf/go-nostr"
)

const (
	clientWriteQueue     = 64
	slowWriteLogAfter    = 200 * time.Millisecond
	slowWriteDisconnect  = 3 * time.Second
	stuckWriteDisconnect = 3 * time.Second
)

type liveSub struct {
	ws     *khatru.WebSocket
	id     string
	filter nostr.Filter
}

type clientWriter struct {
	ws      *khatru.WebSocket
	ch      chan nostr.EventEnvelope
	busyAt  atomic.Int64
	skipped atomic.Int64
	closed  atomic.Bool
}

type clientHub struct {
	mu      sync.Mutex
	writers map[*khatru.WebSocket]*clientWriter
	subs    []liveSub
	skipped atomic.Int64
	slow    atomic.Int64
	kicked  atomic.Int64
}

func newClientHub() *clientHub {
	return &clientHub{writers: make(map[*khatru.WebSocket]*clientWriter)}
}

func safeSubID(ctx context.Context) (id string) {
	defer func() {
		if recover() != nil {
			id = ""
		}
	}()
	return khatru.GetSubscriptionID(ctx)
}

func stringPtr(s string) *string { return &s }

func (h *clientHub) count() int64 {
	h.mu.Lock()
	n := int64(len(h.subs))
	h.mu.Unlock()
	return n
}

func (h *clientHub) noteREQ(ctx context.Context, filter nostr.Filter) {
	ws := khatru.GetConnection(ctx)
	if ws == nil {
		return
	}
	id := safeSubID(ctx)
	h.mu.Lock()
	h.subs = append(h.subs, liveSub{ws: ws, id: id, filter: filter})
	if _, ok := h.writers[ws]; !ok {
		w := &clientWriter{
			ws: ws,
			ch: make(chan nostr.EventEnvelope, clientWriteQueue),
		}
		h.writers[ws] = w
		go w.loop(h)
	}
	h.mu.Unlock()
}

func (h *clientHub) removeWS(ws *khatru.WebSocket) {
	if ws == nil {
		return
	}
	h.mu.Lock()
	kept := h.subs[:0]
	for _, s := range h.subs {
		if s.ws != ws {
			kept = append(kept, s)
		}
	}
	h.subs = kept
	if w, ok := h.writers[ws]; ok {
		w.close()
		delete(h.writers, ws)
	}
	h.mu.Unlock()
}

func (h *clientHub) writerFor(ws *khatru.WebSocket) *clientWriter {
	h.mu.Lock()
	w := h.writers[ws]
	h.mu.Unlock()
	return w
}

// shouldSkipSyncWrite tells khatru to skip its sequential WriteJSON for sockets
// we already fan out to asynchronously (or that are currently blocked).
func (h *clientHub) shouldSkipSyncWrite(ws *khatru.WebSocket) bool {
	w := h.writerFor(ws)
	return w != nil
}

func (h *clientHub) fanout(evt *nostr.Event) int {
	if evt == nil {
		return 0
	}
	h.mu.Lock()
	subs := append([]liveSub(nil), h.subs...)
	h.mu.Unlock()

	matched := 0
	for i := range subs {
		s := &subs[i]
		if !s.filter.Matches(evt) {
			continue
		}
		w := h.writerFor(s.ws)
		if w == nil {
			continue
		}
		env := nostr.EventEnvelope{
			SubscriptionID: stringPtr(s.id),
			Event:          *evt,
		}
		if w.trySend(env) {
			matched++
		} else {
			h.skipped.Add(1)
		}
	}
	return matched
}

func (w *clientWriter) trySend(env nostr.EventEnvelope) bool {
	if w.closed.Load() {
		return false
	}
	select {
	case w.ch <- env:
		return true
	default:
		w.skipped.Add(1)
		return false
	}
}

func (w *clientWriter) close() {
	if w.closed.CompareAndSwap(false, true) {
		close(w.ch)
	}
}

func wsIP(ws *khatru.WebSocket) string {
	if ws == nil || ws.Request == nil {
		return ""
	}
	return khatru.GetIPFromRequest(ws.Request)
}

func (w *clientWriter) loop(h *clientHub) {
	for env := range w.ch {
		w.busyAt.Store(time.Now().UnixNano())
		start := time.Now()
		err := w.ws.WriteJSON(env)
		dur := time.Since(start)
		w.busyAt.Store(0)

		ip := wsIP(w.ws)
		pk := w.ws.AuthedPublicKey
		if dur >= slowWriteLogAfter {
			h.slow.Add(1)
			logging.Warn("slow websocket write: dur=%s ip=%s pubkey=%s queue=%d err=%v",
				dur.Round(time.Millisecond), ip, pk, len(w.ch), err)
		}
		if err != nil {
			logging.Info("websocket write failed: ip=%s pubkey=%s err=%v", ip, pk, err)
			disconnectClient(w.ws, "write failed")
			return
		}
		if dur >= slowWriteDisconnect {
			h.kicked.Add(1)
			logging.Warn("disconnecting slow websocket: dur=%s ip=%s pubkey=%s",
				dur.Round(time.Millisecond), ip, pk)
			disconnectClient(w.ws, "slow websocket write")
			return
		}
	}
}

func (h *clientHub) kickStuckWriters() {
	now := time.Now().UnixNano()
	h.mu.Lock()
	var stuck []*khatru.WebSocket
	for ws, w := range h.writers {
		at := w.busyAt.Load()
		if at == 0 {
			continue
		}
		blocked := time.Duration(now - at)
		if blocked < stuckWriteDisconnect {
			continue
		}
		stuck = append(stuck, ws)
		logging.Warn("websocket write stuck for %s ip=%s pubkey=%s; disconnecting that client only",
			blocked.Round(time.Millisecond), wsIP(ws), ws.AuthedPublicKey)
	}
	h.mu.Unlock()
	for _, ws := range stuck {
		h.kicked.Add(1)
		disconnectClient(ws, "stuck websocket write")
	}
}
