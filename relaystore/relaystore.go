package relaystore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"
)

// RelayStore forwards events to a set of remote nostr relays. It does not persist events locally.
type RelayStore struct {
	urls   []string
	relays map[string]*nostr.Relay
	// queryUrls are the remotes used for answering queries/subscriptions
	queryUrls []string
	// pool manages connections for query remotes
	pool *nostr.SimplePool
	mu   sync.RWMutex
	// publish timeout per remote
	publishTimeout time.Duration
	// verbose enables debug logging
	Verbose bool
	// stats
	publishAttempts     int64
	publishSuccesses    int64
	publishFailures     int64
	queryRequests       int64
	queryEventsReturned int64
}

// Stats holds runtime counters exported by RelayStore
type Stats struct {
	PublishAttempts     int64 `json:"publish_attempts"`
	PublishSuccesses    int64 `json:"publish_successes"`
	PublishFailures     int64 `json:"publish_failures"`
	QueryRequests       int64 `json:"query_requests"`
	QueryEventsReturned int64 `json:"query_events_returned"`
}

// Stats returns a snapshot of the RelayStore counters
func (r *RelayStore) Stats() Stats {
	return Stats{
		PublishAttempts:     atomic.LoadInt64(&r.publishAttempts),
		PublishSuccesses:    atomic.LoadInt64(&r.publishSuccesses),
		PublishFailures:     atomic.LoadInt64(&r.publishFailures),
		QueryRequests:       atomic.LoadInt64(&r.queryRequests),
		QueryEventsReturned: atomic.LoadInt64(&r.queryEventsReturned),
	}
}

// New creates a RelayStore that will forward to the provided comma-separated URLs.
func New(urls []string) *RelayStore {
	rs := &RelayStore{
		urls:           urls,
		relays:         make(map[string]*nostr.Relay),
		publishTimeout: 7 * time.Second,
	}
	return rs
}

// NewWithQueryRemotes creates a RelayStore with separate publish remotes and query remotes.
func NewWithQueryRemotes(publish []string, query []string) *RelayStore {
	rs := &RelayStore{
		urls:           publish,
		queryUrls:      query,
		relays:         make(map[string]*nostr.Relay),
		publishTimeout: 7 * time.Second,
	}
	return rs
}

func (r *RelayStore) Init() error {
	// Attempt to connect to provided relays asynchronously (best-effort)
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	for _, u := range r.urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		go func(url string) {
			if r.Verbose {
				log.Printf("[relaystore] attempting initial connect to %s", url)
			}
			rl, err := nostr.RelayConnect(ctx, url)
			if err != nil {
				if r.Verbose {
					log.Printf("[relaystore][WARN] failed initial connect to %s: %v", url, err)
				}
				// store nothing on failure; we'll attempt reconnects later on publish
				return
			}
			r.mu.Lock()
			r.relays[url] = rl
			r.mu.Unlock()
			if r.Verbose {
				log.Printf("[relaystore] connected to %s", url)
			}
		}(u)
	}

	// setup query pool: if no queryUrls provided, use sensible defaults
	if len(r.queryUrls) == 0 {
		r.queryUrls = []string{"wss://wot.girino.org", "wss://nostr.girino.org"}
	}
	// create a SimplePool for queries
	r.pool = nostr.NewSimplePool(context.Background(), nostr.WithPenaltyBox())
	if r.Verbose {
		log.Printf("[relaystore] query remotes: %v", r.queryUrls)
	}
	return nil
}

func (r *RelayStore) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rl := range r.relays {
		_ = rl.Close()
	}
	r.relays = map[string]*nostr.Relay{}
}

// helper to ensure a relay connection exists; best-effort.
func (r *RelayStore) ensureRelay(ctx context.Context, url string) (*nostr.Relay, error) {
	r.mu.RLock()
	rl, ok := r.relays[url]
	r.mu.RUnlock()
	if ok && rl.IsConnected() {
		return rl, nil
	}
	// try to connect synchronously
	if r.Verbose {
		log.Printf("[relaystore] connecting to %s", url)
	}
	newrl, err := nostr.RelayConnect(ctx, url)
	if err != nil {
		if r.Verbose {
			log.Printf("[relaystore][ERROR] failed to connect to %s: %v", url, err)
		}
		return nil, err
	}
	r.mu.Lock()
	r.relays[url] = newrl
	r.mu.Unlock()
	if r.Verbose {
		log.Printf("[relaystore] connected to %s", url)
	}
	return newrl, nil
}

