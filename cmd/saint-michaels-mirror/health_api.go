// Copyright (c) 2026 Girino Vey.
//
// This software is licensed under Girino's Anarchist License (GAL).
// See LICENSE file for full license text.
// License available at: https://license.girino.org/
package main

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	jsonlib "github.com/girino/nostr-lib/json"
	"github.com/girino/nostr-lib/logging"
	"github.com/girino/nostr-lib/stats"
)

// Health collection timeouts. Keep these well under the Docker healthcheck
// timeout (10s) so a stuck stats provider cannot freeze the probe forever.
var (
	healthStatsTimeout   = 3 * time.Second
	statsEndpointTimeout = 5 * time.Second
)

const goroutineDumpMinInterval = time.Minute

var (
	errStatsTimeout         = errors.New("stats collection timed out")
	lastGoroutineDumpUnix   atomic.Int64
	lastLoggedHealthState   atomic.Value // string
	healthHeartbeatInterval = 30 * time.Second
)

type statsProvider interface {
	GetStats() jsonlib.JsonEntity
}

type healthSnapshot struct {
	Status                       string
	HTTPStatus                   int
	MainHealthState              string
	PublishHealthState           string
	QueryHealthState             string
	MirrorHealthState            string
	BroadcastHealthState         string
	GoroutineHealthState         string
	GoroutineCount               int
	ConsecutivePublishFailures   int64
	ConsecutiveQueryFailures     int64
	ConsecutiveMirrorFailures    int64
	ConsecutiveBroadcastFailures int64
	CollectMS                    int64
	TimedOut                     bool
	TimeoutComponent             string
	Reason                       string
}

func httpStatusForHealth(state string) (int, string) {
	switch state {
	case HealthGreen:
		return http.StatusOK, "healthy"
	case HealthYellow:
		return http.StatusOK, "degraded"
	case HealthRed:
		return http.StatusServiceUnavailable, "unhealthy"
	default:
		return http.StatusInternalServerError, "unknown"
	}
}

// worseHealth returns the worse of two GREEN/YELLOW/RED states.
func worseHealth(current, other string) string {
	if other == HealthRed || (other == HealthYellow && (current == HealthGreen || current == "")) {
		return other
	}
	return current
}

func jsonString(obj *jsonlib.JsonObject, key string) string {
	if obj == nil {
		return ""
	}
	v, ok := obj.Get(key)
	if !ok {
		return ""
	}
	val, ok := v.(*jsonlib.JsonValue)
	if !ok {
		return ""
	}
	s, _ := val.GetString()
	return s
}

func jsonInt(obj *jsonlib.JsonObject, key string) int64 {
	if obj == nil {
		return 0
	}
	v, ok := obj.Get(key)
	if !ok {
		return 0
	}
	val, ok := v.(*jsonlib.JsonValue)
	if !ok {
		return 0
	}
	n, _ := val.GetInt()
	return n
}

func asObject(entity jsonlib.JsonEntity) *jsonlib.JsonObject {
	obj, _ := entity.(*jsonlib.JsonObject)
	return obj
}

func getStatsWithTimeout(p statsProvider, timeout time.Duration) (jsonlib.JsonEntity, time.Duration, error) {
	if p == nil {
		return nil, 0, nil
	}
	start := time.Now()
	ch := make(chan jsonlib.JsonEntity, 1)
	go func() {
		ch <- p.GetStats()
	}()
	select {
	case entity := <-ch:
		return entity, time.Since(start), nil
	case <-time.After(timeout):
		return nil, time.Since(start), errStatsTimeout
	}
}

