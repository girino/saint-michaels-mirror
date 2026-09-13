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
	"html/template"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
	"github.com/girino/nostr-lib/broadcast"
	"github.com/girino/nostr-lib/eventstore/broadcaststore"
	"github.com/girino/nostr-lib/eventstore/relaystore"
	"github.com/girino/nostr-lib/fanout"
	jsonlib "github.com/girino/nostr-lib/json"
	"github.com/girino/nostr-lib/logging"
	"github.com/girino/nostr-lib/mirror"
	"github.com/girino/nostr-lib/stats"
	"github.com/nbd-wtf/go-nostr"
	nip11 "github.com/nbd-wtf/go-nostr/nip11"
	nip19 "github.com/nbd-wtf/go-nostr/nip19"
)

// Goroutine health thresholds
const (
	GoroutineYellowThreshold = 30000  // 30k goroutines = yellow health
	GoroutineRedThreshold    = 100000 // 100k goroutines = red health
)

// Health state constants
const (
	HealthGreen  = "GREEN"
	HealthYellow = "YELLOW"
	HealthRed    = "RED"
)

// getGoroutineHealthState determines the health state based on goroutine count
func getGoroutineHealthState(goroutineCount int) string {
	if goroutineCount >= GoroutineRedThreshold {
		return HealthRed
	} else if goroutineCount >= GoroutineYellowThreshold {
		return HealthYellow
	}
	return HealthGreen
}

// appStatsProvider provides runtime stats for the application
type appStatsProvider struct {
	startTime time.Time
	version   string
}

func (p *appStatsProvider) GetStatsName() string {
	return "app"
}

func (p *appStatsProvider) GetStats() jsonlib.JsonEntity {
	// Get runtime stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Get goroutine health state
	goroutineCount := runtime.NumGoroutine()
	goroutineHealthState := getGoroutineHealthState(goroutineCount)

	// Build app stats object
	appObj := jsonlib.NewJsonObject()
	appObj.Set("version", jsonlib.NewJsonValue(p.version))
	appObj.Set("uptime", jsonlib.NewJsonValue(time.Since(p.startTime).Seconds()))

	goroutineObj := jsonlib.NewJsonObject()
	goroutineObj.Set("count", jsonlib.NewJsonValue(goroutineCount))
	goroutineObj.Set("health_state", jsonlib.NewJsonValue(goroutineHealthState))
	appObj.Set("goroutines", goroutineObj)

	memoryObj := jsonlib.NewJsonObject()
	memoryObj.Set("alloc_bytes", jsonlib.NewJsonValue(m.Alloc))
	memoryObj.Set("total_alloc_bytes", jsonlib.NewJsonValue(m.TotalAlloc))
	memoryObj.Set("sys_bytes", jsonlib.NewJsonValue(m.Sys))
	memoryObj.Set("heap_alloc_bytes", jsonlib.NewJsonValue(m.HeapAlloc))
	memoryObj.Set("heap_sys_bytes", jsonlib.NewJsonValue(m.HeapSys))
	memoryObj.Set("heap_idle_bytes", jsonlib.NewJsonValue(m.HeapIdle))
	memoryObj.Set("heap_inuse_bytes", jsonlib.NewJsonValue(m.HeapInuse))
	memoryObj.Set("gc_cycles", jsonlib.NewJsonValue(m.NumGC))
	memoryObj.Set("gc_pause_ns", jsonlib.NewJsonValue(m.PauseTotalNs))
	appObj.Set("memory", memoryObj)

	gcObj := jsonlib.NewJsonObject()
	gcObj.Set("cycles", jsonlib.NewJsonValue(m.NumGC))
	gcObj.Set("pause_ns", jsonlib.NewJsonValue(m.PauseTotalNs))
	gcObj.Set("next_gc_ns", jsonlib.NewJsonValue(m.NextGC))
	appObj.Set("gc", gcObj)

	return appObj
}

