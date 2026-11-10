// Copyright (c) 2026 Girino Vey.
//
// This software is licensed under Girino's Anarchist License (GAL).
// See LICENSE file for full license text.
// License available at: https://license.girino.org/
package main

import (
	"testing"

	"github.com/girino/nostr-lib/broadcast/manager"
)

func TestDenylistMatch(t *testing.T) {
	d := NewDenylist("wot.girino.org", "relay.example:443", "wss://prefix.example/sub")

	tests := []struct {
		url  string
		want bool
	}{
		{"wss://wot.girino.org", true},
		{"wss://wot.girino.org/lima-haven", true},
		{"WSS://WOT.GIRINO.ORG", true},
		{"wss://relay.example:443", true},
		{"wss://relay.example", true},
		{"wss://prefix.example/sub/x", true},
		{"wss://prefix.example", true},
		{"wss://nostr.girino.org", false},
		{"wss://wot.girino.org.evil.com", false},
		{"", false},
		{"not a url", false},
	}
	for _, tt := range tests {
		if got := d.Match(tt.url); got != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestDenylistAddSelf(t *testing.T) {
	d := NewDenylist()
	d.AddSelf("https://espelho.girino.org")

	if !d.Match("wss://espelho.girino.org") {
		t.Fatal("own public hostname must be denied")
	}
	if d.Match("wss://nostr.girino.org") {
		t.Fatal("a different hostname must not be denied")
	}
}

func TestDenylistFilter(t *testing.T) {
	d := NewDenylist("wot.girino.org")
	kept, dropped := d.Filter([]string{"wss://a.example", "wss://wot.girino.org", "wss://b.example"})

	if len(dropped) != 1 || dropped[0] != "wss://wot.girino.org" {
		t.Fatalf("dropped = %v, want [wss://wot.girino.org]", dropped)
	}
	if len(kept) != 2 || kept[0] != "wss://a.example" || kept[1] != "wss://b.example" {
		t.Fatalf("kept = %v, want the original order without the denied entry", kept)
	}
}

func TestDenylistEmpty(t *testing.T) {
	d := NewDenylist()
	if !d.Empty() {
		t.Fatal("new denylist should be empty")
	}
	d.AddSelf("")
	if !d.Empty() {
		t.Fatal("empty entries must be ignored")
	}
	if d.Match("wss://anything.example") {
		t.Fatal("empty denylist must not match")
	}

	d.AddSelf("https://relay.example")
	if d.Empty() {
		t.Fatal("denylist should not be empty after AddSelf")
	}
	if got := d.Len(); got != 1 {
		t.Fatalf("Len() = %d, want 1", got)
	}
}

func TestPruneDeniedRelays(t *testing.T) {
	mgr := manager.NewManager(10, 0.9)
	mgr.AddRelay("wss://wot.girino.org")
	mgr.AddRelay("wss://wot.girino.org/sierra")
	mgr.AddRelay("wss://relay.damus.io")

	pruneDeniedRelays(NewDenylist("wot.girino.org"), mgr)

	if got := mgr.GetRelayCount(); got != 1 {
		t.Fatalf("relay count = %d, want 1", got)
	}
	for _, u := range mgr.GetAllRelays() {
		if u != "wss://relay.damus.io" {
			t.Fatalf("unexpected relay left in manager: %s", u)
		}
	}
}

func TestPruneDeniedRelaysNilManager(t *testing.T) {
	// Must not panic when the broadcast store is disabled.
	pruneDeniedRelays(NewDenylist("wot.girino.org"), nil)
	pruneDeniedRelays(NewDenylist(), manager.NewManager(10, 0.9))
}