func collectHealthSnapshot(rs, mm, bs, app statsProvider) healthSnapshot {
	start := time.Now()
	snap := healthSnapshot{
		GoroutineCount: runtime.NumGoroutine(),
	}

	type named struct {
		name string
		p    statsProvider
	}
	var relayObj, mirrorObj, broadcastObj, appObj *jsonlib.JsonObject
	for _, item := range []named{
		{"relay", rs},
		{"mirror", mm},
		{"broadcaststore", bs},
		{"app", app},
	} {
		entity, _, err := getStatsWithTimeout(item.p, healthStatsTimeout)
		if err != nil {
			snap.TimedOut = true
			snap.TimeoutComponent = item.name
			snap.Reason = "stats_timeout:" + item.name
			dumpAllGoroutines(fmt.Sprintf("health stats timed out collecting %s after %s", item.name, healthStatsTimeout))
			break
		}
		switch item.name {
		case "relay":
			relayObj = asObject(entity)
		case "mirror":
			mirrorObj = asObject(entity)
		case "broadcaststore":
			broadcastObj = asObject(entity)
		case "app":
			appObj = asObject(entity)
		}
	}

	snap.MainHealthState = jsonString(relayObj, "main_health_state")
	snap.PublishHealthState = jsonString(relayObj, "publish_health_state")
	snap.QueryHealthState = jsonString(relayObj, "query_health_state")
	snap.ConsecutivePublishFailures = jsonInt(relayObj, "consecutive_publish_failures")
	snap.ConsecutiveQueryFailures = jsonInt(relayObj, "consecutive_query_failures")

	snap.MirrorHealthState = jsonString(mirrorObj, "mirror_health_state")
	snap.ConsecutiveMirrorFailures = jsonInt(mirrorObj, "consecutive_mirror_failures")
	snap.MainHealthState = worseHealth(snap.MainHealthState, snap.MirrorHealthState)

	snap.BroadcastHealthState = jsonString(broadcastObj, "health_state")
	snap.ConsecutiveBroadcastFailures = jsonInt(broadcastObj, "consecutive_failures")
	snap.MainHealthState = worseHealth(snap.MainHealthState, snap.BroadcastHealthState)

	if appObj != nil {
		if goroutinesObj, ok := appObj.Get("goroutines"); ok {
			if goroutinesVal, ok := goroutinesObj.(*jsonlib.JsonObject); ok {
				snap.GoroutineHealthState = jsonString(goroutinesVal, "health_state")
				if n := jsonInt(goroutinesVal, "count"); n > 0 {
					snap.GoroutineCount = int(n)
				}
				snap.MainHealthState = worseHealth(snap.MainHealthState, snap.GoroutineHealthState)
			}
		}
	}

	if snap.TimedOut {
		snap.MainHealthState = HealthRed
		if snap.Reason == "" {
			snap.Reason = "stats_timeout"
		}
	}

	snap.HTTPStatus, snap.Status = httpStatusForHealth(snap.MainHealthState)
	snap.CollectMS = time.Since(start).Milliseconds()
	return snap
}

func (s healthSnapshot) toJSON(service, version string) *jsonlib.JsonObject {
	health := jsonlib.NewJsonObject()
	health.Set("status", jsonlib.NewJsonValue(s.Status))
	health.Set("service", jsonlib.NewJsonValue(service))
	health.Set("version", jsonlib.NewJsonValue(version))
	health.Set("main_health_state", jsonlib.NewJsonValue(s.MainHealthState))
	health.Set("publish_health_state", jsonlib.NewJsonValue(s.PublishHealthState))
	health.Set("query_health_state", jsonlib.NewJsonValue(s.QueryHealthState))
	health.Set("mirror_health_state", jsonlib.NewJsonValue(s.MirrorHealthState))
	health.Set("broadcast_health_state", jsonlib.NewJsonValue(s.BroadcastHealthState))
	health.Set("goroutine_health_state", jsonlib.NewJsonValue(s.GoroutineHealthState))
	health.Set("goroutine_count", jsonlib.NewJsonValue(s.GoroutineCount))
	health.Set("consecutive_publish_failures", jsonlib.NewJsonValue(s.ConsecutivePublishFailures))
	health.Set("consecutive_query_failures", jsonlib.NewJsonValue(s.ConsecutiveQueryFailures))
	health.Set("consecutive_mirror_failures", jsonlib.NewJsonValue(s.ConsecutiveMirrorFailures))
	health.Set("consecutive_broadcast_failures", jsonlib.NewJsonValue(s.ConsecutiveBroadcastFailures))
	health.Set("collect_ms", jsonlib.NewJsonValue(s.CollectMS))
	if s.Reason != "" {
		health.Set("reason", jsonlib.NewJsonValue(s.Reason))
	}
	if s.TimeoutComponent != "" {
		health.Set("timeout_component", jsonlib.NewJsonValue(s.TimeoutComponent))
	}
	return health
}

func (s healthSnapshot) logLine(req *http.Request) string {
	from := ""
	ua := ""
	if req != nil {
		from = req.RemoteAddr
		ua = req.UserAgent()
	}
	return fmt.Sprintf(
		"health check: status=%s state=%s query=%s mirror=%s broadcast=%s goroutines=%s(%d) failures=p%d/q%d/m%d/b%d duration=%dms reason=%q ua=%q from=%s",
		s.Status, s.MainHealthState, s.QueryHealthState, s.MirrorHealthState, s.BroadcastHealthState,
		s.GoroutineHealthState, s.GoroutineCount,
		s.ConsecutivePublishFailures, s.ConsecutiveQueryFailures, s.ConsecutiveMirrorFailures, s.ConsecutiveBroadcastFailures,
		s.CollectMS, s.Reason, ua, from,
	)
}

