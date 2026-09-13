// Copyright (c) 2026 Girino Vey.
//
// This software is licensed under Girino's Anarchist License (GAL).
// See LICENSE file for full license text.
// License available at: https://license.girino.org/
package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jsonlib "github.com/girino/nostr-lib/json"
)

type fakeProvider struct {
	obj   *jsonlib.JsonObject
	delay time.Duration
}

func (f *fakeProvider) GetStats() jsonlib.JsonEntity {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if f == nil || f.obj == nil {
		return jsonlib.NewJsonObject()
	}
	return f.obj
}

func obj(pairs ...any) *jsonlib.JsonObject {
	o := jsonlib.NewJsonObject()
	for i := 0; i+1 < len(pairs); i += 2 {
		key, _ := pairs[i].(string)
		switch v := pairs[i+1].(type) {
		case string:
			o.Set(key, jsonlib.NewJsonValue(v))
		case int:
			o.Set(key, jsonlib.NewJsonValue(v))
		case int64:
			o.Set(key, jsonlib.NewJsonValue(v))
		default:
			o.Set(key, jsonlib.NewJsonValue(v))
		}
	}
	return o
}

func TestHTTPStatusForHealth(t *testing.T) {
	tests := []struct {
		state      string
		wantStatus int
		wantText   string
	}{
		{HealthGreen, http.StatusOK, "healthy"},
		{HealthYellow, http.StatusOK, "degraded"},
		{HealthRed, http.StatusServiceUnavailable, "unhealthy"},
		{"", http.StatusInternalServerError, "unknown"},
		{"PURPLE", http.StatusInternalServerError, "unknown"},
	}
	for _, tt := range tests {
		gotStatus, gotText := httpStatusForHealth(tt.state)
		if gotStatus != tt.wantStatus || gotText != tt.wantText {
			t.Errorf("httpStatusForHealth(%q) = %d %q, want %d %q",
				tt.state, gotStatus, gotText, tt.wantStatus, tt.wantText)
		}
	}
}

func TestWorseHealth(t *testing.T) {
	if got := worseHealth(HealthGreen, HealthYellow); got != HealthYellow {
		t.Fatalf("GREEN+YELLOW = %s, want YELLOW", got)
	}
	if got := worseHealth(HealthGreen, HealthRed); got != HealthRed {
		t.Fatalf("GREEN+RED = %s, want RED", got)
	}
	if got := worseHealth(HealthYellow, HealthRed); got != HealthRed {
		t.Fatalf("YELLOW+RED = %s, want RED", got)
	}
	if got := worseHealth(HealthYellow, HealthGreen); got != HealthYellow {
		t.Fatalf("YELLOW+GREEN = %s, want YELLOW", got)
	}
	if got := worseHealth("", HealthYellow); got != HealthYellow {
		t.Fatalf("empty+YELLOW = %s, want YELLOW", got)
	}
}

func TestCollectHealthSnapshotMergesWorstState(t *testing.T) {
	rs := &fakeProvider{obj: obj("main_health_state", HealthGreen, "query_health_state", HealthGreen, "consecutive_query_failures", 0)}
	mm := &fakeProvider{obj: obj("mirror_health_state", HealthRed, "consecutive_mirror_failures", 12)}
	bs := &fakeProvider{obj: obj("health_state", HealthGreen, "consecutive_failures", 0)}
	app := &fakeProvider{obj: func() *jsonlib.JsonObject {
		o := jsonlib.NewJsonObject()
		g := jsonlib.NewJsonObject()
		g.Set("count", jsonlib.NewJsonValue(10))
		g.Set("health_state", jsonlib.NewJsonValue(HealthGreen))
		o.Set("goroutines", g)
		return o
	}()}

	snap := collectHealthSnapshot(rs, mm, bs, app)
	if snap.MainHealthState != HealthRed {
		t.Fatalf("main state = %s, want RED (mirror)", snap.MainHealthState)
	}
	if snap.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("http status = %d, want 503", snap.HTTPStatus)
	}
	if snap.ConsecutiveMirrorFailures != 12 {
		t.Fatalf("mirror failures = %d, want 12", snap.ConsecutiveMirrorFailures)
	}
	if snap.TimedOut {
		t.Fatal("expected no timeout")
	}
}

func TestCollectHealthSnapshotTimeout(t *testing.T) {
	old := healthStatsTimeout
	healthStatsTimeout = 30 * time.Millisecond
	defer func() { healthStatsTimeout = old }()

	rs := &fakeProvider{
		obj:   obj("main_health_state", HealthGreen, "query_health_state", HealthGreen),
		delay: 200 * time.Millisecond,
	}
	snap := collectHealthSnapshot(rs, nil, nil, nil)
	if !snap.TimedOut {
		t.Fatal("expected timeout")
	}
	if snap.TimeoutComponent != "relay" {
		t.Fatalf("timeout component = %q, want relay", snap.TimeoutComponent)
	}
	if snap.MainHealthState != HealthRed || snap.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("timeout should be RED/503, got %s/%d", snap.MainHealthState, snap.HTTPStatus)
	}
	if !strings.Contains(snap.Reason, "stats_timeout") {
		t.Fatalf("reason = %q, want stats_timeout", snap.Reason)
	}
}

func TestHandleHealthAPI(t *testing.T) {
	rs := &fakeProvider{obj: obj("main_health_state", HealthGreen, "query_health_state", HealthGreen)}
	mm := &fakeProvider{obj: obj("mirror_health_state", HealthGreen)}
	handler := handleHealthAPI("test-relay", rs, mm, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("User-Agent", "curl/7.88.1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("json: %v body=%s", err, body)
	}
	if payload["status"] != "healthy" {
		t.Fatalf("status field = %v, want healthy", payload["status"])
	}
	if _, ok := payload["collect_ms"]; !ok {
		t.Fatal("missing collect_ms")
	}
	if _, ok := payload["goroutine_count"]; !ok {
		t.Fatal("missing goroutine_count")
	}
}

func TestCollectHealthSnapshotTypedNilProvider(t *testing.T) {
	var rs *droppingMirror
	var bs *droppingMirror
	mm := &droppingMirror{hub: newClientHub()}
	app := &appStatsProvider{startTime: time.Now(), version: "test"}
	snap := collectHealthSnapshot(rs, mm, bs, app)
	if snap.HTTPStatus == 0 {
		t.Fatal("typed-nil stats providers must not panic or skip HTTP status")
	}
	handler := handleHealthAPI("test-relay", rs, mm, bs, app)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusServiceUnavailable && rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.Bytes())
	}
}

func TestHandleLiveAPI(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/live", nil)
	rec := httptest.NewRecorder()
	handleLiveAPI().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestIsDockerHealthcheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "curl/8.0.0")
	if !isDockerHealthcheck(req) {
		t.Fatal("curl UA should be treated as docker healthcheck")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	if isDockerHealthcheck(req) {
		t.Fatal("browser UA should not be treated as docker healthcheck")
	}
}
