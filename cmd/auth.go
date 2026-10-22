package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rayli09/schwab-cli/internal/auth"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage OAuth tokens (schwab-py: schwab.auth)",
	Long: `Commands for creating and inspecting the token file, mirroring
schwab-py's auth helpers:

  auth login   — client_from_manual_flow: print the authorization URL, accept
                 the pasted redirect URL, and exchange it for a token.
  auth status  — TokenMetadata.token_age and expiry state, without ever
                 printing token values.

Every other command loads the token file the way client_from_token_file /
easy_client do: the access token is refreshed automatically when expired,
and a clear error is raised once the grant passes Schwab's 7-day
refresh-token lifetime.`,
}

var (
	flagCallbackURL string
	flagReceivedURL string
)

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Create a token via the manual OAuth flow (schwab-py: client_from_manual_flow)",
	Long: `Walks through Schwab's OAuth login flow by copy-pasting URLs, exactly
like schwab-py's client_from_manual_flow:

  1. Open the printed authorization URL in a browser and log in.
  2. Allow the app access to your account.
  3. Copy the ENTIRE redirect URL from the browser address bar and paste it
     at the prompt (or pass it non-interactively with --received-url).

The resulting token is written to --token-path in schwab-py's format. The
callback URL must exactly match the one configured on your Schwab app
(only 127.0.0.1 hosts make sense for this flow; the default is
https://127.0.0.1).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireCredentials(); err != nil {
			return err
		}
		oauthClient := &auth.OAuthClient{
			APIKey:            flagAPIKey,
			AppSecret:         flagAppSecret,
			WaitBeforeRequest: paceRequest,
		}

		state, err := auth.NewState()
		if err != nil {
			return err
		}
		authURL := oauthClient.AuthorizationURL(flagCallbackURL, state)

		fmt.Fprintln(os.Stderr, "Open this URL in your browser, log in, and allow access:")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "    "+authURL)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Then paste the ENTIRE redirect URL here.")

		received := flagReceivedURL
		if received == "" {
			fmt.Fprint(os.Stderr, "Redirect URL> ")
			reader := bufio.NewReader(os.Stdin)
			line, err := reader.ReadString('\n')
			if err != nil && line == "" {
				return err
			}
			received = strings.TrimSpace(line)
		}

		code, err := auth.ParseCallbackCode(received, state)
		if err != nil {
			return err
		}
		tok, err := oauthClient.ExchangeCode(context.Background(), flagCallbackURL, code)
		if err != nil {
			return err
		}
		tf := &auth.TokenFile{
			CreationTimestamp: time.Now().Unix(),
			Token:             *tok,
		}
		if err := tf.Save(flagTokenPath); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Token written to %s. Note: Schwab refresh tokens "+
			"expire 7 days after creation; re-run this command before then.\n",
			flagTokenPath)
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show token age and expiry state (schwab-py: TokenMetadata.token_age)",
	RunE: func(cmd *cobra.Command, args []string) error {
		tf, err := auth.LoadTokenFile(flagTokenPath)
		if err != nil {
			return err
		}
		now := time.Now()
		age := tf.Age(now)
		out := map[string]any{
			"token_path":             flagTokenPath,
			"creation_timestamp":     tf.CreationTimestamp,
			"created_at":             time.Unix(tf.CreationTimestamp, 0).UTC().Format(time.RFC3339),
			"token_age_seconds":      int64(age.Seconds()),
			"access_token_expired":   tf.AccessTokenExpired(now),
			"refresh_token_expired":  tf.RefreshTokenExpired(now),
			"refresh_expires_in_sec": int64((auth.RefreshTokenLifetime - age).Seconds()),
		}
		if tf.Token.ExpiresAt > 0 {
			out["access_token_expires_at"] = time.Unix(tf.Token.ExpiresAt, 0).UTC().Format(time.RFC3339)
		}
		enc, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(enc))
		return nil
	},
}

func init() {
	authLoginCmd.Flags().StringVar(&flagCallbackURL, "callback-url",
		"https://127.0.0.1",
		"callback URL configured on your Schwab app (must match exactly)")
	authLoginCmd.Flags().StringVar(&flagReceivedURL, "received-url", "",
		"the full redirect URL from the browser (skips the interactive prompt)")
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
}