func isDockerHealthcheck(req *http.Request) bool {
	if req == nil {
		return false
	}
	return strings.Contains(strings.ToLower(req.UserAgent()), "curl")
}

func logHealthSnapshot(s healthSnapshot, req *http.Request) {
	line := s.logLine(req)
	prev, _ := lastLoggedHealthState.Load().(string)
	switch {
	case s.TimedOut || s.Status == "unhealthy" || s.MainHealthState == HealthRed:
		logging.Error("%s", line)
		if s.GoroutineHealthState == HealthRed || s.GoroutineHealthState == HealthYellow {
			dumpAllGoroutines(fmt.Sprintf("goroutine health %s count=%d", s.GoroutineHealthState, s.GoroutineCount))
		}
	case s.Status != "healthy" || s.MainHealthState == HealthYellow:
		logging.Warn("%s", line)
		if s.GoroutineHealthState == HealthYellow {
			dumpAllGoroutines(fmt.Sprintf("goroutine health YELLOW count=%d", s.GoroutineCount))
		}
	case prev != "" && prev != s.MainHealthState:
		logging.Info("health state changed %s -> %s; %s", prev, s.MainHealthState, line)
	case isDockerHealthcheck(req):
		logging.Info("%s", line)
	}
	lastLoggedHealthState.Store(s.MainHealthState)
}

func dumpAllGoroutines(reason string) {
	now := time.Now().Unix()
	prev := lastGoroutineDumpUnix.Load()
	if now-prev < int64(goroutineDumpMinInterval.Seconds()) {
		logging.Error("%s (goroutine dump suppressed)", reason)
		return
	}
	if !lastGoroutineDumpUnix.CompareAndSwap(prev, now) {
		return
	}
	buf := make([]byte, 2<<20)
	n := runtime.Stack(buf, true)
	logging.Error("%s\n%s", reason, buf[:n])
}

func collectAllStatsWithTimeout(timeout time.Duration) (*jsonlib.JsonObject, time.Duration, error) {
	start := time.Now()
	ch := make(chan *jsonlib.JsonObject, 1)
	go func() {
		ch <- stats.GetCollector().GetAllStats()
	}()
	select {
	case obj := <-ch:
		return obj, time.Since(start), nil
	case <-time.After(timeout):
		dumpAllGoroutines(fmt.Sprintf("GetAllStats timed out after %s (likely lock contention in a stats provider)", timeout))
		return nil, time.Since(start), errStatsTimeout
	}
}

func startHealthDiagnostics() {
	go func() {
		heartbeat := time.NewTicker(healthHeartbeatInterval)
		defer heartbeat.Stop()
		for range heartbeat.C {
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			n := runtime.NumGoroutine()
			logging.Info("health heartbeat: goroutines=%d alloc_mb=%.1f sys_mb=%.1f gc=%d",
				n,
				float64(mem.Alloc)/1024/1024,
				float64(mem.Sys)/1024/1024,
				mem.NumGC,
			)
			if n >= GoroutineYellowThreshold {
				dumpAllGoroutines(fmt.Sprintf("heartbeat goroutine count %d", n))
			}
		}
	}()
}

func handleHealthAPI(serviceName string, rs, mm, bs, app statsProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		snap := collectHealthSnapshot(rs, mm, bs, app)
		logHealthSnapshot(snap, req)
		jsonData, err := jsonlib.MarshalIndent(snap.toJSON(serviceName, Version), "", "  ")
		if err != nil {
			http.Error(w, "failed to encode health status", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(snap.HTTPStatus)
		_, _ = w.Write(jsonData)
	}
}

func handleLiveAPI() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		obj := jsonlib.NewJsonObject()
		obj.Set("status", jsonlib.NewJsonValue("ok"))
		obj.Set("goroutines", jsonlib.NewJsonValue(runtime.NumGoroutine()))
		jsonData, err := jsonlib.MarshalIndent(obj, "", "  ")
		if err != nil {
			http.Error(w, "failed to encode live status", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jsonData)
	}
}

func handleStatsAPI() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		allStats, dur, err := collectAllStatsWithTimeout(statsEndpointTimeout)
		if err != nil {
			logging.Error("stats endpoint: %v after %s ua=%q from=%s", err, dur, req.UserAgent(), req.RemoteAddr)
			http.Error(w, "stats collection timed out", http.StatusServiceUnavailable)
			return
		}
		if dur > time.Second {
			logging.Warn("stats endpoint slow: %s ua=%q from=%s", dur, req.UserAgent(), req.RemoteAddr)
		}
		jsonData, err := jsonlib.MarshalIndent(allStats, "", "  ")
		if err != nil {
			http.Error(w, "failed to encode stats", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonData)
	}
}
