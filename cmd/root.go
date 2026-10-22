// Package cmd defines the schwab-cli command tree. Command names are the
// kebab-case forms of schwab-py's Client method names; each command maps to
// exactly one schwab-py method (see README.md for the parity checklist).
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/rayli09/schwab-cli/internal/auth"
	"github.com/rayli09/schwab-cli/internal/client"
)

var (
	flagAPIKey    string
	flagAppSecret string
	flagTokenPath string
	flagTimeout   time.Duration
	flagBaseURL   string
)

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "schwab-cli",
	Short: "Command-line client for the Charles Schwab Trader API (Go port of schwab-py)",
	Long: `schwab-cli is a Go port of the Python library schwab-py. It exposes
schwab-py's Client methods as subcommands speaking to the same endpoints,
with the same parameters, defaults, and enum values, and it reads and writes
token files in schwab-py's format.

Credentials come from your Schwab developer application: pass --api-key and
--app-secret (or set SCHWAB_API_KEY / SCHWAB_APP_SECRET), then run
'schwab-cli auth login' once to create a token file. Schwab refresh tokens
expire seven days after creation, so the login must be repeated weekly.

Order-mutating commands (place-order, replace-order, cancel-order) are
dry-run by default: they print the request and send nothing. Pass --execute
to actually send them.`,
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key",
		os.Getenv("SCHWAB_API_KEY"),
		"Schwab app key (env: SCHWAB_API_KEY)")
	rootCmd.PersistentFlags().StringVar(&flagAppSecret, "app-secret",
		os.Getenv("SCHWAB_APP_SECRET"),
		"Schwab app secret (env: SCHWAB_APP_SECRET)")
	defToken := os.Getenv("SCHWAB_TOKEN_PATH")
	if defToken == "" {
		defToken = "token.json"
	}
	rootCmd.PersistentFlags().StringVar(&flagTokenPath, "token-path", defToken,
		"path to the schwab-py-format token file (env: SCHWAB_TOKEN_PATH)")
	rootCmd.PersistentFlags().DurationVar(&flagTimeout, "timeout",
		client.DefaultTimeout, "HTTP timeout for API calls")
	rootCmd.PersistentFlags().StringVar(&flagBaseURL, "base-url", "",
		"override the API base URL (for testing)")

	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(streamCmd)
	registerTraderCommands(rootCmd)
	registerMarketDataCommands(rootCmd)
}

// tokenManager builds the auth.Manager for the configured credentials.
func tokenManager() *auth.Manager {
	return &auth.Manager{
		Path: flagTokenPath,
		OAuth: &auth.OAuthClient{
			APIKey:    flagAPIKey,
			AppSecret: flagAppSecret,
		},
	}
}

// newRESTClient builds an authenticated REST client from the global flags.
func newRESTClient() *client.Client {
	c := client.New(flagAPIKey, tokenManager())
	c.Timeout = flagTimeout
	if flagBaseURL != "" {
		c.BaseURL = flagBaseURL
	}
	return c
}

// requireCredentials fails early with a helpful message when the app key or
// secret is missing.
func requireCredentials() error {
	if flagAPIKey == "" || flagAppSecret == "" {
		return fmt.Errorf("both --api-key and --app-secret are required " +
			"(or set SCHWAB_API_KEY and SCHWAB_APP_SECRET)")
	}
	return nil
}

// printResponse renders an API response the way a schwab-py user would
// inspect it: the body as pretty-printed JSON on stdout. Error statuses are
// reported on stderr and returned as errors.
func printResponse(resp *client.Response) error {
	if len(resp.Body) > 0 {
		if json.Valid(resp.Body) {
			var buf json.RawMessage = resp.Body
			out, err := json.MarshalIndent(buf, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
		} else {
			fmt.Println(string(resp.Body))
		}
	}
	if resp.IsError() {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}
	return nil
}

// runCall executes one REST call and prints its response.
func runCall(fn func(ctx context.Context) (*client.Response, error)) error {
	if err := requireCredentials(); err != nil {
		return err
	}
	resp, err := fn(context.Background())
	if err != nil {
		return err
	}
	return printResponse(resp)
}

// parseTimeFlag parses a datetime flag. Accepted layouts: RFC3339, the
// schwab-style ISO layout (2006-01-02T15:04:05Z), a bare date (2006-01-02),
// or epoch milliseconds (interpreted when the value is all digits and too
// long to be a date). Naive values are interpreted as UTC.
func parseTimeFlag(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf(
		"could not parse datetime %q (want RFC3339 or YYYY-MM-DD)", s)
}

// parseDateFlag parses a date-only flag (YYYY-MM-DD).
func parseDateFlag(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse date %q (want YYYY-MM-DD)", s)
	}
	return t.UTC(), nil
}
