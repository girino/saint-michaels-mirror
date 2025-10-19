package main

import (
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
	"github.com/girino/relay-agregator/relaystore"
	"github.com/nbd-wtf/go-nostr"
	nip11 "github.com/nbd-wtf/go-nostr/nip11"
	nip19 "github.com/nbd-wtf/go-nostr/nip19"
)

func main() {
	// Track start time for uptime calculation
	startTime := time.Now()

	// use LoadConfig to read env/flags
	cfg := LoadConfig()

	// initialize relaystore with provided remotes or default
	var rs *relaystore.RelayStore
	if len(cfg.PublishRemotes) > 0 || len(cfg.QueryRemotes) > 0 {
		if len(cfg.QueryRemotes) > 0 {
			rs = relaystore.NewWithQueryRemotes(cfg.PublishRemotes, cfg.QueryRemotes)
		} else {
			rs = relaystore.New(cfg.PublishRemotes)
		}
	} else {
		defaultRemote := "ws://localhost:10547"
		rs = relaystore.New([]string{defaultRemote})
	}
	if cfg.Verbose {
		rs.Verbose = true
	}
	if err := rs.Init(); err != nil {
		log.Fatalf("initializing relaystore: %v", err)
	}

	// create a basic khatru relay
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
	if sec != "" {
		// try nip19 decode first
		if strings.HasPrefix(sec, "nsec") {
			if pfx, val, err := nip19.Decode(sec); err == nil && pfx == "nsec" {
				if s, ok := val.(string); ok {
					// s should be hex private key
					// try hex decode
					if _, err := hex.DecodeString(s); err == nil {
						// derive pubkey
						if pk, err := nostr.GetPublicKey(s); err == nil {
							if r.Info.PubKey == "" {
								r.Info.PubKey = pk
							}
						}
					}
				}
			}
		} else {
			// assume it's hex
			if _, err := hex.DecodeString(sec); err == nil {
				if pk, err := nostr.GetPublicKey(sec); err == nil {
					if r.Info.PubKey == "" {
						r.Info.PubKey = pk
					}
				}
			}
		}
		// do not log secrets
	}

	// Ensure some canonical NIP-11 fields are set on the relay Info. ApplyToRelay
	// sets most fields from config; here we only set safe defaults when empty
	// and make sure SupportedNIPs includes 11 so khatru will serve NIP-11.
	if r.Info == nil {
		r.Info = &nip11.RelayInformationDocument{}
	}
	if r.Info.Software == "" {
		r.Info.Software = "https://github.com/girino/relay-agregator"
	}
	if r.Info.Version == "" {
		r.Info.Version = Version
	}
	// ensure SupportedNIPs contains 11 and 45 (we add 45 in case a store/feature needs it)
	ensureSupportedNips(r, []int{11, 45})

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

	// expose stats endpoint using the relay's router
	mux := r.Router()
	mux.HandleFunc("/stats", func(w http.ResponseWriter, req *http.Request) {
		// Get relaystore stats
		relayStats := rs.Stats()

		// Get runtime stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Build comprehensive stats response
		stats := map[string]interface{}{
			// Relay store stats
			"relay": relayStats,

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
	mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		health := map[string]interface{}{
			"status":  "healthy",
			"service": "khatru-relay",
			"version": Version,
		}
		if err := json.NewEncoder(w).Encode(health); err != nil {
			http.Error(w, "failed to encode health status", http.StatusInternalServerError)
			return
		}
	})

	// khatru will serve NIP-11 itself; we only expose metrics here.
	// parse the HTML template once and serve it with r.Info as data
	tplPath := "cmd/khatru-relay/templates/index.html"
	tpl, err := template.ParseFiles(tplPath)
	if err != nil {
		log.Fatalf("failed to parse template %s: %v", tplPath, err)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// build a minimal view model expected by the template
		vm := struct {
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
		}{
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

		if err := tpl.Execute(w, vm); err != nil {
			http.Error(w, "template render error", http.StatusInternalServerError)
			log.Printf("template execute error: %v", err)
		}
	})

	// serve static assets (icon/banner) from ./cmd/khatru-relay/static
	fs := http.FileServer(http.Dir("cmd/khatru-relay/static"))
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

	log.Printf("Starting khatru relay on %s", cfg.Addr)
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
