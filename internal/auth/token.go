// Package auth implements the OAuth2 token lifecycle for the Schwab Trader
// API, mirroring the semantics of schwab-py's schwab.auth module.
//
// Token files written by this package are JSON-compatible with schwab-py's
// token format:
//
//	{
//	  "creation_timestamp": 1699999999,
//	  "token": {
//	    "access_token": "...",
//	    "refresh_token": "...",
//	    "token_type": "Bearer",
//	    "expires_in": 1800,
//	    "expires_at": 1700001800
//	  }
//	}
//
// so a single token file can be shared between schwab-py and schwab-cli.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// RefreshTokenLifetime is Schwab's hard limit: a refresh token (and the
// grant it belongs to) expires seven days after the token was created. This
// matches schwab-py's TokenMetadata/token_age design, where the creation
// timestamp never changes across refreshes.
const RefreshTokenLifetime = 7 * 24 * time.Hour

// AccessTokenLeeway mirrors the leeway=300 used by schwab-py when it builds
// its OAuth2 session: an access token is considered expired when it is within
// this window of its nominal expiry.
const AccessTokenLeeway = 300 * time.Second

// Token is an OAuth2 token as returned by Schwab's token endpoint. Unknown
// fields returned by the endpoint are preserved verbatim so token files
// round-trip losslessly with schwab-py/authlib.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
	Scope        string `json:"scope,omitempty"`

	// Extra holds any additional fields present in the token response so
	// they survive a load/save round-trip.
	Extra map[string]json.RawMessage `json:"-"`
}

var knownTokenFields = map[string]bool{
	"access_token":  true,
	"refresh_token": true,
	"token_type":    true,
	"expires_in":    true,
	"expires_at":    true,
	"scope":         true,
}

// UnmarshalJSON decodes the standard OAuth2 token fields and preserves any
// additional fields in Extra.
func (t *Token) UnmarshalJSON(data []byte) error {
	type alias Token
	aux := (*alias)(t)
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k := range raw {
		if knownTokenFields[k] {
			delete(raw, k)
		}
	}
	if len(raw) > 0 {
		t.Extra = raw
	}
	return nil
}

// MarshalJSON encodes the standard fields plus any Extra fields.
func (t Token) MarshalJSON() ([]byte, error) {
	type alias Token
	out, err := json.Marshal((*alias)(&t))
	if err != nil {
		return nil, err
	}
	if len(t.Extra) == 0 {
		return out, nil
	}
	merged := map[string]json.RawMessage{}
	if err := json.Unmarshal(out, &merged); err != nil {
		return nil, err
	}
	for k, v := range t.Extra {
		merged[k] = v
	}
	return json.Marshal(merged)
}

// TokenFile is the on-disk token format shared with schwab-py: the token
// wrapped in metadata carrying the creation timestamp of the original grant.
type TokenFile struct {
	CreationTimestamp int64 `json:"creation_timestamp"`
	Token             Token `json:"token"`
}

// LoadTokenFile reads and parses a schwab-py-compatible token file. It
// returns an error matching schwab-py's behavior when the file uses the
// legacy (pre-metadata) format.
func LoadTokenFile(path string) (*TokenFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Detect the legacy format (a bare token with no metadata wrapper), for
	// which schwab-py raises an error instructing the user to recreate it.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("parsing token file %s: %w", path, err)
	}
	if _, ok := probe["creation_timestamp"]; !ok {
		return nil, errors.New(
			"the token format has changed since this token was created; " +
				"please delete it and create a new one")
	}
	var tf TokenFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("parsing token file %s: %w", path, err)
	}
	return &tf, nil
}

// Save writes the token file atomically with owner-only permissions.
func (tf *TokenFile) Save(path string) error {
	data, err := json.Marshal(tf)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Age returns how long ago the token grant was created, mirroring
// schwab-py's TokenMetadata.token_age.
func (tf *TokenFile) Age(now time.Time) time.Duration {
	return now.Sub(time.Unix(tf.CreationTimestamp, 0))
}

// AccessTokenExpired reports whether the access token should be refreshed,
// using the same 300-second leeway as schwab-py's OAuth session.
func (tf *TokenFile) AccessTokenExpired(now time.Time) bool {
	if tf.Token.ExpiresAt == 0 {
		return true
	}
	return !now.Add(AccessTokenLeeway).Before(time.Unix(tf.Token.ExpiresAt, 0))
}

// RefreshTokenExpired reports whether the grant has passed Schwab's 7-day
// absolute refresh-token lifetime, measured from the original creation
// timestamp exactly as schwab-py measures token age.
func (tf *TokenFile) RefreshTokenExpired(now time.Time) bool {
	return tf.Age(now) >= RefreshTokenLifetime
}

// ErrRefreshTokenExpired is returned when a token's grant is older than
// Schwab's 7-day refresh-token lifetime and an interactive login is required.
var ErrRefreshTokenExpired = errors.New(
	"refresh token has passed Schwab's 7-day lifetime; re-run " +
		"`schwab-cli auth login` to create a new token")
