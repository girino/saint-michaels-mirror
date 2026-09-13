// Copyright (c) 2026 Girino Vey.
//
// This software is licensed under Girino's Anarchist License (GAL).
// See LICENSE file for full license text.
// License available at: https://license.girino.org/
package main

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/fiatjaf/khatru"
	"github.com/girino/nostr-lib/logging"
)

// Hard cap on simultaneous websockets. Idle crawlers were able to open tens of
// thousands of connections (rate limit is per-IP, and IPs are cheap).
const maxWebsocketConnections = 256

type wsLimiter struct {
	n atomic.Int64
}

func (l *wsLimiter) reject(r *http.Request) bool {
	if l.n.Load() >= maxWebsocketConnections {
		from := ""
		if r != nil {
			from = khatru.GetIPFromRequest(r)
		}
		logging.Warn("connection rejected: at cap (%d), from=%s", maxWebsocketConnections, from)
		return true
	}
	return false
}

func (l *wsLimiter) onConnect(_ context.Context) {
	l.n.Add(1)
}

func (l *wsLimiter) onDisconnect(_ context.Context) {
	if l.n.Add(-1) < 0 {
		l.n.Store(0)
	}
}

func (l *wsLimiter) count() int64 { return l.n.Load() }
