package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/fiatjaf/eventstore/slicestore"
	"github.com/fiatjaf/khatru"
	"github.com/girino/relay-agregator/relaystore"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	dataDir := flag.String("data", "./data", "path to store data")
	remotes := flag.String("remotes", "", "comma-separated list of remote relay URLs to forward events to")
	queryRemotes := flag.String("query-remotes", "", "comma-separated list of remote relay URLs to use for queries/subscriptions")
	verbose := flag.Bool("verbose", false, "enable verbose/debug logging")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatalf("creating data dir: %v", err)
	}

	// choose storage: if remotes provided, use relaystore; otherwise use in-memory slicestore
	var rStore interface{}
	if *remotes != "" || *queryRemotes != "" {
		// use relaystore, allow separate query remotes
		pub := []string{}
		qry := []string{}
		if *remotes != "" {
			pub = strings.Split(*remotes, ",")
		}
		if *queryRemotes != "" {
			qry = strings.Split(*queryRemotes, ",")
		}
		var rs *relaystore.RelayStore
		if len(qry) > 0 {
			rs = relaystore.NewWithQueryRemotes(pub, qry)
		} else {
			rs = relaystore.New(pub)
		}
		if *verbose {
			rs.Verbose = true
		}
		if err := rs.Init(); err != nil {
			log.Fatalf("initializing relaystore: %v", err)
		}
		rStore = rs
	} else {
		// default to a test remote that can be used locally
		defaultRemote := "ws://localhost:10547"
		rs := relaystore.New([]string{defaultRemote})
		if *verbose {
			rs.Verbose = true
		}
		if err := rs.Init(); err != nil {
			log.Printf("warning: initializing relaystore default remote failed: %v", err)
		}
		rStore = rs
	}

	// create a basic khatru relay
	r := khatru.NewRelay()

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
	default:
		log.Fatalf("unsupported store type: %T", s)
	}

	// parse addr into host and port
	host, portStr, err := net.SplitHostPort(*addr)
	if err != nil {
		// maybe user provided only a port like ":8080"
		if (*addr)[0] == ':' {
			host = ""
			portStr = (*addr)[1:]
		} else {
			log.Fatalf("invalid addr: %v", err)
		}
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("invalid port: %v", err)
	}

	log.Printf("Starting khatru relay on %s", *addr)
	if err := r.Start(host, port); err != nil {
		log.Fatalf("relay exited: %v", err)
	}
}
