// Copyright (c) 2026 Girino Vey.
//
// This software is licensed under Girino's Anarchist License (GAL).
// See LICENSE file for full license text.
// License available at: https://license.girino.org/
//
// NIP-42 pubkey whitelist for client access to Espelho de São Miguel.
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/fiatjaf/khatru"
	"github.com/girino/nostr-lib/logging"
	"github.com/nbd-wtf/go-nostr"
	nip19 "github.com/nbd-wtf/go-nostr/nip19"
)

const (
	whitelistAuthRequiredMsg = "auth-required: authenticate with a whitelisted npub"
	whitelistRestrictedMsg   = "restricted: pubkey not whitelisted"
)

// PubkeyWhitelist restricts REQ, COUNT, and EVENT access to authenticated pubkeys.
// An empty/disabled whitelist leaves the relay open (existing behavior).
type PubkeyWhitelist struct {
	pubkeys map[string]struct{}
}

// ParsePubkeyWhitelist parses a comma-separated list of npub or 64-char hex pubkeys.
// Empty input disables the whitelist.
func ParsePubkeyWhitelist(raw string) (*PubkeyWhitelist, error) {
	w := &PubkeyWhitelist{pubkeys: make(map[string]struct{})}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		pk, err := normalizePubkey(entry)
		if err != nil {
			return nil, err
		}
		w.pubkeys[pk] = struct{}{}
	}
	return w, nil
}

func normalizePubkey(entry string) (string, error) {
	if strings.HasPrefix(strings.ToLower(entry), "npub1") {
		prefix, val, err := nip19.Decode(entry)
		if err != nil {
			return "", fmt.Errorf("invalid npub %q: %w", entry, err)
		}
		if prefix != "npub" {
			return "", fmt.Errorf("expected npub, got %s: %s", prefix, entry)
		}
		hexpk, ok := val.(string)
		if !ok {
			return "", fmt.Errorf("unexpected npub payload type for %q", entry)
		}
		return strings.ToLower(hexpk), nil
	}

	hexpk := strings.ToLower(entry)
	b, err := hex.DecodeString(hexpk)
	if err != nil || len(b) != 32 {
		return "", fmt.Errorf("invalid pubkey %q: want npub or 64-char hex", entry)
	}
	return hexpk, nil
}

// Enabled reports whether access control is active.
func (w *PubkeyWhitelist) Enabled() bool {
	return w != nil && len(w.pubkeys) > 0
}

// Len returns the number of allowed pubkeys.
func (w *PubkeyWhitelist) Len() int {
	if w == nil {
		return 0
	}
	return len(w.pubkeys)
}

// Contains reports whether hexPubkey is on the whitelist.
func (w *PubkeyWhitelist) Contains(hexPubkey string) bool {
	if w == nil {
		return false
	}
	_, ok := w.pubkeys[strings.ToLower(hexPubkey)]
	return ok
}

// checkAccess returns a reject decision for an already-resolved authed pubkey.
func (w *PubkeyWhitelist) checkAccess(authedHex string) (reject bool, msg string) {
	if !w.Enabled() {
		return false, ""
	}
	if authedHex == "" {
		return true, whitelistAuthRequiredMsg
	}
	if !w.Contains(authedHex) {
		return true, whitelistRestrictedMsg
	}
	return false, ""
}

// RejectFilter rejects REQ unless the client authenticated as a whitelisted pubkey.
func (w *PubkeyWhitelist) RejectFilter(ctx context.Context, filter nostr.Filter) (reject bool, msg string) {
	return w.rejectAuthed(ctx, "filter")
}

// RejectCountFilter rejects COUNT unless the client authenticated as a whitelisted pubkey.
func (w *PubkeyWhitelist) RejectCountFilter(ctx context.Context, filter nostr.Filter) (reject bool, msg string) {
	return w.rejectAuthed(ctx, "count")
}

// RejectEvent rejects EVENT unless the client authenticated as a whitelisted pubkey.
func (w *PubkeyWhitelist) RejectEvent(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
	return w.rejectAuthed(ctx, "event")
}

func (w *PubkeyWhitelist) rejectAuthed(ctx context.Context, kind string) (reject bool, msg string) {
	authed := khatru.GetAuthed(ctx)
	reject, msg = w.checkAccess(authed)
	if reject {
		logging.Warn("whitelist rejected %s: %s, authed=%s, from=%s", kind, msg, authed, khatru.GetIP(ctx))
	}
	return reject, msg
}
