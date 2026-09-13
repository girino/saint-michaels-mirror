// Copyright (c) 2026 Girino Vey.
//
// This software is licensed under Girino's Anarchist License (GAL).
// See LICENSE file for full license text.
// License available at: https://license.girino.org/
package main

import (
	"strings"
	"testing"

	nip19 "github.com/nbd-wtf/go-nostr/nip19"
)

func mustNpub(t *testing.T, hexpk string) string {
	t.Helper()
	npub, err := nip19.EncodePublicKey(hexpk)
	if err != nil {
		t.Fatalf("encode npub: %v", err)
	}
	return npub
}

func TestParsePubkeyWhitelistEmpty(t *testing.T) {
	w, err := ParsePubkeyWhitelist("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Enabled() {
		t.Fatal("empty list should disable whitelist")
	}
	if w.Len() != 0 {
		t.Fatalf("Len = %d, want 0", w.Len())
	}
}

func TestParsePubkeyWhitelistNpubAndHex(t *testing.T) {
	hex1 := "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	hex2 := strings.ToUpper("c6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5")
	npub1 := mustNpub(t, hex1)

	w, err := ParsePubkeyWhitelist(npub1 + ", " + hex2 + ",,")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !w.Enabled() {
		t.Fatal("expected whitelist to be enabled")
	}
	if w.Len() != 2 {
		t.Fatalf("Len = %d, want 2", w.Len())
	}
	if !w.Contains(hex1) {
		t.Fatalf("missing npub-decoded pubkey %s", hex1)
	}
	if !w.Contains(strings.ToLower(hex2)) {
		t.Fatalf("missing hex pubkey %s", hex2)
	}
}

func TestParsePubkeyWhitelistDedupes(t *testing.T) {
	hexpk := "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	npub := mustNpub(t, hexpk)

	w, err := ParsePubkeyWhitelist(hexpk + "," + npub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Len() != 1 {
		t.Fatalf("Len = %d, want 1", w.Len())
	}
}

func TestParsePubkeyWhitelistInvalid(t *testing.T) {
	cases := []string{
		"not-a-key",
		"npub1invalid",
		"abcd",
		"zzzz667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
	}
	for _, raw := range cases {
		if _, err := ParsePubkeyWhitelist(raw); err == nil {
			t.Errorf("ParsePubkeyWhitelist(%q) succeeded, want error", raw)
		}
	}
}

func TestCheckAccess(t *testing.T) {
	hexpk := "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	other := "c6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5"
	w, err := ParsePubkeyWhitelist(hexpk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if reject, _ := w.checkAccess(hexpk); reject {
		t.Fatal("whitelisted pubkey should be allowed")
	}
	if reject, msg := w.checkAccess(""); !reject || msg != whitelistAuthRequiredMsg {
		t.Fatalf("unauthenticated: reject=%v msg=%q", reject, msg)
	}
	if reject, msg := w.checkAccess(other); !reject || msg != whitelistRestrictedMsg {
		t.Fatalf("unknown pubkey: reject=%v msg=%q", reject, msg)
	}
}

func TestCheckAccessDisabled(t *testing.T) {
	w, err := ParsePubkeyWhitelist("  ,  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reject, msg := w.checkAccess(""); reject || msg != "" {
		t.Fatalf("disabled whitelist should allow anyone, got reject=%v msg=%q", reject, msg)
	}
}
