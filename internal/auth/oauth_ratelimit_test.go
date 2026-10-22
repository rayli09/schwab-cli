package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// tokenStub serves a minimal valid token response and counts requests.
func tokenStub(t *testing.T, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"access_token": "A", "refresh_token": "R",
				"expires_in": 1800, "token_type": "Bearer"
			}`))
		}))
	t.Cleanup(srv.Close)
	return srv
}

func TestWaitBeforeRequestHookRuns(t *testing.T) {
	var hits, waits atomic.Int32
	srv := tokenStub(t, &hits)
	c := &OAuthClient{
		APIKey:    "K",
		AppSecret: "S",
		TokenURL:  srv.URL,
		WaitBeforeRequest: func(context.Context) error {
			waits.Add(1)
			return nil
		},
	}
	if _, err := c.Refresh(context.Background(), "REFRESH"); err != nil {
		t.Fatal(err)
	}
	if waits.Load() != 1 {
		t.Errorf("waits = %d, want 1 (hook runs before the token POST)", waits.Load())
	}
	if hits.Load() != 1 {
		t.Errorf("hits = %d, want 1", hits.Load())
	}
}

func TestWaitBeforeRequestErrorAbortsCall(t *testing.T) {
	var hits atomic.Int32
	srv := tokenStub(t, &hits)
	boom := errors.New("pacer unavailable")
	c := &OAuthClient{
		APIKey:    "K",
		AppSecret: "S",
		TokenURL:  srv.URL,
		WaitBeforeRequest: func(context.Context) error {
			return boom
		},
	}
	if _, err := c.ExchangeCode(context.Background(),
		"https://127.0.0.1", "CODE"); !errors.Is(err, boom) {
		t.Errorf("err = %v, want the hook's error", err)
	}
	if hits.Load() != 0 {
		t.Errorf("hits = %d, want 0 (no request after a hook error)", hits.Load())
	}
}
