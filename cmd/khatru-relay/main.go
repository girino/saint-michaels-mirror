package main

import (
	"flag"
	"log"
	"net"
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
	verbose := flag.Bool("verbose", false, "enable verbose/debug logging")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatalf("creating data dir: %v", err)
	}

	// choose storage: if remotes provided, use relaystore; otherwise use in-memory slicestore
	var rStore interface{}
	if *remotes != "" {
		// use relaystore
		rs := relaystore.New(strings.Split(*remotes, ","))
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
