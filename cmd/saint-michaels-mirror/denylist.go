// Copyright (c) 2026 Girino Vey.
//
// This software is licensed under Girino's Anarchist License (GAL).
// See LICENSE file for full license text.
// License available at: https://license.girino.org/
package main

import (
	"net"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/girino/nostr-lib/logging"
)

// Denylist holds upstream relay hosts and URL prefixes the service must never
// dial. Its main job is to keep the relay from calling itself: a relay that is
// listed in its own query/broadcast remotes (directly or because a discovered
// relay list advertises it) loops back through its own frontend, trips rate
// limits and keeps probing itself for liveness.
//
// Entries may be:
//   - a bare hostname:      wot.girino.org
//   - a host and port:      relay.example:443
//   - a full URL prefix:    wss://relay.example/sub
//
// Matching is done on the hostname, case-insensitively, so any path under a
// denied host is also denied.
type Denylist struct {
	hosts    map[string]struct{}
	prefixes []string
	entries  []string
}

// NewDenylist builds a denylist from raw entries, ignoring empty ones.
func NewDenylist(entries ...string) *Denylist {
	d := &Denylist{hosts: make(map[string]struct{})}
	for _, raw := range entries {
		d.add(raw)
	}
	return d
}

func (d *Denylist) add(raw string) {
	entry := strings.TrimSpace(raw)
	if entry == "" {
		return
	}
	if strings.Contains(entry, "://") {
		if u, err := url.Parse(entry); err == nil && u.Hostname() != "" {
			d.hosts[strings.ToLower(u.Hostname())] = struct{}{}
		}
		d.prefixes = append(d.prefixes, strings.ToLower(strings.TrimRight(entry, "/")))
		d.entries = append(d.entries, entry)
		return
	}
	host := entry
	if h, _, err := net.SplitHostPort(entry); err == nil {
		host = h
	}
	host = strings.ToLower(strings.Trim(host, "[]"))
	if host == "" {
		return
	}
	d.hosts[host] = struct{}{}
	d.entries = append(d.entries, entry)
}

// AddSelf adds the hostname of the relay's own public URL (RELAY_SERVICE_URL).
func (d *Denylist) AddSelf(serviceURL string) {
	d.add(serviceURL)
}

// AddLocalIPs adds every address assigned to this machine, plus loopback and
// the local hostname, so a relay URL that points at the host by IP or by its
// own name is denied as well (e.g. ws://127.0.0.1:3337, wss://203.0.113.7).
func (d *Denylist) AddLocalIPs() {
	for _, raw := range []string{"127.0.0.1", "::1", "0.0.0.0", "::"} {
		d.addIP(net.ParseIP(raw))
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logging.Warn("upstream denylist: could not list local interfaces: %v", err)
	} else {
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				d.addIP(v.IP)
			case *net.IPAddr:
				d.addIP(v.IP)
			}
		}
	}
	if host, err := os.Hostname(); err == nil {
		d.add(host)
	}
}

// addIP stores an address in canonical form so IPv4 and IPv6 spellings match.
func (d *Denylist) addIP(ip net.IP) {
	if ip == nil {
		return
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	d.add(ip.String())
}

// Match reports whether rawURL points at a denied upstream.
func (d *Denylist) Match(rawURL string) bool {
	if d == nil || len(d.hosts) == 0 && len(d.prefixes) == 0 {
		return false
	}
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if ip := net.ParseIP(host); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			host = v4.String()
		} else {
			host = ip.String()
		}
	}
	if host != "" {
		if _, denied := d.hosts[host]; denied {
			return true
		}
	}
	lower := strings.ToLower(rawURL)
	for _, p := range d.prefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

// Filter splits urls into kept and dropped, preserving the original order.
func (d *Denylist) Filter(urls []string) (kept, dropped []string) {
	for _, u := range urls {
		if d.Match(u) {
			dropped = append(dropped, u)
			continue
		}
		kept = append(kept, u)
	}
	return kept, dropped
}

// Empty reports whether the denylist has no entries.
func (d *Denylist) Empty() bool {
	return d == nil || len(d.hosts) == 0 && len(d.prefixes) == 0
}

// Len returns the number of distinct entries.
func (d *Denylist) Len() int {
	if d == nil {
		return 0
	}
	return len(d.entries)
}

// String returns a stable, comma-separated representation for logs.
func (d *Denylist) String() string {
	if d == nil {
		return ""
	}
	entries := append([]string(nil), d.entries...)
	sort.Strings(entries)
	return strings.Join(entries, ",")
}

// filterAndLog drops denied URLs from a configured list and logs the result.
func filterAndLog(d *Denylist, name string, urls []string) []string {
	kept, dropped := d.Filter(urls)
	for _, u := range dropped {
		logging.Warn("upstream denylist: ignoring %s entry %s (banned upstream)", name, u)
	}
	return kept
}
