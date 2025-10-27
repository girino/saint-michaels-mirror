// Copyright (c) 2025 Girino Vey.
//
// This software is licensed under Girino's Anarchist License (GAL).
// See LICENSE file for full license text.
// License available at: https://license.girino.org/
//
// BroadcastStore - Nostr eventstore that broadcasts events using BroadcastSystem.
package broadcaststore

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/girino/nostr-lib/broadcast"
	jsonlib "github.com/girino/nostr-lib/json"
	"github.com/girino/nostr-lib/logging"
	"github.com/nbd-wtf/go-nostr"
)

// BroadcastStore implements eventstore.Store interface for broadcasting events
type BroadcastStore struct {
	broadcastSystem *broadcast.BroadcastSystem
	eventCache      map[string]time.Time
	cacheMu         sync.RWMutex
	cacheTTL        time.Duration
	cleanupTicker   *time.Ticker
	stopCleanup     chan struct{}
	wg              sync.WaitGroup

	// Stats tracking
	attempts               int64
	successes              int64
	failures               int64
	consecutiveFailures    int64
	maxConsecutiveFailures int64
}

// NewBroadcastStore creates a new BroadcastStore with the given configuration
func NewBroadcastStore(cfg *broadcast.Config, cacheTTL time.Duration, maxConsecutiveFailures int64) *BroadcastStore {
	bs := &BroadcastStore{
		broadcastSystem:        broadcast.NewBroadcastSystem(cfg),
		eventCache:             make(map[string]time.Time),
		cacheTTL:               cacheTTL,
		maxConsecutiveFailures: maxConsecutiveFailures,
		stopCleanup:            make(chan struct{}),
	}
	return bs
}

// Init initializes the broadcast store and starts the broadcast system
func (bs *BroadcastStore) Init() error {
	logging.DebugMethod("broadcaststore", "Init", "Initializing broadcast store")
	bs.broadcastSystem.Start()
	bs.startCacheCleanup()
	return nil
}

// Close stops the broadcast system and cleans up resources
func (bs *BroadcastStore) Close() {
	logging.DebugMethod("broadcaststore", "Close", "Closing broadcast store")
	close(bs.stopCleanup)
	bs.wg.Wait()
	bs.broadcastSystem.Stop()
}

// GetBroadcastSystem returns the underlying broadcast system
func (bs *BroadcastStore) GetBroadcastSystem() *broadcast.BroadcastSystem {
	return bs.broadcastSystem
}

// SaveEvent broadcasts an event if it hasn't been cached recently
func (bs *BroadcastStore) SaveEvent(ctx context.Context, evt *nostr.Event) error {
	// Check if event is cached
	if bs.isEventCached(evt.ID) {
		logging.DebugMethod("broadcaststore", "SaveEvent", "Event %s is cached, skipping broadcast", evt.ID)
		return nil
	}

	// Increment attempts
	atomic.AddInt64(&bs.attempts, 1)

	// Broadcast the event
	bs.broadcastSystem.BroadcastEvent(evt)

	// Cache the event
	bs.cacheEvent(evt.ID)

	// For now, we consider broadcast as successful
	// In a real implementation, we might want feedback from the broadcaster
	atomic.AddInt64(&bs.successes, 1)
	atomic.StoreInt64(&bs.consecutiveFailures, 0)

	logging.DebugMethod("broadcaststore", "SaveEvent", "Broadcast event %s", evt.ID)
	return nil
}

// QueryEvents returns an empty closed channel since we don't store events locally
func (bs *BroadcastStore) QueryEvents(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error) {
	logging.DebugMethod("broadcaststore", "QueryEvents", "QueryEvents called but returning empty channel")
	ch := make(chan *nostr.Event)
	close(ch)
	return ch, nil
}

// DeleteEvent broadcasts a delete event (kind 5)
func (bs *BroadcastStore) DeleteEvent(ctx context.Context, evt *nostr.Event) error {

	logging.DebugMethod("broadcaststore", "DeleteEvent", "DeleteEvent called for event %s", evt.ID)

	// For now, we just return nil as the event was already deleted
	// In a full implementation, we would broadcast a delete event (kind 5) here
	return nil
}

// ReplaceEvent replaces an event (atomically)
func (bs *BroadcastStore) ReplaceEvent(ctx context.Context, evt *nostr.Event) error {
	// For replaceable events, we just save the new event
	return bs.SaveEvent(ctx, evt)
}

// GetStatsName returns the name of this stats provider
func (bs *BroadcastStore) GetStatsName() string {
	return "broadcaststore"
}

// GetStats returns stats as JsonEntity
func (bs *BroadcastStore) GetStats() jsonlib.JsonEntity {
	obj := jsonlib.NewJsonObject()
	
	// Only return our local BroadcastStore-specific stats
	// The broadcaster and manager stats are already registered globally
	// and will appear at the top level in GetAllStats()
	obj.Set("attempts", jsonlib.NewJsonValue(atomic.LoadInt64(&bs.attempts)))
	obj.Set("successes", jsonlib.NewJsonValue(atomic.LoadInt64(&bs.successes)))
	obj.Set("failures", jsonlib.NewJsonValue(atomic.LoadInt64(&bs.failures)))
	obj.Set("consecutive_failures", jsonlib.NewJsonValue(atomic.LoadInt64(&bs.consecutiveFailures)))
	obj.Set("cache_size", jsonlib.NewJsonValue(int64(bs.getCacheSize())))
	
	return obj
}

// isEventCached checks if an event is in the cache
func (bs *BroadcastStore) isEventCached(eventID string) bool {
	bs.cacheMu.RLock()
	defer bs.cacheMu.RUnlock()

	timestamp, exists := bs.eventCache[eventID]
	if !exists {
		return false
	}

	// Check if cache entry is expired
	return time.Since(timestamp) < bs.cacheTTL
}

// cacheEvent adds an event to the cache
func (bs *BroadcastStore) cacheEvent(eventID string) {
	bs.cacheMu.Lock()
	defer bs.cacheMu.Unlock()
	bs.eventCache[eventID] = time.Now()
}

// getCacheSize returns the current size of the cache
func (bs *BroadcastStore) getCacheSize() int {
	bs.cacheMu.RLock()
	defer bs.cacheMu.RUnlock()
	return len(bs.eventCache)
}

// startCacheCleanup starts a goroutine to clean up expired cache entries
func (bs *BroadcastStore) startCacheCleanup() {
	bs.cleanupTicker = time.NewTicker(bs.cacheTTL)
	bs.wg.Add(1)

	go func() {
		defer bs.wg.Done()
		for {
			select {
			case <-bs.cleanupTicker.C:
				bs.cleanupCache()
			case <-bs.stopCleanup:
				return
			}
		}
	}()
}

// cleanupCache removes expired entries from the cache
func (bs *BroadcastStore) cleanupCache() {
	bs.cacheMu.Lock()
	defer bs.cacheMu.Unlock()

	now := time.Now()
	for eventID, timestamp := range bs.eventCache {
		if now.Sub(timestamp) >= bs.cacheTTL {
			delete(bs.eventCache, eventID)
		}
	}

	logging.DebugMethod("broadcaststore", "cleanupCache", "Cleaned up expired cache entries, cache size: %d", len(bs.eventCache))
}
