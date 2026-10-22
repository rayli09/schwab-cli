package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// countingPacer records how many times a request waited on the pacer and,
// when wrap is non-nil, delegates to a real limiter for timing tests.
type countingPacer struct {
	waits atomic.Int32
	wrap  Pacer
}

func (p *countingPacer) Wait(ctx context.Context) error {
	p.waits.Add(1)
	if p.wrap != nil {
		return p.wrap.Wait(ctx)
	}
	return nil
}

// scriptedServer replies with the given statuses in order (repeating the
// last one) and counts requests. retryAfter, when non-empty, is sent as a
// Retry-After header on 429 responses.
func scriptedServer(t *testing.T, statuses []int, retryAfter string,
	hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			n := int(hits.Add(1))
			status := statuses[len(statuses)-1]
			if n <= len(statuses) {
				status = statuses[n-1]
			}
			if status == http.StatusTooManyRequests && retryAfter != "" {
				w.Header().Set("Retry-After", retryAfter)
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{}`))
		}))
	t.Cleanup(srv.Close)
	return srv
}

func scriptedClient(srv *httptest.Server) *Client {
	c := New("KEY_PLACEHOLDER", staticTokens{"TOKEN_PLACEHOLDER"})
	c.BaseURL = srv.URL
	return c
}

func TestNewRateLimiterValidation(t *testing.T) {
	for _, rpm := range []int{0, -1, -120} {
		if _, err := NewRateLimiter(rpm); err == nil {
			t.Errorf("NewRateLimiter(%d): expected error", rpm)
		}
	}
	lim, err := NewRateLimiter(120)
	if err != nil {
		t.Fatal(err)
	}
	if lim.Burst() != 1 {
		t.Errorf("burst = %d, want 1 (smooth pacing)", lim.Burst())
	}
	// 120/minute = one token every 500ms = a limit of 2 per second.
	if lim.Limit() != rate.Limit(2) {
		t.Errorf("limit = %v, want 2/s (120/min)", lim.Limit())
	}
	if lim.Limit() != rate.Every(500*time.Millisecond) {
		t.Errorf("limit = %v, want Every(500ms)", lim.Limit())
	}
}

func TestInvalidClientRateLimitFailsFast(t *testing.T) {
	var hits atomic.Int32
	srv := scriptedServer(t, []int{200}, "", &hits)
	c := scriptedClient(srv)
	c.RateLimitPerMinute = -5
	if _, err := c.GetQuote(ctx(), "AAPL", nil); err == nil {
		t.Fatal("expected error for negative rate limit")
	} else if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error should name the rate limit: %v", err)
	}
	if hits.Load() != 0 {
		t.Errorf("no request should have been issued, got %d", hits.Load())
	}
}

func TestPacingEnforced(t *testing.T) {
	var hits atomic.Int32
	srv := scriptedServer(t, []int{200}, "", &hits)
	c := scriptedClient(srv)
	// One request every 60ms: the first goes immediately (burst 1), the
	// next three each wait a full interval, so 4 requests take >= ~180ms.
	c.Pacer = &countingPacer{wrap: rate.NewLimiter(rate.Every(60*time.Millisecond), 1)}
	start := time.Now()
	for i := 0; i < 4; i++ {
		if _, err := c.GetQuote(ctx(), "AAPL", nil); err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)
	if hits.Load() != 4 {
		t.Fatalf("hits = %d, want 4", hits.Load())
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("4 requests took %v; pacing should spread them over >= ~180ms", elapsed)
	}
	if elapsed > 10*time.Second {
		t.Errorf("4 requests took %v; pacing is far slower than configured", elapsed)
	}
	if got := c.Pacer.(*countingPacer).waits.Load(); got != 4 {
		t.Errorf("pacer waits = %d, want 4 (one per request)", got)
	}
}

func TestRetryOn429ThenSuccess(t *testing.T) {
	var hits atomic.Int32
	// Two 429s (Retry-After: 0 so the test does not really wait), then 200.
	srv := scriptedServer(t, []int{429, 429, 200}, "0", &hits)
	c := scriptedClient(srv)
	pacer := &countingPacer{}
	c.Pacer = pacer
	c.RetryBaseDelay = time.Millisecond
	resp, err := c.GetQuote(ctx(), "AAPL", nil)
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if hits.Load() != 3 {
		t.Errorf("hits = %d, want 3 (1 initial + 2 retries)", hits.Load())
	}
	if pacer.waits.Load() != 3 {
		t.Errorf("pacer waits = %d, want 3: retries must pass through "+
			"the limiter too", pacer.waits.Load())
	}
}

func TestPersistent429FailsWithRateLimitError(t *testing.T) {
	var hits atomic.Int32
	srv := scriptedServer(t, []int{429}, "0", &hits)
	c := scriptedClient(srv)
	c.Pacer = &countingPacer{}
	c.RetryBaseDelay = time.Millisecond
	_, err := c.GetQuote(ctx(), "AAPL", nil)
	if err == nil {
		t.Fatal("expected error after retries are exhausted")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "rate limit") || !strings.Contains(msg, "429") {
		t.Errorf("error should name the rate limit and the 429: %v", err)
	}
	// DefaultMaxRetries retries after the initial attempt.
	if want := int32(1 + DefaultMaxRetries); hits.Load() != want {
		t.Errorf("hits = %d, want %d", hits.Load(), want)
	}
}

func TestRetryAfterHeaderIsHonored(t *testing.T) {
	var hits atomic.Int32
	srv := scriptedServer(t, []int{429, 200}, "1", &hits)
	c := scriptedClient(srv)
	c.Pacer = &countingPacer{}
	// A tiny backoff base ensures that if Retry-After were ignored the
	// call would return almost immediately.
	c.RetryBaseDelay = time.Millisecond
	start := time.Now()
	if _, err := c.GetQuote(ctx(), "AAPL", nil); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Errorf("retry happened after %v; Retry-After: 1 should have "+
			"delayed it by ~1s", elapsed)
	}
	if hits.Load() != 2 {
		t.Errorf("hits = %d, want 2", hits.Load())
	}
}

func TestBackoffDelayDoublesWithJitter(t *testing.T) {
	base := 100 * time.Millisecond
	for attempt := 0; attempt < 3; attempt++ {
		want := base << attempt
		for i := 0; i < 50; i++ {
			d := backoffDelay(base, attempt)
			if d < want*3/4 || d > want*5/4 {
				t.Fatalf("backoffDelay(%v, %d) = %v, want within ±25%% of %v",
					base, attempt, d, want)
			}
		}
	}
}

func TestRetryAfterParsing(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		header string
		want   time.Duration
		ok     bool
	}{
		{"", 0, false},
		{"0", 0, true},
		{"5", 5 * time.Second, true},
		{"3600", MaxRetryAfter, true}, // capped
		{"-3", 0, false},
		{"garbage", 0, false},
		{now.Add(30 * time.Second).Format(http.TimeFormat), 30 * time.Second, true},
		{now.Add(-time.Minute).Format(http.TimeFormat), 0, true},
	}
	for _, tc := range cases {
		got, ok := retryAfterDelay(tc.header, now)
		if got != tc.want || ok != tc.ok {
			t.Errorf("retryAfterDelay(%q) = (%v, %v), want (%v, %v)",
				tc.header, got, ok, tc.want, tc.ok)
		}
	}
}

func TestPacerWaitRespectsContext(t *testing.T) {
	var hits atomic.Int32
	srv := scriptedServer(t, []int{200}, "", &hits)
	c := scriptedClient(srv)
	c.Pacer = rate.NewLimiter(rate.Every(time.Hour), 1)
	ctx, cancel := context.WithCancel(context.Background())
	// Spend the single burst token, then cancel while the next request
	// is parked in Wait.
	if _, err := c.GetQuote(ctx, "AAPL", nil); err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := c.GetQuote(ctx, "AAPL", nil); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if hits.Load() != 1 {
		t.Errorf("hits = %d, want 1 (second request never left the pacer)", hits.Load())
	}
}
