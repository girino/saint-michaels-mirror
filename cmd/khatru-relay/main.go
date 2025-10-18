package main

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/fiatjaf/eventstore/slicestore"
	"github.com/fiatjaf/khatru"
	"github.com/girino/relay-agregator/relaystore"
	"github.com/nbd-wtf/go-nostr"
	nip19 "github.com/nbd-wtf/go-nostr/nip19"
)

func main() {
	// use LoadConfig to read env/flags
	cfg := LoadConfig()

	// choose storage: if remotes provided, use relaystore; otherwise use in-memory slicestore
	var rStore interface{}
	if len(cfg.PublishRemotes) > 0 || len(cfg.QueryRemotes) > 0 {
		var rs *relaystore.RelayStore
		if len(cfg.QueryRemotes) > 0 {
			rs = relaystore.NewWithQueryRemotes(cfg.PublishRemotes, cfg.QueryRemotes)
		} else {
			rs = relaystore.New(cfg.PublishRemotes)
		}
		if cfg.Verbose {
			rs.Verbose = true
		}
		if err := rs.Init(); err != nil {
			log.Fatalf("initializing relaystore: %v", err)
		}
		rStore = rs
	} else {
		defaultRemote := "ws://localhost:10547"
		rs := relaystore.New([]string{defaultRemote})
		if cfg.Verbose {
			rs.Verbose = true
		}
		if err := rs.Init(); err != nil {
			log.Printf("warning: initializing relaystore default remote failed: %v", err)
		}
		rStore = rs
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

	// hook store functions into relay
	switch s := rStore.(type) {
	case *slicestore.SliceStore:
		r.StoreEvent = append(r.StoreEvent, s.SaveEvent)
		r.QueryEvents = append(r.QueryEvents, s.QueryEvents)
		r.CountEvents = append(r.CountEvents, s.CountEvents)
	case *relaystore.RelayStore:
		r.StoreEvent = append(r.StoreEvent, s.SaveEvent)
		r.QueryEvents = append(r.QueryEvents, s.QueryEvents)
		r.CountEvents = append(r.CountEvents, s.CountEvents)
		// expose stats endpoint using the relay's router
		mux := r.Router()
		mux.HandleFunc("/stats", func(w http.ResponseWriter, req *http.Request) {
			stats := s.Stats()
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(stats); err != nil {
				http.Error(w, "failed to encode stats", http.StatusInternalServerError)
				return
			}
		})

		// serve homepage template and NIP-11 JSON
		mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			// if Accept header requests nostr+json, serve NIP-11 JSON
			if strings.Contains(req.Header.Get("Accept"), "application/nostr+json") || strings.Contains(req.Header.Get("Accept"), "application/json") {
				w.Header().Set("Content-Type", "application/nostr+json")
				// encode r.Info as JSON
				if err := json.NewEncoder(w).Encode(r.Info); err != nil {
					http.Error(w, "failed to encode nip11", http.StatusInternalServerError)
				}
				return
			}
			// otherwise serve template
			http.ServeFile(w, req, "cmd/khatru-relay/templates/index.html")
		})

		// also serve well-known path explicitly
		mux.HandleFunc("/.well-known/nostr.json", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/nostr+json")
			if err := json.NewEncoder(w).Encode(r.Info); err != nil {
				http.Error(w, "failed to encode nip11", http.StatusInternalServerError)
			}
		})
	default:
		log.Fatalf("unsupported store type: %T", s)
	}

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
