package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/rayli09/schwab-cli/internal/stream"
)

var (
	streamEquitySymbols   []string
	streamOptionSymbols   []string
	streamFuturesSymbols  []string
	streamForexSymbols    []string
	streamFOptionsSymbols []string
	streamChartSymbols    []string
	streamChartFutures    []string
	streamAccountActivity bool
	streamNYSEBook        []string
	streamNASDAQBook      []string
	streamOptionsBook     []string
	streamScreenerEquity  []string
	streamScreenerOptions []string
	streamFields          string
)

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Stream real-time data over WebSocket (schwab-py: stream.StreamClient)",
	Long: `Connect to Schwab's streaming service the way schwab-py's StreamClient
does: fetch the user preferences for the streamer connection parameters,
log in, subscribe, and print every incoming message as a JSON line until
interrupted (Ctrl-C), then log out cleanly.

At least one subscription flag is required. Symbols for level-one and chart
feeds are given verbatim (option symbols use Schwab's 21-character format;
futures symbols look like /ES). Book subscriptions take their keys verbatim
too — append _BID or _ASK to a symbol to subscribe to one side of its book.
--fields selects the field ids to receive (comma-separated); the default is
all fields, mirroring schwab-py's *_subs methods called without fields.
Screener keys are passed verbatim, exactly as schwab-py passes the symbols
argument of its screener *_subs methods.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireCredentials(); err != nil {
			return err
		}

		type sub struct {
			feedName string
			keys     []string
			extra    map[string]string
		}
		var plan []sub
		add := func(name string, keys []string) {
			if len(keys) > 0 {
				plan = append(plan, sub{feedName: name, keys: keys})
			}
		}
		add("level-one-equity", streamEquitySymbols)
		add("level-one-option", streamOptionSymbols)
		add("level-one-futures", streamFuturesSymbols)
		add("level-one-forex", streamForexSymbols)
		add("level-one-futures-options", streamFOptionsSymbols)
		add("chart-equity", streamChartSymbols)
		add("chart-futures", streamChartFutures)
		if streamAccountActivity {
			plan = append(plan, sub{feedName: "account-activity"})
		}
		add("nyse-book", streamNYSEBook)
		add("nasdaq-book", streamNASDAQBook)
		add("options-book", streamOptionsBook)
		// Screener keys go verbatim, exactly as schwab-py's screener *_subs
		// methods pass their symbols argument (any sort directive is part
		// of the key a schwab-py user supplies).
		add("screener-equity", streamScreenerEquity)
		add("screener-option", streamScreenerOptions)
		if len(plan) == 0 {
			return fmt.Errorf("at least one subscription flag is required " +
				"(e.g. --equities AAPL,MSFT)")
		}

		ctx, stop := signal.NotifyContext(context.Background(),
			os.Interrupt, syscall.SIGTERM)
		defer stop()

		sc := stream.New(newRESTClient())
		if err := sc.Login(ctx); err != nil {
			return err
		}
		defer func() {
			logoutCtx, cancel := context.WithTimeout(
				context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = sc.Logout(logoutCtx)
		}()
		for _, s := range plan {
			feed := stream.Feeds[s.feedName]
			if err := sc.SubscribeParams(ctx, feed, s.keys, streamFields, s.extra); err != nil {
				return fmt.Errorf("subscribe %s: %w", feed.Service, err)
			}
		}
		fmt.Fprintln(os.Stderr, "streaming; Ctrl-C to stop")

		return sc.RunLoop(ctx, func(raw json.RawMessage) error {
			// Pretty-printing would break the one-message-per-line
			// contract; validate and pass through compactly.
			var v any
			if err := json.Unmarshal(raw, &v); err != nil {
				fmt.Println(string(raw))
				return nil
			}
			out, err := json.Marshal(v)
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		})
	},
}

func init() {
	f := streamCmd.Flags()
	f.StringSliceVar(&streamEquitySymbols, "equities", nil,
		"subscribe to LEVELONE_EQUITIES for these symbols")
	f.StringSliceVar(&streamOptionSymbols, "options", nil,
		"subscribe to LEVELONE_OPTIONS for these 21-char option symbols")
	f.StringSliceVar(&streamFuturesSymbols, "futures", nil,
		"subscribe to LEVELONE_FUTURES for these symbols (e.g. /ES)")
	f.StringSliceVar(&streamForexSymbols, "forex", nil,
		"subscribe to LEVELONE_FOREX for these pairs (e.g. EUR/USD)")
	f.StringSliceVar(&streamFOptionsSymbols, "futures-options", nil,
		"subscribe to LEVELONE_FUTURES_OPTIONS for these symbols")
	f.StringSliceVar(&streamChartSymbols, "chart-equity", nil,
		"subscribe to CHART_EQUITY for these symbols")
	f.StringSliceVar(&streamChartFutures, "chart-futures", nil,
		"subscribe to CHART_FUTURES for these symbols (e.g. /ES)")
	f.BoolVar(&streamAccountActivity, "account-activity", false,
		"subscribe to ACCT_ACTIVITY (keyed by your streamer correl id)")
	f.StringSliceVar(&streamNYSEBook, "nyse-book", nil,
		"subscribe to NYSE_BOOK for these keys (symbol, symbol_BID, or symbol_ASK)")
	f.StringSliceVar(&streamNASDAQBook, "nasdaq-book", nil,
		"subscribe to NASDAQ_BOOK for these keys (symbol, symbol_BID, or symbol_ASK)")
	f.StringSliceVar(&streamOptionsBook, "options-book", nil,
		"subscribe to OPTIONS_BOOK for these keys (symbol, symbol_BID, or symbol_ASK)")
	f.StringSliceVar(&streamScreenerEquity, "screener-equity", nil,
		"subscribe to SCREENER_EQUITY for these keys (verbatim)")
	f.StringSliceVar(&streamScreenerOptions, "screener-option", nil,
		"subscribe to SCREENER_OPTION for these keys (verbatim)")
	f.StringVar(&streamFields, "fields", "",
		"comma-separated field ids to receive (default: all fields)")
}