// QueryEvents returns an empty, closed channel because this store does not persist events.
func (r *RelayStore) QueryEvents(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error) {
	// if no pool available, return closed channel
	if r.pool == nil {
		ch := make(chan *nostr.Event)
		close(ch)
		return ch, nil
	}

	// increment query request counter
	atomic.AddInt64(&r.queryRequests, 1)

	// use FetchMany which ends when all relays return EOSE
	evch := r.pool.FetchMany(ctx, r.queryUrls, filter)
	out := make(chan *nostr.Event)

	go func() {
		defer close(out)
		for ie := range evch {
			// ie is a nostr.RelayEvent containing the Event pointer
			if ie.Event != nil {
				// count returned events
				atomic.AddInt64(&r.queryEventsReturned, 1)
				select {
				case out <- ie.Event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

// DeleteEvent is a no-op for relay forwarding store.
func (r *RelayStore) DeleteEvent(ctx context.Context, evt *nostr.Event) error {
	return nil
}

// SaveEvent forwards the event to all configured remotes. It returns nil if at least one remote accepted the event.
func (r *RelayStore) SaveEvent(ctx context.Context, evt *nostr.Event) error {
	// publish to all remotes concurrently and collect errors
	var wg sync.WaitGroup
	errsMu := sync.Mutex{}
	var errs []error

	// if no remotes configured, simply return nil (nothing to do)
	if len(r.urls) == 0 {
		if r.Verbose {
			log.Printf("[relaystore][WARN] no remotes configured, not forwarding event %s", evt.ID)
		}
		return nil
	}

	for _, url := range r.urls {
		url := strings.TrimSpace(url)
		if url == "" {
			continue
		}
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			// create a child context with timeout for each publish
			cctx, cancel := context.WithTimeout(ctx, r.publishTimeout)
			defer cancel()

			if r.Verbose {
				log.Printf("[relaystore][DEBUG] publishing event %s to %s", evt.ID, u)
			}

			// count attempt
			atomic.AddInt64(&r.publishAttempts, 1)

			rl, err := r.ensureRelay(cctx, u)
			if err != nil {
				errsMu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", u, err))
				errsMu.Unlock()
				if r.Verbose {
					log.Printf("[relaystore][WARN] publish to %s failed to get relay: %v", u, err)
				}
				return
			}

			if err := rl.Publish(cctx, *evt); err != nil {
				errsMu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", u, err))
				errsMu.Unlock()
				// count failure
				atomic.AddInt64(&r.publishFailures, 1)
				if r.Verbose {
					log.Printf("[relaystore][WARN] publish to %s failed: %v", u, err)
				}
				return
			}
			// count success
			atomic.AddInt64(&r.publishSuccesses, 1)
			if r.Verbose {
				log.Printf("[relaystore][DEBUG] publish to %s succeeded for event %s", u, evt.ID)
			}
		}(url)
	}

	wg.Wait()

	if len(errs) == 0 {
		return nil
	}

	// if all remotes failed, return aggregated error
	return errors.New(strings.Join(func() []string {
		ss := make([]string, len(errs))
		for i, e := range errs {
			ss[i] = e.Error()
		}
		return ss
	}(), "; "))
}

// ReplaceEvent just forwards the event (best-effort), similar to SaveEvent.
func (r *RelayStore) ReplaceEvent(ctx context.Context, evt *nostr.Event) error {
	return r.SaveEvent(ctx, evt)
}

// CountEvents returns 0 because we don't store anything.
func (r *RelayStore) CountEvents(ctx context.Context, filter nostr.Filter) (int64, error) {
	return 0, nil
}

// Ensure RelayStore implements eventstore.Store and eventstore.Counter
var _ eventstore.Store = (*RelayStore)(nil)
var _ eventstore.Counter = (*RelayStore)(nil)
