package client

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// DefaultRateLimitPerMinute is the request rate Schwab publishes for the
// Trader API: 120 requests per minute per app.
const DefaultRateLimitPerMinute = 120

// DefaultMaxRetries is how many times a request that comes back with
// HTTP 429 is retried before the call fails.
const DefaultMaxRetries = 3

// DefaultRetryBaseDelay is the first backoff delay after a 429 that carries
// no Retry-After header; each subsequent retry doubles it (with jitter).
const DefaultRetryBaseDelay = time.Second

// MaxRetryAfter caps how long a server's Retry-After header can make us
// wait. Schwab's rate-limit window is one minute, so waiting longer than
// that is never necessary.
const MaxRetryAfter = time.Minute

// Pacer blocks until the caller is allowed to issue one request.
// *rate.Limiter satisfies it; tests substitute fakes.
type Pacer interface {
	Wait(ctx context.Context) error
}

// NewRateLimiter builds the shared request pacer for a per-minute budget.
// Pacing is smooth: the burst is 1, so requests are spaced evenly at
// 60/requestsPerMinute seconds and no rolling 60-second window can ever
// contain more than requestsPerMinute requests.
func NewRateLimiter(requestsPerMinute int) (*rate.Limiter, error) {
	if requestsPerMinute <= 0 {
		return nil, fmt.Errorf("invalid rate limit %d: must be a positive "+
			"number of requests per minute", requestsPerMinute)
	}
	interval := time.Minute / time.Duration(requestsPerMinute)
	return rate.NewLimiter(rate.Every(interval), 1), nil
}

// retryAfterDelay interprets a Retry-After header value (delta-seconds or
// an HTTP date), capped at MaxRetryAfter. The second return value reports
// whether the header carried a usable value.
func retryAfterDelay(headerValue string, now time.Time) (time.Duration, bool) {
	headerValue = strings.TrimSpace(headerValue)
	if headerValue == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(headerValue); err == nil {
		if secs < 0 {
			return 0, false
		}
		return min(time.Duration(secs)*time.Second, MaxRetryAfter), true
	}
	if t, err := http.ParseTime(headerValue); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0
		}
		return min(d, MaxRetryAfter), true
	}
	return 0, false
}

// backoffDelay is the wait before retry number attempt (0-based) when the
// 429 response carried no usable Retry-After: baseDelay doubled per attempt
// with ±25% jitter.
func backoffDelay(baseDelay time.Duration, attempt int) time.Duration {
	d := baseDelay << attempt
	jitter := time.Duration(float64(d) * 0.25 * (rand.Float64()*2 - 1))
	return d + jitter
}

// retryDelay decides how long to wait before the next attempt after a 429:
// the server's Retry-After when present, exponential backoff otherwise.
func retryDelay(header http.Header, attempt int, baseDelay time.Duration) time.Duration {
	if d, ok := retryAfterDelay(header.Get("Retry-After"), time.Now()); ok {
		return d
	}
	return backoffDelay(baseDelay, attempt)
}

// sleepContext waits for d, returning early if ctx is done.
func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
