package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAuthorizationURL(t *testing.T) {
	c := &OAuthClient{APIKey: "KEY_PLACEHOLDER", AppSecret: "SECRET_PLACEHOLDER"}
	raw := c.AuthorizationURL("https://127.0.0.1", "STATE123")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme+"://"+u.Host+u.Path != AuthorizeEndpoint {
		t.Errorf("authorize URL base = %s, want %s", raw, AuthorizeEndpoint)
	}
	q := u.Query()
	if q.Get("client_id") != "KEY_PLACEHOLDER" {
		t.Errorf("client_id = %q", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != "https://127.0.0.1" {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q", q.Get("response_type"))
	}
	if q.Get("state") != "STATE123" {
		t.Errorf("state = %q", q.Get("state"))
	}
}

func TestParseCallbackCode(t *testing.T) {
	code, err := ParseCallbackCode(
		"https://127.0.0.1/?code=CODE42&state=S1", "S1")
	if err != nil || code != "CODE42" {
		t.Errorf("got (%q, %v), want (CODE42, nil)", code, err)
	}
	if _, err := ParseCallbackCode(
		"https://127.0.0.1/?code=CODE42&state=S1", "OTHER"); err == nil {
		t.Error("expected state mismatch error")
	}
	if _, err := ParseCallbackCode(
		"https://127.0.0.1/?error=access_denied&state=S1", "S1"); err == nil {
		t.Error("expected error for access_denied callback")
	}
}

// refreshStub serves one token response and records the request form.
func refreshStub(t *testing.T, form *url.Values, authz *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err != nil {
				t.Errorf("ParseForm: %v", err)
			}
			*form = r.Form
			*authz = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"access_token": "NEW_ACCESS",
				"refresh_token": "NEW_REFRESH",
				"token_type": "Bearer",
				"expires_in": 1800
			}`))
		}))
}

func managerAt(t *testing.T, tokenURL string, tf *TokenFile, now time.Time) *Manager {
	t.Helper()
	path := writeFixture(t, "{}")
	if err := tf.Save(path); err != nil {
		t.Fatal(err)
	}
	return &Manager{
		Path: path,
		OAuth: &OAuthClient{
			APIKey:    "KEY_PLACEHOLDER",
			AppSecret: "SECRET_PLACEHOLDER",
			TokenURL:  tokenURL,
		},
		Now: func() time.Time { return now },
	}
}

func TestManagerFreshTokenNoRefreshWhenValid(t *testing.T) {
	now := time.Unix(1700000000, 0)
	tf := &TokenFile{
		CreationTimestamp: now.Add(-time.Hour).Unix(),
		Token: Token{
			AccessToken:  "STILL_VALID",
			RefreshToken: "REFRESH_PLACEHOLDER",
			ExpiresAt:    now.Add(time.Hour).Unix(),
		},
	}
	// A zero-value TokenURL would hit the real endpoint if a refresh were
	// attempted; a valid token must not trigger a network call at all.
	m := managerAt(t, "", tf, now)
	got, err := m.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if got != "STILL_VALID" {
		t.Errorf("AccessToken = %q, want STILL_VALID", got)
	}
}

func TestManagerRefreshesExpiredAccessToken(t *testing.T) {
	now := time.Unix(1700000000, 0)
	var form url.Values
	var authz string
	srv := refreshStub(t, &form, &authz)
	defer srv.Close()

	tf := &TokenFile{
		CreationTimestamp: now.Add(-24 * time.Hour).Unix(), // grant is 1 day old
		Token: Token{
			AccessToken:  "OLD_ACCESS",
			RefreshToken: "OLD_REFRESH",
			ExpiresAt:    now.Add(-time.Minute).Unix(), // access token expired
		},
	}
	m := managerAt(t, srv.URL, tf, now)
	got, err := m.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if got != "NEW_ACCESS" {
		t.Errorf("AccessToken = %q, want NEW_ACCESS", got)
	}
	if form.Get("grant_type") != "refresh_token" {
		t.Errorf("grant_type = %q, want refresh_token", form.Get("grant_type"))
	}
	if form.Get("refresh_token") != "OLD_REFRESH" {
		t.Errorf("refresh_token form value = %q", form.Get("refresh_token"))
	}
	if !strings.HasPrefix(authz, "Basic ") {
		t.Errorf("Authorization = %q, want Basic auth", authz)
	}

	// The saved file must carry the new token but the ORIGINAL creation
	// timestamp (schwab-py preserves it so the 7-day clock keeps running).
	saved, err := LoadTokenFile(m.Path)
	if err != nil {
		t.Fatal(err)
	}
	if saved.CreationTimestamp != tf.CreationTimestamp {
		t.Errorf("creation_timestamp changed across refresh: %d -> %d",
			tf.CreationTimestamp, saved.CreationTimestamp)
	}
	if saved.Token.AccessToken != "NEW_ACCESS" {
		t.Errorf("saved access token = %q", saved.Token.AccessToken)
	}
}

func TestManagerRefreshTokenExpiredFails(t *testing.T) {
	now := time.Unix(1700000000, 0)
	tf := &TokenFile{
		CreationTimestamp: now.Add(-8 * 24 * time.Hour).Unix(), // 8 days old
		Token: Token{
			AccessToken:  "OLD_ACCESS",
			RefreshToken: "OLD_REFRESH",
			ExpiresAt:    now.Add(-time.Minute).Unix(),
		},
	}
	m := managerAt(t, "http://127.0.0.1:1/unused", tf, now)
	_, err := m.AccessToken(context.Background())
	if !errors.Is(err, ErrRefreshTokenExpired) {
		t.Fatalf("err = %v, want ErrRefreshTokenExpired", err)
	}
	if !strings.Contains(err.Error(), "auth login") {
		t.Errorf("error should direct the user to re-login: %v", err)
	}
}

func TestExchangeCode(t *testing.T) {
	var form url.Values
	var authz string
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()
			form = r.Form
			authz = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "A1",
				"refresh_token": "R1",
				"token_type":    "Bearer",
				"expires_in":    1800,
			})
		}))
	defer srv.Close()

	c := &OAuthClient{APIKey: "K", AppSecret: "S", TokenURL: srv.URL}
	tok, err := c.ExchangeCode(context.Background(), "https://127.0.0.1", "CODE9")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if tok.AccessToken != "A1" || tok.RefreshToken != "R1" {
		t.Errorf("token = %+v", tok)
	}
	if tok.ExpiresAt == 0 {
		t.Error("expires_at was not derived from expires_in")
	}
	if form.Get("grant_type") != "authorization_code" ||
		form.Get("code") != "CODE9" ||
		form.Get("redirect_uri") != "https://127.0.0.1" {
		t.Errorf("form = %v", form)
	}
	if !strings.HasPrefix(authz, "Basic ") {
		t.Errorf("Authorization = %q, want Basic", authz)
	}
}