func main() {
	// Track start time for uptime calculation
	startTime := time.Now()

	// use LoadConfig to read env/flags
	cfg := LoadConfig()

	// Initialize logging package from config
	// Examples:
	//   - VERBOSE=1 or VERBOSE=true: enable all verbose logging
	//   - VERBOSE=relaystore: enable verbose for relaystore module only
	//   - VERBOSE=relaystore.QueryEvents,mirror: enable specific method + module
	//   - VERBOSE=: disable all verbose logging (default)
	logging.SetVerbose(cfg.Verbose)

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
			logging.DebugMethod("main", "main", "generated new relay secret key")
		}
	}

	// Derive relay pubkey if not already set
	if sec != "" {
		if strings.HasPrefix(sec, "nsec") {
			// Decode nsec
			if _, val, err := nip19.Decode(sec); err == nil {
				if s, ok := val.(string); ok {
					// derive pubkey
					if pk, err := nostr.GetPublicKey(s); err == nil && r.Info.PubKey == "" {
						r.Info.PubKey = pk
					}
				}
			}
		} else {
			// assume it's hex
			if _, err := hex.DecodeString(sec); err == nil {
				if pk, err := nostr.GetPublicKey(sec); err == nil && r.Info.PubKey == "" {
					r.Info.PubKey = pk
				}
			}
		}
		// do not log secrets
	}

	// initialize relaystore with mandatory query relays
	var rs *relaystore.RelayStore
	if len(cfg.QueryRemotes) > 0 {
		// Query remotes are mandatory - use them
		rs = relaystore.New(cfg.QueryRemotes)
	} else {
		// No query remotes provided - fail
		logging.Fatal("no query remotes provided - relaystore requires query remotes")
	}
	if err := rs.Init(); err != nil {
		logging.Fatal("initializing relaystore: %v", err)
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

	whitelist, err := ParsePubkeyWhitelist(cfg.AllowedPubkeys)
	if err != nil {
		logging.Fatal("invalid ALLOWED_NPUBS: %v", err)
	}
	if whitelist.Enabled() {
		logging.Info("pubkey whitelist enabled: %d npub(s); NIP-42 AUTH required on connect", whitelist.Len())
		r.OnConnect = append(r.OnConnect, whitelist.RequestAuthOnConnect)
		r.RejectFilter = append(r.RejectFilter, whitelist.RejectFilter)
		r.RejectCountFilter = append(r.RejectCountFilter, whitelist.RejectCountFilter)
		r.RejectEvent = append(r.RejectEvent, whitelist.RejectEvent)
		if r.Info.Limitation == nil {
			r.Info.Limitation = &nip11.RelayLimitationDocument{}
		}
		r.Info.Limitation.AuthRequired = true
		r.Info.Limitation.RestrictedWrites = true
	}

	// Drop idle/dead sockets faster; khatru WriteJSON has no write deadline.
	r.PingPeriod = 20 * time.Second
	r.PongWait = 40 * time.Second

	// Apply custom connection and filter policies for upstream relay protection
	filterIpRateLimiter := policies.FilterIPRateLimiter(20, time.Minute, 100)
	r.RejectFilter = append(r.RejectFilter,
		// Restrictive filter rate limiting to prevent upstream overload
		func(ctx context.Context, filter nostr.Filter) (reject bool, msg string) {
			reject, msg = filterIpRateLimiter(ctx, filter)
			if reject {
				logging.Warn("filter IP rate limiter: %v, %s, from: %s", reject, msg, khatru.GetIP(ctx))
			}
			return reject, msg
		},
	)

	// Strict connection rate limiting to prevent bot abuse
	connectionRateLimiter := policies.ConnectionRateLimiter(1, time.Minute*5, 100)
	r.RejectConnection = append(r.RejectConnection,
		// Strict connection limiting to prevent bot abuse
		func(req *http.Request) (reject bool) {
			reject = connectionRateLimiter(req)
			if reject {
				logging.Warn("connection rate limiter: %v, from: %s", reject, khatru.GetIPFromRequest(req))
			}
			return reject
		},
	)

	// initialize broadcaststore if seed relays are configured
	var bs *broadcaststore.BroadcastStore
	if len(cfg.BroadcastSeedRelays) > 0 {
		// Create broadcast config
		broadcastConfig := &broadcast.Config{
			TopNRelays:       cfg.MaxPublishRelays,
			SuccessRateDecay: 0.9,
			MandatoryRelays:  cfg.BroadcastMandatoryRelays,
			WorkerCount:      cfg.BroadcastWorkers,
			CacheTTL:         5 * time.Minute,
			InitialTimeout:   7 * time.Second,
		}

		// Create broadcaststore
		bs = broadcaststore.NewBroadcastStore(broadcastConfig, 10)
		if err := bs.Init(); err != nil {
			logging.Fatal("initializing broadcaststore: %v", err)
		}
		defer bs.Close()

		// Perform discovery from seed relays
		ctx := context.Background()
		bs.GetBroadcastSystem().DiscoverFromSeeds(ctx, cfg.BroadcastSeedRelays)
		bs.GetBroadcastSystem().MarkInitialized()

		// Add mandatory relays to the manager for tracking
		if len(cfg.BroadcastMandatoryRelays) > 0 {
			logging.Info("Adding %d mandatory relays to manager for tracking...", len(cfg.BroadcastMandatoryRelays))
			bs.GetBroadcastSystem().AddMandatoryRelays(cfg.BroadcastMandatoryRelays)
		}

		logging.Info("broadcaststore initialized with %d seed relays", len(cfg.BroadcastSeedRelays))

		// Register broadcaststore stats provider
		stats.GetCollector().RegisterProvider(bs)

		// Start periodic refresh
		logging.Info("Starting periodic refresh background task...")
		go startPeriodicRefresh(ctx, cfg, bs.GetBroadcastSystem())
	}

	// hook store functions into relay
	// Use broadcaststore for SaveEvent if available, otherwise use relaystore
	if bs != nil {
		r.StoreEvent = append(r.StoreEvent, bs.SaveEvent)
		r.RejectEvent = append(r.RejectEvent, bs.RejectEvent)
	} else {
		r.StoreEvent = append(r.StoreEvent, rs.SaveEvent)
	}
	r.QueryEvents = append(r.QueryEvents, rs.QueryEvents)
	r.CountEvents = append(r.CountEvents, rs.CountEvents)

	// After whitelist + rate limiters so REQ tracking only sees allowed filters.
	hub := fanout.Attach(r, fanout.WithMaxConnections(256))
	defer hub.Close()

	mm := mirror.NewMirrorManager(cfg.QueryRemotes)
	if err := mm.Init(); err != nil {
		logging.Fatal("initializing mirror manager: %v", err)
	}
	if err := mm.StartMirroringHub(r, hub); err != nil {
		logging.Fatal("starting mirror: %v", err)
	}
	defer mm.StopMirroring()

	// register stats providers with global collector
	stats.GetCollector().RegisterProvider(rs)
	stats.GetCollector().RegisterProvider(mm)
	stats.GetCollector().RegisterProvider(hub)
	appProvider := &appStatsProvider{
		startTime: startTime,
		version:   Version,
	}
	stats.GetCollector().RegisterProvider(appProvider)
	startHealthDiagnostics()

	// expose stats endpoint using the relay's router
	mux := r.Router()
	mux.HandleFunc("/api/v1/stats", handleStatsAPI())
	// Docker healthchecks hit /api/v1/health. Collect component stats directly
	// (not GetAllStats) so a stuck broadcast manager cannot freeze the probe.
	mux.HandleFunc("/api/v1/health", handleHealthAPI(r.Info.Name, rs, mm, bs, appProvider))
	mux.HandleFunc("/api/v1/live", handleLiveAPI())

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
			logging.Error("%s template execute error: %v", pageName, err)
		}
	}

	// khatru will serve NIP-11 itself; we only expose metrics here.
	// parse templates with inheritance (base template + page templates)
	baseTplPath := "cmd/saint-michaels-mirror/templates/base.html"

	// parse main page template
	mainTplPath := "cmd/saint-michaels-mirror/templates/index.html"
	mainTpl, err := template.ParseFiles(baseTplPath, mainTplPath)
	if err != nil {
		logging.Fatal("failed to parse main template %s: %v", mainTplPath, err)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		vm := buildViewModel(false) // Main page doesn't show back link
		renderTemplate(w, mainTpl, vm, "main")
	})

	// parse stats page template
	statsTplPath := "cmd/saint-michaels-mirror/templates/stats.html"
	statsTpl, err := template.ParseFiles(baseTplPath, statsTplPath)
	if err != nil {
		logging.Fatal("failed to parse stats template %s: %v", statsTplPath, err)
	}
	mux.HandleFunc("/stats", func(w http.ResponseWriter, req *http.Request) {
		vm := buildViewModel(true) // Stats page shows back link
		renderTemplate(w, statsTpl, vm, "stats")
	})

	// parse health page template
	healthTplPath := "cmd/saint-michaels-mirror/templates/health.html"
	healthTpl, err := template.ParseFiles(baseTplPath, healthTplPath)
	if err != nil {
		logging.Fatal("failed to parse health template %s: %v", healthTplPath, err)
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
			logging.Fatal("invalid addr: %v", err)
		}

	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		logging.Fatal("invalid port: %v", err)
	}

	logging.Info("Starting %s on %s", ProjectName, cfg.Addr)
	if err := r.Start(host, port); err != nil {
		logging.Fatal("relay exited: %v", err)
	}
}

func startPeriodicRefresh(ctx context.Context, cfg *Config, broadcastSystem *broadcast.BroadcastSystem) {
	ticker := time.NewTicker(cfg.BroadcastRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logging.Debug("")
			logging.Debug("==============================================================")
			logging.Info("Starting periodic relay refresh...")
			logging.Debug("==============================================================")

			broadcastSystem.DiscoverFromSeeds(ctx, cfg.BroadcastSeedRelays)

			topRelays := broadcastSystem.GetTopRelays()
			logging.Info("Refresh complete: %d top relays from %d total relays", len(topRelays), broadcastSystem.GetRelayCount())
			logging.Debug("==============================================================")
			logging.Debug("")

		case <-ctx.Done():
			logging.Debug("Periodic refresh stopped")
			return
		}
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
