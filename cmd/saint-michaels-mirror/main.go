// Copyright (c) 2025 Girino Vey.
//
// This software is licensed under Girino's Anarchist License (GAL).
// See LICENSE file for full license text.
// License available at: https://license.girino.org/
//
// Espelho de São Miguel - A Nostr relay aggregator built on khatru.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"log"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fiatjaf/khatru"
	"github.com/girino/saint-michaels-mirror/mirror"
	"github.com/girino/saint-michaels-mirror/relaystore"
	"github.com/nbd-wtf/go-nostr"
	nip11 "github.com/nbd-wtf/go-nostr/nip11"
	nip19 "github.com/nbd-wtf/go-nostr/nip19"
)

func main() {
	// Track start time for uptime calculation
	startTime := time.Now()

	// use LoadConfig to read env/flags
	cfg := LoadConfig()

	// create a basic khatru relay instance
	r := khatru.NewRelay()

	// apply NIP-11 fields from config
	ApplyToRelay(r, cfg)

	// handle RELAY_SECKEY: accept nsec bech32 or raw hex; derive pubkey and set Info.PubKey if not provided
	sec := cfg.RelaySecKey
	if sec == "" {
		// attempt to generate a new secret if none provided
		s := nostr.GeneratePrivateKey()
		if s != "" {
			sec = s
			if cfg.Verbose {
				log.Printf("generated new relay secret key")
			}
		}
	}

	// Decode nsec to hex if needed
	decodedSec := sec
	if sec != "" {
		// try nip19 decode first
		if strings.HasPrefix(sec, "nsec") {
			if pfx, val, err := nip19.Decode(sec); err == nil && pfx == "nsec" {
				if s, ok := val.(string); ok {
					// s should be hex private key - use it directly
					decodedSec = s
					// derive pubkey
					if pk, err := nostr.GetPublicKey(s); err == nil {
						if r.Info.PubKey == "" {
							r.Info.PubKey = pk
						}
					}
				}
			}
		} else {
			// assume it's hex
			if _, err := hex.DecodeString(sec); err == nil {
				decodedSec = sec
				if pk, err := nostr.GetPublicKey(sec); err == nil {
					if r.Info.PubKey == "" {
						r.Info.PubKey = pk
					}
				}
			}
		}
		// do not log secrets
	}

	// initialize relaystore with mandatory query relays and optional publish relays
	var rs *relaystore.RelayStore
	if len(cfg.QueryRemotes) > 0 {
		// Query remotes are mandatory - use them
		rs = relaystore.New(cfg.QueryRemotes, cfg.PublishRemotes, decodedSec)
	} else {
		// No query remotes provided - fail
		log.Fatalf("no query remotes provided - relaystore requires query remotes")
	}
	if cfg.Verbose {
		rs.Verbose = true
	}
	if err := rs.Init(); err != nil {
		log.Fatalf("initializing relaystore: %v", err)
	}

	// initialize mirror manager with query remotes or fail
	var mm *mirror.MirrorManager
	if len(cfg.QueryRemotes) > 0 {
		mm = mirror.NewMirrorManager(cfg.QueryRemotes)
		if cfg.Verbose {
			mm.Verbose = true
		}
		if err := mm.Init(); err != nil {
			log.Fatalf("initializing mirror manager: %v", err)
		}
	} else {
		// No query remotes provided - fail
		log.Fatalf("no query remotes provided - mirror manager requires query remotes")
	}

	// Ensure some canonical NIP-11 fields are set on the relay Info. ApplyToRelay
	// sets most fields from config; here we only set safe defaults when empty
	// and make sure SupportedNIPs includes 11 so khatru will serve NIP-11.
	if r.Info == nil {
		r.Info = &nip11.RelayInformationDocument{}
	}
	if r.Info.Software == "" {
		r.Info.Software = "https://gitworkshop.dev/npub18lav8fkgt8424rxamvk8qq4xuy9n8mltjtgztv2w44hc5tt9vets0hcfsz/relay.ngit.dev/saint-michaels-mirror"
	}
	if r.Info.Version == "" {
		r.Info.Version = Version
	}
	// ensure SupportedNIPs contains 11, 42, and 45 (we add 45 in case a store/feature needs it)
	ensureSupportedNips(r, []int{11, 42, 45})

	// populate other NIP-11 fields from config if provided (explicitly override)
	if cfg.RelayName != "" {
		r.Info.Name = cfg.RelayName
	}
	if cfg.RelayDescription != "" {
		r.Info.Description = cfg.RelayDescription
	}
	if cfg.RelayContact != "" {
		r.Info.Contact = cfg.RelayContact
	}
	if cfg.RelayIcon != "" {
		r.Info.Icon = cfg.RelayIcon
	}
	if cfg.RelayBanner != "" {
		r.Info.Banner = cfg.RelayBanner
	}
	if cfg.RelayPubKey != "" {
		r.Info.PubKey = cfg.RelayPubKey
	}

	// If we derived a secret earlier and didn't set the pubkey via config,
	// try to set it here as a final step.
	if r.Info.PubKey == "" && sec != "" {
		if strings.HasPrefix(sec, "nsec") {
			if _, val, err := nip19.Decode(sec); err == nil {
				if s, ok := val.(string); ok {
					if pk, err := nostr.GetPublicKey(s); err == nil {
						r.Info.PubKey = pk
					}
				}
			}
		} else {
			if _, err := hex.DecodeString(sec); err == nil {
				if pk, err := nostr.GetPublicKey(sec); err == nil {
					r.Info.PubKey = pk
				}
			}
		}
	}

	// hook store functions into relay
	r.StoreEvent = append(r.StoreEvent, rs.SaveEvent)
	r.QueryEvents = append(r.QueryEvents, rs.QueryEvents)
	r.CountEvents = append(r.CountEvents, rs.CountEvents)

	// Add khatru anti-spam policies using RejectEvent hook
	r.RejectEvent = append(r.RejectEvent, func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
		// Policy 1: Reject events that are too large (>32KB)
		if len(event.Content) > 32768 {
			return true, "blocked: event content too large"
		}

		// Policy 2: Reject events with excessive tags (>100 tags)
		if len(event.Tags) > 100 {
			return true, "blocked: too many tags"
		}

		// Policy 3: Reject events with malformed timestamps
		now := nostr.Timestamp(time.Now().Unix())
		if event.CreatedAt > now+3600 { // More than 1 hour in future
			return true, "blocked: event timestamp too far in future"
		}
		if event.CreatedAt < now-31536000 { // More than 1 year in past
			return true, "blocked: event timestamp too far in past"
		}

		// Policy 4: Reject events with suspicious kind values
		if event.Kind < 0 || event.Kind > 30000 {
			return true, "blocked: invalid event kind"
		}

		// Policy 5: Reject events with empty content but many tags (potential spam)
		if len(strings.TrimSpace(event.Content)) == 0 && len(event.Tags) > 20 {
			return true, "blocked: empty content with excessive tags"
		}

		// Allow the event
		return false, ""
	})

	// Add connection-level anti-spam policies using RejectConnection hook
	r.RejectConnection = append(r.RejectConnection, func(r *http.Request) bool {
		// Basic connection validation - reject obviously invalid requests
		if r == nil || r.RemoteAddr == "" {
			return true
		}

		// Allow all other connections
		return false
	})

	// start event mirroring from query relays
	if err := mm.StartMirroring(r); err != nil {
		log.Fatalf("[mirror] failed to start mirroring: %v", err)
	}
	defer mm.StopMirroring()

	// expose stats endpoint using the relay's router
	mux := r.Router()
	mux.HandleFunc("/api/v1/stats", func(w http.ResponseWriter, req *http.Request) {
		// Get relaystore stats
		relayStats := rs.Stats()

		// Get mirror stats
		mirrorStats := mm.Stats()

		// Get runtime stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Build comprehensive stats response
		stats := map[string]interface{}{
			// Relay store stats
			"relay": relayStats,

			// Mirror stats
			"mirror": mirrorStats,

			// Application runtime stats
			"app": map[string]interface{}{
				"version":    Version,
				"uptime":     time.Since(startTime).Seconds(),
				"goroutines": runtime.NumGoroutine(),
				"memory": map[string]interface{}{
					"alloc_bytes":       m.Alloc,
					"total_alloc_bytes": m.TotalAlloc,
					"sys_bytes":         m.Sys,
					"heap_alloc_bytes":  m.HeapAlloc,
					"heap_sys_bytes":    m.HeapSys,
					"heap_idle_bytes":   m.HeapIdle,
					"heap_inuse_bytes":  m.HeapInuse,
					"gc_cycles":         m.NumGC,
					"gc_pause_ns":       m.PauseTotalNs,
				},
				"gc": map[string]interface{}{
					"cycles":     m.NumGC,
					"pause_ns":   m.PauseTotalNs,
					"next_gc_ns": m.NextGC,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			http.Error(w, "failed to encode stats", http.StatusInternalServerError)
			return
		}
	})

	// expose health endpoint for docker healthchecks
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get relaystore health status
		relayStats := rs.Stats()

		// Get mirror health status
		mirrorStats := mm.Stats()

		// Determine overall health status and HTTP status code
		var httpStatus int
		var status string

		// Use the worst health state between relay and mirror
		mainHealthState := relayStats.MainHealthState
		if mirrorStats.MirrorHealthState == "RED" || (mirrorStats.MirrorHealthState == "YELLOW" && mainHealthState == "GREEN") {
			mainHealthState = mirrorStats.MirrorHealthState
		}

		switch mainHealthState {
		case "GREEN":
			httpStatus = http.StatusOK
			status = "healthy"
		case "YELLOW":
			httpStatus = http.StatusOK // Still OK but degraded
			status = "degraded"
		case "RED":
			httpStatus = http.StatusServiceUnavailable
			status = "unhealthy"
		default:
			httpStatus = http.StatusInternalServerError
			status = "unknown"
		}

		health := map[string]interface{}{
			"status":                       status,
			"service":                      r.Info.Name,
			"version":                      Version,
			"main_health_state":            mainHealthState,
			"publish_health_state":         relayStats.PublishHealthState,
			"query_health_state":           relayStats.QueryHealthState,
			"mirror_health_state":          mirrorStats.MirrorHealthState,
			"consecutive_publish_failures": relayStats.ConsecutivePublishFailures,
			"consecutive_query_failures":   relayStats.ConsecutiveQueryFailures,
			"consecutive_mirror_failures":  mirrorStats.ConsecutiveMirrorFailures,
		}

		w.WriteHeader(httpStatus)
		if err := json.NewEncoder(w).Encode(health); err != nil {
			http.Error(w, "failed to encode health status", http.StatusInternalServerError)
			return
		}
	})

	// Define view model struct for templates
	type ViewModel struct {
		Name           string
		Description    string
		PubKey         string
		PubKeyNPub     string
		Contact        string
		ContactHref    string
		ContactIsLink  bool
		SoftwareHref   string
		SoftwareIsLink bool
		SupportedNIPs  []any
		Software       string
		Version        string
		Icon           string
		Banner         string
		ServiceURL     string
		ShowBackLink   bool
		ProjectName    string
	}

	// buildViewModel creates a view model from relay info
	buildViewModel := func(showBackLink bool) ViewModel {
		vm := ViewModel{
			Name:           r.Info.Name,
			Description:    r.Info.Description,
			PubKey:         r.Info.PubKey,
			PubKeyNPub:     "",
			Contact:        r.Info.Contact,
			ContactHref:    "",
			ContactIsLink:  false,
			SoftwareHref:   "",
			SoftwareIsLink: false,
			SupportedNIPs:  r.Info.SupportedNIPs,
			Software:       r.Info.Software,
			Version:        r.Info.Version,
			Icon:           r.Info.Icon,
			Banner:         r.Info.Banner,
			ServiceURL:     r.ServiceURL,
			ShowBackLink:   showBackLink,
			ProjectName:    ProjectName,
		}

		// compute contact link if it's an email or nostr nip19 pub/profile
		if vm.Contact == "" && vm.PubKey != "" {
			// expose pubkey as npub contact when none provided
			if npub, err := nip19.EncodePublicKey(vm.PubKey); err == nil && npub != "" {
				vm.Contact = npub
			}
		}

		// compute npub for explicit display
		if vm.PubKey != "" {
			if npub, err := nip19.EncodePublicKey(vm.PubKey); err == nil && npub != "" {
				vm.PubKeyNPub = npub
			}
		}

		if vm.Contact != "" {
			c := strings.TrimSpace(vm.Contact)
			// npub / nprofile
			if strings.HasPrefix(c, "npub") || strings.HasPrefix(c, "nprofile") {
				vm.ContactHref = "https://njump.me/" + c
				vm.ContactIsLink = true
			} else if strings.Contains(c, "@") && !strings.Contains(c, " ") {
				// treat as email
				vm.ContactHref = "mailto:" + c
				vm.ContactIsLink = true
			}
		}

		// software link detection (http/https)
		if vm.Software != "" {
			s := strings.TrimSpace(vm.Software)
			if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
				vm.SoftwareHref = s
				vm.SoftwareIsLink = true
			}
		}

		return vm
	}

	// renderTemplate is a helper function to render templates with error handling
	renderTemplate := func(w http.ResponseWriter, tpl *template.Template, vm ViewModel, pageName string) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.Execute(w, vm); err != nil {
			http.Error(w, "template render error", http.StatusInternalServerError)
			log.Printf("%s template execute error: %v", pageName, err)
		}
	}

	// khatru will serve NIP-11 itself; we only expose metrics here.
	// parse templates with inheritance (base template + page templates)
	baseTplPath := "cmd/saint-michaels-mirror/templates/base.html"

	// parse main page template
	mainTplPath := "cmd/saint-michaels-mirror/templates/index.html"
	mainTpl, err := template.ParseFiles(baseTplPath, mainTplPath)
	if err != nil {
		log.Fatalf("failed to parse main template %s: %v", mainTplPath, err)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		vm := buildViewModel(false) // Main page doesn't show back link
		renderTemplate(w, mainTpl, vm, "main")
	})

	// parse stats page template
	statsTplPath := "cmd/saint-michaels-mirror/templates/stats.html"
	statsTpl, err := template.ParseFiles(baseTplPath, statsTplPath)
	if err != nil {
		log.Fatalf("failed to parse stats template %s: %v", statsTplPath, err)
	}
	mux.HandleFunc("/stats", func(w http.ResponseWriter, req *http.Request) {
		vm := buildViewModel(true) // Stats page shows back link
		renderTemplate(w, statsTpl, vm, "stats")
	})

	// parse health page template
	healthTplPath := "cmd/saint-michaels-mirror/templates/health.html"
	healthTpl, err := template.ParseFiles(baseTplPath, healthTplPath)
	if err != nil {
		log.Fatalf("failed to parse health template %s: %v", healthTplPath, err)
	}
	mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		vm := buildViewModel(true) // Health page shows back link
		renderTemplate(w, healthTpl, vm, "health")
	})

	// serve static assets (icon/banner) from ./cmd/saint-michaels-mirror/static
	fs := http.FileServer(http.Dir("cmd/saint-michaels-mirror/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// parse addr into host and port
	host, portStr, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		// maybe user provided only a port like ":8080"
		if cfg.Addr[0] == ':' {
			host = ""
			portStr = cfg.Addr[1:]
		} else {
			log.Fatalf("invalid addr: %v", err)
		}

	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("invalid port: %v", err)
	}

	log.Printf("Starting %s on %s", ProjectName, cfg.Addr)
	if err := r.Start(host, port); err != nil {
		log.Fatalf("relay exited: %v", err)
	}
}

func ensureSupportedNips(r *khatru.Relay, nips []int) {
	if r == nil || r.Info == nil {
		return
	}
	present := map[int]bool{}
	for _, v := range r.Info.SupportedNIPs {
		switch vv := v.(type) {
		case int:
			present[vv] = true
		case int64:
			present[int(vv)] = true
		}
	}
	for _, ni := range nips {
		if !present[ni] {
			r.Info.SupportedNIPs = append(r.Info.SupportedNIPs, ni)
		}
	}
}
