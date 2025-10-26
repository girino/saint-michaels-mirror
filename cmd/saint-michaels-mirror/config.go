// Copyright (c) 2025 Girino Vey.
//
// This software is licensed under Girino's Anarchist License (GAL).
// See LICENSE file for full license text.
// License available at: https://license.girino.org/
//
// Configuration management for Espelho de São Miguel.
package main

import (
	"flag"
	"os"
	"strings"

	"github.com/fiatjaf/khatru"
)

// Config holds runtime configuration coming from environment and CLI flags.
type Config struct {
	Addr           string
	PublishRemotes []string
	QueryRemotes   []string
	Verbose        string

	RelayServiceURL  string
	RelayName        string
	RelayDescription string
	RelayContact     string
	RelaySecKey      string
	RelayPubKey      string
	RelayIcon        string
	RelayBanner      string

	// Broadcast settings
	BroadcastTopN            int
	BroadcastWorkers         int
	BroadcastCacheTTL        string
	BroadcastSeedRelays      []string
	BroadcastMandatoryRelays []string
}

// LoadConfig reads environment variables and flags. Flags override env values.
func LoadConfig() *Config {
	envAddr := os.Getenv("ADDR")
	if envAddr == "" {
		envAddr = ":3337"
	}
	envRemotes := os.Getenv("PUBLISH_REMOTES")
	envQueryRemotes := os.Getenv("QUERY_REMOTES")
	envVerbose := os.Getenv("VERBOSE")

	addr := flag.String("addr", envAddr, "address to listen on")
	remotes := flag.String("remotes", envRemotes, "comma-separated list of remote relay URLs to forward events to (env: PUBLISH_REMOTES)")
	queryRemotes := flag.String("query-remotes", envQueryRemotes, "comma-separated list of remote relay URLs to use for queries/subscriptions (env: QUERY_REMOTES)")
	verbose := flag.String("verbose", envVerbose, "verbose logging control: '1'/'true' for all, 'relaystore' for module, 'relaystore.QueryEvents,mirror' for specific methods (env: VERBOSE)")
	flag.Parse()

	pub := []string{}
	qry := []string{}
	if *remotes != "" {
		pub = strings.Split(*remotes, ",")
	}
	if *queryRemotes != "" {
		qry = strings.Split(*queryRemotes, ",")
	}

	// Parse broadcast settings
	broadcastTopN := 10
	if os.Getenv("BROADCAST_TOP_N") != "" {
		broadcastTopN = 10 // default
	}

	broadcastWorkers := 5
	if os.Getenv("BROADCAST_WORKERS") != "" {
		broadcastWorkers = 5 // default
	}

	broadcastCacheTTL := "1h"
	if os.Getenv("BROADCAST_CACHE_TTL") != "" {
		broadcastCacheTTL = os.Getenv("BROADCAST_CACHE_TTL")
	}

	broadcastSeedRelays := []string{}
	if os.Getenv("BROADCAST_SEED_RELAYS") != "" {
		broadcastSeedRelays = strings.Split(os.Getenv("BROADCAST_SEED_RELAYS"), ",")
	}

	broadcastMandatoryRelays := []string{}
	if os.Getenv("BROADCAST_MANDATORY_RELAYS") != "" {
		broadcastMandatoryRelays = strings.Split(os.Getenv("BROADCAST_MANDATORY_RELAYS"), ",")
	}

	cfg := &Config{
		Addr:           *addr,
		PublishRemotes: pub,
		QueryRemotes:   qry,
		Verbose:        *verbose,

		RelayServiceURL:  os.Getenv("RELAY_SERVICE_URL"),
		RelayName:        os.Getenv("RELAY_NAME"),
		RelayDescription: os.Getenv("RELAY_DESCRIPTION"),
		RelayContact:     os.Getenv("RELAY_CONTACT"),
		RelaySecKey:      os.Getenv("RELAY_SECKEY"),
		RelayPubKey:      os.Getenv("RELAY_PUBKEY"),
		RelayIcon:        os.Getenv("RELAY_ICON"),
		RelayBanner:      os.Getenv("RELAY_BANNER"),

		BroadcastTopN:            broadcastTopN,
		BroadcastWorkers:         broadcastWorkers,
		BroadcastCacheTTL:        broadcastCacheTTL,
		BroadcastSeedRelays:      broadcastSeedRelays,
		BroadcastMandatoryRelays: broadcastMandatoryRelays,
	}

	return cfg
}

// ApplyToRelay applies config NIP-11 fields to a khatru Relay instance.
func ApplyToRelay(r *khatru.Relay, cfg *Config) {
	if cfg.RelayServiceURL != "" {
		r.ServiceURL = cfg.RelayServiceURL
	}
	if cfg.RelayName != "" {
		r.Info.Name = cfg.RelayName
	} else {
		r.Info.Name = "relay-agregator"
	}
	if cfg.RelayDescription != "" {
		r.Info.Description = cfg.RelayDescription
	}
	if cfg.RelayContact != "" {
		r.Info.Contact = cfg.RelayContact
	}
	// software and version are fixed
	r.Info.Software = "https://gitworkshop.dev/npub18lav8fkgt8424rxamvk8qq4xuy9n8mltjtgztv2w44hc5tt9vets0hcfsz/relay.ngit.dev/saint-michaels-mirror"
	r.Info.Version = Version
	if cfg.RelayPubKey != "" {
		r.Info.PubKey = cfg.RelayPubKey
	}
	if cfg.RelayIcon != "" {
		r.Info.Icon = cfg.RelayIcon
	}
	if cfg.RelayBanner != "" {
		r.Info.Banner = cfg.RelayBanner
	}
}
