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
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fiatjaf/khatru"
)

// getEnvOr returns the environment variable value or a default if not set
func getEnvOr(env, defaultValue string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	return defaultValue
}

// Config holds runtime configuration coming from environment and CLI flags.
type Config struct {
	Addr         string
	QueryRemotes []string
	Verbose      string

	RelayServiceURL  string
	RelayName        string
	RelayDescription string
	RelayContact     string
	RelaySecKey      string
	RelayPubKey      string
	RelayIcon        string
	RelayBanner      string

	// Broadcast settings
	MaxPublishRelays         int
	BroadcastWorkers         int
	BroadcastCacheTTL        string
	BroadcastSeedRelays      []string
	BroadcastMandatoryRelays []string
	BroadcastRefreshInterval time.Duration
}

// LoadConfig reads environment variables and flags. Flags override env values.
func LoadConfig() *Config {
	envAddr := os.Getenv("ADDR")
	if envAddr == "" {
		envAddr = ":3337"
	}
	envQueryRemotes := os.Getenv("QUERY_REMOTES")
	envVerbose := os.Getenv("VERBOSE")

	// Basic settings
	addr := flag.String("addr", envAddr, "address to listen on (env: ADDR)")
	queryRemotes := flag.String("query-remotes", envQueryRemotes, "comma-separated list of remote relay URLs to use for queries/subscriptions (env: QUERY_REMOTES)")
	verbose := flag.String("verbose", envVerbose, "verbose logging control: '1'/'true' for all, 'relaystore' for module, 'relaystore.QueryEvents,mirror' for specific methods (env: VERBOSE)")

	// Relay identity settings
	relayServiceURL := flag.String("relay-service-url", os.Getenv("RELAY_SERVICE_URL"), "service URL for relay (env: RELAY_SERVICE_URL)")
	relayName := flag.String("relay-name", os.Getenv("RELAY_NAME"), "relay name (env: RELAY_NAME)")
	relayDescription := flag.String("relay-description", os.Getenv("RELAY_DESCRIPTION"), "relay description (env: RELAY_DESCRIPTION)")
	relayContact := flag.String("relay-contact", os.Getenv("RELAY_CONTACT"), "relay contact (env: RELAY_CONTACT)")
	relaySecKey := flag.String("relay-seckey", os.Getenv("RELAY_SECKEY"), "relay secret key (env: RELAY_SECKEY)")
	relayPubKey := flag.String("relay-pubkey", os.Getenv("RELAY_PUBKEY"), "relay public key (env: RELAY_PUBKEY)")
	relayIcon := flag.String("relay-icon", os.Getenv("RELAY_ICON"), "relay icon URL (env: RELAY_ICON)")
	relayBanner := flag.String("relay-banner", os.Getenv("RELAY_BANNER"), "relay banner URL (env: RELAY_BANNER)")

	// Broadcast settings
	envMaxPublishRelays := os.Getenv("MAX_PUBLISH_RELAYS")
	maxPublishRelaysVal := 50
	if envMaxPublishRelays != "" {
		if v, err := strconv.Atoi(envMaxPublishRelays); err == nil {
			maxPublishRelaysVal = v
		}
	}
	maxPublishRelays := flag.Int("max-publish-relays", maxPublishRelaysVal, "maximum number of top relays to use for publishing events (env: MAX_PUBLISH_RELAYS)")

	envBroadcastWorkers := os.Getenv("BROADCAST_WORKERS")
	broadcastWorkersVal := runtime.NumCPU() * 2
	if envBroadcastWorkers != "" {
		if v, err := strconv.Atoi(envBroadcastWorkers); err == nil {
			broadcastWorkersVal = v
		}
	}
	broadcastWorkers := flag.Int("broadcast-workers", broadcastWorkersVal, "number of worker goroutines for broadcasting (env: BROADCAST_WORKERS)")

	broadcastCacheTTL := flag.String("broadcast-cache-ttl", getEnvOr("BROADCAST_CACHE_TTL", "1h"), "cache TTL for broadcast events (env: BROADCAST_CACHE_TTL)")
	broadcastSeedRelays := flag.String("broadcast-seed-relays", os.Getenv("BROADCAST_SEED_RELAYS"), "comma-separated list of seed relays for broadcast discovery (env: BROADCAST_SEED_RELAYS)")
	broadcastMandatoryRelays := flag.String("broadcast-mandatory-relays", os.Getenv("BROADCAST_MANDATORY_RELAYS"), "comma-separated list of mandatory relays for broadcasting (env: BROADCAST_MANDATORY_RELAYS)")

	// Parse refresh interval
	envRefreshInterval := getEnvOr("BROADCAST_REFRESH_INTERVAL", "24h")
	refreshIntervalVal, err := time.ParseDuration(envRefreshInterval)
	if err != nil {
		// Default to 24 hours if parsing fails
		refreshIntervalVal = 24 * time.Hour
	}
	broadcastRefreshInterval := flag.Duration("broadcast-refresh-interval", refreshIntervalVal, "interval for periodic relay discovery refresh (env: BROADCAST_REFRESH_INTERVAL)")

	flag.Parse()

	qry := []string{}
	if *queryRemotes != "" {
		qry = strings.Split(*queryRemotes, ",")
	}

	// Parse broadcast relay lists
	broadcastSeedList := []string{}
	if *broadcastSeedRelays != "" {
		broadcastSeedList = strings.Split(*broadcastSeedRelays, ",")
	}

	broadcastMandatoryList := []string{}
	if *broadcastMandatoryRelays != "" {
		broadcastMandatoryList = strings.Split(*broadcastMandatoryRelays, ",")
	}

	cfg := &Config{
		Addr:         *addr,
		QueryRemotes: qry,
		Verbose:      *verbose,

		RelayServiceURL:  *relayServiceURL,
		RelayName:        *relayName,
		RelayDescription: *relayDescription,
		RelayContact:     *relayContact,
		RelaySecKey:      *relaySecKey,
		RelayPubKey:      *relayPubKey,
		RelayIcon:        *relayIcon,
		RelayBanner:      *relayBanner,

		MaxPublishRelays:         *maxPublishRelays,
		BroadcastWorkers:         *broadcastWorkers,
		BroadcastCacheTTL:        *broadcastCacheTTL,
		BroadcastSeedRelays:      broadcastSeedList,
		BroadcastMandatoryRelays: broadcastMandatoryList,
		BroadcastRefreshInterval: *broadcastRefreshInterval,
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
