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
	nip11 "github.com/nbd-wtf/go-nostr/nip11"
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
		r.Info.Version = "0.1.0"
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

		// khatru will serve NIP-11 itself; we only expose metrics here.
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

func ensureSupportedNip11(r *khatru.Relay) {
	if r == nil || r.Info == nil {
		return
	}
	for _, v := range r.Info.SupportedNIPs {
		switch vv := v.(type) {
		case int:
			if vv == 11 {
				return
			}
		case int64:
			if int(vv) == 11 {
				return
			}
		}
	}
	r.Info.SupportedNIPs = append(r.Info.SupportedNIPs, 11)
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
