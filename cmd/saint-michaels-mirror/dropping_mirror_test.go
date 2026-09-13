// Copyright (c) 2026 Girino Vey.
//
// This software is licensed under Girino's Anarchist License (GAL).
// See LICENSE file for full license text.
// License available at: https://license.girino.org/
package main

import (
	"testing"
	"time"

	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"
)

func TestEnqueueMirrorEventDropsWhenFull(t *testing.T) {
	ch := make(chan *nostr.Event, 2)
	a := &nostr.Event{ID: "a"}
	b := &nostr.Event{ID: "b"}
	c := &nostr.Event{ID: "c"}
	if !enqueueMirrorEvent(ch, a) || !enqueueMirrorEvent(ch, b) {
		t.Fatal("expected first two enqueues to succeed")
	}
	if enqueueMirrorEvent(ch, c) {
		t.Fatal("expected full queue to drop")
	}
	if got := <-ch; got.ID != "a" {
		t.Fatalf("got %s, want a", got.ID)
	}
}

func TestDroppingMirrorStatsGreen(t *testing.T) {
	d := &droppingMirror{}
	d.mirrored.Store(10)
	d.live.Store(2)
	obj := d.GetStats()
	snap := collectHealthSnapshot(nil, d, nil, nil)
	if snap.MirrorHealthState != HealthGreen {
		t.Fatalf("state=%s want GREEN", snap.MirrorHealthState)
	}
	if obj == nil {
		t.Fatal("nil stats")
	}
}

func TestWsLimiterCap(t *testing.T) {
	l := &wsLimiter{}
	l.n.Store(maxWebsocketConnections)
	if !l.reject(nil) {
		t.Fatal("expected reject at cap")
	}
	l.onDisconnect(nil)
	if l.reject(nil) {
		t.Fatal("expected accept after disconnect")
	}
}

func TestNoteDropLogsAtMostOncePerInterval(t *testing.T) {
	d := &droppingMirror{}
	d.lastDropLog.Store(time.Now().Unix())
	for i := 0; i < 100; i++ {
		d.noteDrop()
	}
	if d.dropped.Load() != 100 {
		t.Fatalf("dropped=%d want 100", d.dropped.Load())
	}
}

func TestClientWriterDropsWhenFull(t *testing.T) {
	w := &clientWriter{ch: make(chan nostr.EventEnvelope, 1), ws: &khatru.WebSocket{}}
	env := nostr.EventEnvelope{}
	if !w.trySend(env) {
		t.Fatal("first send should succeed")
	}
	if w.trySend(env) {
		t.Fatal("full per-client queue should drop only that client")
	}
}

func TestFanoutMatchesFilter(t *testing.T) {
	h := newClientHub()
	ws := &khatru.WebSocket{}
	w := &clientWriter{ws: ws, ch: make(chan nostr.EventEnvelope, 4)}
	h.writers[ws] = w
	h.subs = []liveSub{{ws: ws, id: "sub1", filter: nostr.Filter{Kinds: []int{1}}}}

	if n := h.fanout(&nostr.Event{Kind: 1, ID: "a"}); n != 1 {
		t.Fatalf("kind1 matched=%d want 1", n)
	}
	if n := h.fanout(&nostr.Event{Kind: 7, ID: "b"}); n != 0 {
		t.Fatalf("kind7 matched=%d want 0", n)
	}
	if h.shouldSkipSyncWrite(ws) != true {
		t.Fatal("managed sockets should skip khatru sync writes")
	}
	if h.shouldSkipSyncWrite(&khatru.WebSocket{}) {
		t.Fatal("unknown sockets should not skip")
	}
}

func TestSubTrackerCountsPerSocket(t *testing.T) {
	s := newSubTracker()
	if s.count() != 0 {
		t.Fatalf("count=%d want 0", s.count())
	}
	s.noteREQ(nil)
	if s.count() != 0 {
		t.Fatal("nil websocket should be ignored")
	}

	ws := &khatru.WebSocket{}
	s.noteREQ(ws)
	s.noteREQ(ws)
	if s.count() != 2 {
		t.Fatalf("count=%d want 2", s.count())
	}
	s.removeWS(ws)
	if s.count() != 0 {
		t.Fatalf("after disconnect count=%d want 0", s.count())
	}
}
