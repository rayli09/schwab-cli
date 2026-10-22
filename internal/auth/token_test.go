package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// schwab-py writes token files as {"creation_timestamp": <unix>, "token":
// {...}} via its update_token_metadata wrapper. This fixture mirrors that
// shape exactly, including one field Schwab's endpoint returns that this
// package does not model, to prove unknown fields survive a round-trip.
const schwabPyFixture = `{
  "creation_timestamp": 1700000000,
  "token": {
    "access_token": "ACCESS_PLACEHOLDER",
    "refresh_token": "REFRESH_PLACEHOLDER",
    "token_type": "Bearer",
    "expires_in": 1800,
    "expires_at": 1700001800,
    "scope": "api",
    "id_token": "ID_TOKEN_PLACEHOLDER"
  }
}`

func writeFixture(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "token.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTokenFileRoundTrip(t *testing.T) {
	path := writeFixture(t, schwabPyFixture)
	tf, err := LoadTokenFile(path)
	if err != nil {
		t.Fatalf("LoadTokenFile: %v", err)
	}
	if tf.CreationTimestamp != 1700000000 {
		t.Errorf("creation_timestamp = %d, want 1700000000", tf.CreationTimestamp)
	}
	if tf.Token.AccessToken != "ACCESS_PLACEHOLDER" ||
		tf.Token.RefreshToken != "REFRESH_PLACEHOLDER" ||
		tf.Token.ExpiresAt != 1700001800 {
		t.Errorf("token fields not parsed: %+v", tf.Token)
	}
	if _, ok := tf.Token.Extra["id_token"]; !ok {
		t.Errorf("unknown field id_token was dropped on load")
	}

	out := filepath.Join(t.TempDir(), "out.json")
	if err := tf.Save(out); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"creation_timestamp":1700000000`,
		`"access_token":"ACCESS_PLACEHOLDER"`,
		`"refresh_token":"REFRESH_PLACEHOLDER"`,
		`"expires_at":1700001800`,
		`"id_token":"ID_TOKEN_PLACEHOLDER"`,
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("saved token file missing %s\ngot: %s", want, data)
		}
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("saved token file permissions = %o, want 600", perm)
	}
}

func TestLoadTokenFileLegacyFormat(t *testing.T) {
	// A bare token with no creation_timestamp wrapper: schwab-py raises an
	// error telling the user to recreate the token, and so do we.
	path := writeFixture(t, `{"access_token": "X", "refresh_token": "Y"}`)
	if _, err := LoadTokenFile(path); err == nil {
		t.Fatal("expected error for legacy token format")
	} else if !strings.Contains(err.Error(), "create a new one") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAccessTokenExpired(t *testing.T) {
	now := time.Unix(1700000000, 0)
	cases := []struct {
		name      string
		expiresAt int64
		want      bool
	}{
		{"valid", now.Add(time.Hour).Unix(), false},
		{"inside 300s leeway", now.Add(299 * time.Second).Unix(), true},
		{"at leeway boundary", now.Add(300 * time.Second).Unix(), true},
		{"past expiry", now.Add(-time.Minute).Unix(), true},
		{"no expiry recorded", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tf := &TokenFile{Token: Token{ExpiresAt: tc.expiresAt}}
			if got := tf.AccessTokenExpired(now); got != tc.want {
				t.Errorf("AccessTokenExpired = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRefreshTokenExpired(t *testing.T) {
	created := time.Unix(1700000000, 0)
	tf := &TokenFile{CreationTimestamp: created.Unix()}
	cases := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"fresh", created.Add(time.Hour), false},
		{"just under 7 days", created.Add(7*24*time.Hour - time.Second), false},
		{"exactly 7 days", created.Add(7 * 24 * time.Hour), true},
		{"well past 7 days", created.Add(30 * 24 * time.Hour), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tf.RefreshTokenExpired(tc.now); got != tc.want {
				t.Errorf("RefreshTokenExpired = %v, want %v", got, tc.want)
			}
		})
	}
}
