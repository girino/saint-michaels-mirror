package relaystore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"
)

// RelayStore forwards events to a set of remote nostr relays. It does not persist events locally.
type RelayStore struct {
	urls   []string
	relays map[string]*nostr.Relay
	mu     sync.RWMutex
	// publish timeout per remote
	publishTimeout time.Duration
	// verbose enables debug logging
	Verbose bool
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
	ch := make(chan *nostr.Event)
	close(ch)
	return ch, nil
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
				if r.Verbose {
					log.Printf("[relaystore][WARN] publish to %s failed: %v", u, err)
				}
				return
			}
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
