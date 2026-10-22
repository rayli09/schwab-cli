package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rayli09/schwab-cli/internal/client"
)

// The int/float/bool flag helpers convert cobra flags into the pointer-typed
// optional parameters of the client, honoring flag presence (Changed) the
// way schwab-py honors "argument not passed".

func intFlag(cmd *cobra.Command, name string) (*int, error) {
	if !cmd.Flags().Changed(name) {
		return nil, nil
	}
	v, err := cmd.Flags().GetInt(name)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func floatFlag(cmd *cobra.Command, name string) (*float64, error) {
	if !cmd.Flags().Changed(name) {
		return nil, nil
	}
	v, err := cmd.Flags().GetFloat64(name)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func boolFlag(cmd *cobra.Command, name string) (*bool, error) {
	if !cmd.Flags().Changed(name) {
		return nil, nil
	}
	v, err := cmd.Flags().GetBool(name)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func quoteFieldList(vals []string) ([]client.QuoteFields, error) {
	return parseEnumList(vals, "fields", func(s string) (client.QuoteFields, error) {
		f := client.QuoteFields(s)
		if !f.Valid() {
			return "", fmt.Errorf("invalid quote field %q", s)
		}
		return f, nil
	})
}

func registerMarketDataCommands(root *cobra.Command) {
	////////////////////////////////////////////////////////////////////////
	// Quotes

	var quoteFields []string
	getQuoteCmd := &cobra.Command{
		Use:   "get-quote <symbol>",
		Short: "Quote for one symbol (schwab-py: get_quote)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fields, err := quoteFieldList(quoteFields)
			if err != nil {
				return err
			}
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetQuote(ctx, args[0], fields)
			})
		},
	}
	getQuoteCmd.Flags().StringSliceVar(&quoteFields, "fields", nil,
		"quote fields: quote, fundamental, extended, reference, regular")
	root.AddCommand(getQuoteCmd)

	var quotesFields []string
	var quotesIndicative bool
	getQuotesCmd := &cobra.Command{
		Use:   "get-quotes <symbol> [symbol...]",
		Short: "Quotes for one or more symbols (schwab-py: get_quotes)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fields, err := quoteFieldList(quotesFields)
			if err != nil {
				return err
			}
			indicative, err := boolFlag(cmd, "indicative")
			if err != nil {
				return err
			}
			_ = quotesIndicative
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetQuotes(ctx, args, fields, indicative)
			})
		},
	}
	getQuotesCmd.Flags().StringSliceVar(&quotesFields, "fields", nil,
		"quote fields: quote, fundamental, extended, reference, regular")
	getQuotesCmd.Flags().BoolVar(&quotesIndicative, "indicative", false,
		"request indicative quotes")
	root.AddCommand(getQuotesCmd)

	////////////////////////////////////////////////////////////////////////
	// Option chain

	oc := &cobra.Command{
		Use:   "get-option-chain <symbol>",
		Short: "Option chain for a symbol (schwab-py: get_option_chain)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := buildOptionChainQuery(cmd)
			if err != nil {
				return err
			}
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetOptionChain(ctx, args[0], q)
			})
		},
	}
	oc.Flags().String("contract-type", "", "CALL, PUT, or ALL")
	oc.Flags().Int("strike-count", 0, "number of strikes above/below at-the-money")
	oc.Flags().Bool("include-underlying-quote", false, "include the underlying quote")
	oc.Flags().String("strategy", "", "strategy chain type (SINGLE, ANALYTICAL, COVERED, VERTICAL, ...)")
	oc.Flags().Float64("interval", 0, "strike interval for spread strategy chains")
	oc.Flags().Float64("strike", 0, "return options only at this strike")
	oc.Flags().String("strike-range", "", "ITM, NTM, OTM, SAK, SBK, SNK, or ALL")
	oc.Flags().String("from-date", "", "only expirations after this date (YYYY-MM-DD)")
	oc.Flags().String("to-date", "", "only expirations before this date (YYYY-MM-DD)")
	oc.Flags().Float64("volatility", 0, "volatility for ANALYTICAL chains")
	oc.Flags().Float64("underlying-price", 0, "underlying price for ANALYTICAL chains")
	oc.Flags().Float64("interest-rate", 0, "interest rate for ANALYTICAL chains")
	oc.Flags().Int("days-to-expiration", 0, "days to expiration for ANALYTICAL chains")
	oc.Flags().String("exp-month", "", "expiration month (JAN..DEC or ALL)")
	oc.Flags().String("option-type", "", "S (standard), NS (non-standard), or ALL")
	oc.Flags().String("entitlement", "", "client entitlement (PP, NP, PN)")
	root.AddCommand(oc)

	root.AddCommand(&cobra.Command{
		Use:   "get-option-expiration-chain <symbol>",
		Short: "Option expiration chain for a symbol (schwab-py: get_option_expiration_chain)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetOptionExpirationChain(ctx, args[0])
			})
		},
	})

	////////////////////////////////////////////////////////////////////////
	// Price history

	ph := &cobra.Command{
		Use:   "get-price-history <symbol>",
		Short: "Price history for a symbol (schwab-py: get_price_history)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := buildPriceHistoryQuery(cmd)
			if err != nil {
				return err
			}
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetPriceHistory(ctx, args[0], q)
			})
		},
	}
	ph.Flags().String("period-type", "", "day, month, year, or ytd")
	ph.Flags().Int("period", 0, "number of periods")
	ph.Flags().String("frequency-type", "", "minute, daily, weekly, or monthly")
	ph.Flags().Int("frequency", 0, "frequency within each candle")
	ph.Flags().String("start-datetime", "", "start datetime (RFC3339 or YYYY-MM-DD)")
	ph.Flags().String("end-datetime", "", "end datetime (RFC3339 or YYYY-MM-DD)")
	ph.Flags().Bool("need-extended-hours-data", false, "include extended hours data")
	ph.Flags().Bool("need-previous-close", false, "include previous close")
	root.AddCommand(ph)

	// The get_price_history_every_* wrappers share one flag set and differ
	// only in the client method they call.
	type variantDef struct {
		name  string
		short string
		call  func(c *client.Client, ctx context.Context, symbol string,
			start, end *time.Time, ext, prev *bool) (*client.Response, error)
	}
	defs := []variantDef{
		{"get-price-history-every-minute", "Per-minute price history (schwab-py: get_price_history_every_minute)",
			func(c *client.Client, ctx context.Context, s string, start, end *time.Time, ext, prev *bool) (*client.Response, error) {
				return c.GetPriceHistoryEveryMinute(ctx, s, start, end, ext, prev)
			}},
		{"get-price-history-every-five-minutes", "Per-five-minutes price history (schwab-py: get_price_history_every_five_minutes)",
			func(c *client.Client, ctx context.Context, s string, start, end *time.Time, ext, prev *bool) (*client.Response, error) {
				return c.GetPriceHistoryEveryFiveMinutes(ctx, s, start, end, ext, prev)
			}},
		{"get-price-history-every-ten-minutes", "Per-ten-minutes price history (schwab-py: get_price_history_every_ten_minutes)",
			func(c *client.Client, ctx context.Context, s string, start, end *time.Time, ext, prev *bool) (*client.Response, error) {
				return c.GetPriceHistoryEveryTenMinutes(ctx, s, start, end, ext, prev)
			}},
		{"get-price-history-every-fifteen-minutes", "Per-fifteen-minutes price history (schwab-py: get_price_history_every_fifteen_minutes)",
			func(c *client.Client, ctx context.Context, s string, start, end *time.Time, ext, prev *bool) (*client.Response, error) {
				return c.GetPriceHistoryEveryFifteenMinutes(ctx, s, start, end, ext, prev)
			}},
		{"get-price-history-every-thirty-minutes", "Per-thirty-minutes price history (schwab-py: get_price_history_every_thirty_minutes)",
			func(c *client.Client, ctx context.Context, s string, start, end *time.Time, ext, prev *bool) (*client.Response, error) {
				return c.GetPriceHistoryEveryThirtyMinutes(ctx, s, start, end, ext, prev)
			}},
		{"get-price-history-every-day", "Daily price history (schwab-py: get_price_history_every_day)",
			func(c *client.Client, ctx context.Context, s string, start, end *time.Time, ext, prev *bool) (*client.Response, error) {
				return c.GetPriceHistoryEveryDay(ctx, s, start, end, ext, prev)
			}},
		{"get-price-history-every-week", "Weekly price history (schwab-py: get_price_history_every_week)",
			func(c *client.Client, ctx context.Context, s string, start, end *time.Time, ext, prev *bool) (*client.Response, error) {
				return c.GetPriceHistoryEveryWeek(ctx, s, start, end, ext, prev)
			}},
	}
	for _, def := range defs {
		def := def
		var startS, endS string
		vc := &cobra.Command{
			Use:   def.name + " <symbol>",
			Short: def.short,
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				var start, end *time.Time
				if startS != "" {
					t, err := parseTimeFlag(startS)
					if err != nil {
						return err
					}
					start = &t
				}
				if endS != "" {
					t, err := parseTimeFlag(endS)
					if err != nil {
						return err
					}
					end = &t
				}
				extP, err := boolFlag(cmd, "need-extended-hours-data")
				if err != nil {
					return err
				}
				prevP, err := boolFlag(cmd, "need-previous-close")
				if err != nil {
					return err
				}
				c := newRESTClient()
				return runCall(func(ctx context.Context) (*client.Response, error) {
					return def.call(c, ctx, args[0], start, end, extP, prevP)
				})
			},
		}
		vc.Flags().StringVar(&startS, "start-datetime", "", "start datetime (RFC3339 or YYYY-MM-DD)")
		vc.Flags().StringVar(&endS, "end-datetime", "", "end datetime (RFC3339 or YYYY-MM-DD)")
		vc.Flags().Bool("need-extended-hours-data", false, "include extended hours data")
		vc.Flags().Bool("need-previous-close", false, "include previous close")
		root.AddCommand(vc)
	}

	////////////////////////////////////////////////////////////////////////
	// Movers

	var moversSort string
	moversCmd := &cobra.Command{
		Use:   "get-movers <index>",
		Short: "Top movers for an index (schwab-py: get_movers)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index := client.MoversIndex(args[0])
			if !index.Valid() {
				return fmt.Errorf("invalid index %q", args[0])
			}
			var sort *client.MoversSortOrder
			if moversSort != "" {
				s := client.MoversSortOrder(moversSort)
				if !s.Valid() {
					return fmt.Errorf("invalid --sort-order value %q", moversSort)
				}
				sort = &s
			}
			var freq *client.MoversFrequency
			if v, err := intFlag(cmd, "frequency"); err != nil {
				return err
			} else if v != nil {
				f := client.MoversFrequency(*v)
				if !f.Valid() {
					return fmt.Errorf("invalid --frequency value %d", *v)
				}
				freq = &f
			}
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetMovers(ctx, index, sort, freq)
			})
		},
	}
	moversCmd.Flags().StringVar(&moversSort, "sort-order", "",
		"VOLUME, TRADES, PERCENT_CHANGE_UP, or PERCENT_CHANGE_DOWN")
	moversCmd.Flags().Int("frequency", 0, "minimum move magnitude: 0, 1, 5, 10, 30, or 60")
	root.AddCommand(moversCmd)

	////////////////////////////////////////////////////////////////////////
	// Market hours

	var hoursDate string
	marketHoursCmd := &cobra.Command{
		Use:   "get-market-hours <market> [market...]",
		Short: "Market hours for one or more markets (schwab-py: get_market_hours)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var markets []client.MarketHoursMarket
			for _, a := range args {
				for _, part := range strings.Split(a, ",") {
					m := client.MarketHoursMarket(strings.TrimSpace(part))
					if !m.Valid() {
						return fmt.Errorf("invalid market %q", part)
					}
					markets = append(markets, m)
				}
			}
			var date *time.Time
			if hoursDate != "" {
				t, err := parseDateFlag(hoursDate)
				if err != nil {
					return err
				}
				date = &t
			}
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetMarketHours(ctx, markets, date)
			})
		},
	}
	marketHoursCmd.Flags().StringVar(&hoursDate, "date", "",
		"date to return hours for (YYYY-MM-DD)")
	root.AddCommand(marketHoursCmd)

	////////////////////////////////////////////////////////////////////////
	// Instruments

	var projection string
	instrumentsCmd := &cobra.Command{
		Use:   "get-instruments <symbol-or-search-term> [more...]",
		Short: "Search instruments or get fundamentals (schwab-py: get_instruments)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := client.InstrumentProjection(projection)
			if !p.Valid() {
				return fmt.Errorf("invalid --projection value %q", projection)
			}
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetInstruments(ctx, args, p)
			})
		},
	}
	instrumentsCmd.Flags().StringVar(&projection, "projection", "symbol-search",
		"symbol-search, symbol-regex, desc-search, desc-regex, search, or fundamental")
	root.AddCommand(instrumentsCmd)

	root.AddCommand(&cobra.Command{
		Use:   "get-instrument-by-cusip <cusip>",
		Short: "Instrument information by CUSIP (schwab-py: get_instrument_by_cusip)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newRESTClient()
			return runCall(func(ctx context.Context) (*client.Response, error) {
				return c.GetInstrumentByCusip(ctx, args[0])
			})
		},
	})
}

// buildOptionChainQuery converts the get-option-chain flags into the
// client's query struct, validating enum-valued flags.
func buildOptionChainQuery(cmd *cobra.Command) (client.OptionChainQuery, error) {
	var q client.OptionChainQuery
	var err error
	if s, _ := cmd.Flags().GetString("contract-type"); s != "" {
		v := client.OptionsContractType(s)
		if !v.Valid() {
			return q, fmt.Errorf("invalid --contract-type value %q", s)
		}
		q.ContractType = &v
	}
	if q.StrikeCount, err = intFlag(cmd, "strike-count"); err != nil {
		return q, err
	}
	if q.IncludeUnderlyingQuote, err = boolFlag(cmd, "include-underlying-quote"); err != nil {
		return q, err
	}
	if s, _ := cmd.Flags().GetString("strategy"); s != "" {
		v := client.OptionsStrategy(s)
		if !v.Valid() {
			return q, fmt.Errorf("invalid --strategy value %q", s)
		}
		q.Strategy = &v
	}
	if q.Interval, err = floatFlag(cmd, "interval"); err != nil {
		return q, err
	}
	if q.Strike, err = floatFlag(cmd, "strike"); err != nil {
		return q, err
	}
	if s, _ := cmd.Flags().GetString("strike-range"); s != "" {
		v := client.OptionsStrikeRange(s)
		if !v.Valid() {
			return q, fmt.Errorf("invalid --strike-range value %q", s)
		}
		q.StrikeRange = &v
	}
	if s, _ := cmd.Flags().GetString("from-date"); s != "" {
		t, err := parseDateFlag(s)
		if err != nil {
			return q, err
		}
		q.FromDate = &t
	}
	if s, _ := cmd.Flags().GetString("to-date"); s != "" {
		t, err := parseDateFlag(s)
		if err != nil {
			return q, err
		}
		q.ToDate = &t
	}
	if q.Volatility, err = floatFlag(cmd, "volatility"); err != nil {
		return q, err
	}
	if q.UnderlyingPrice, err = floatFlag(cmd, "underlying-price"); err != nil {
		return q, err
	}
	if q.InterestRate, err = floatFlag(cmd, "interest-rate"); err != nil {
		return q, err
	}
	if q.DaysToExpiration, err = intFlag(cmd, "days-to-expiration"); err != nil {
		return q, err
	}
	if s, _ := cmd.Flags().GetString("exp-month"); s != "" {
		v := client.OptionsExpirationMonth(s)
		if !v.Valid() {
			return q, fmt.Errorf("invalid --exp-month value %q", s)
		}
		q.ExpMonth = &v
	}
	if s, _ := cmd.Flags().GetString("option-type"); s != "" {
		v := client.OptionsType(s)
		if !v.Valid() {
			return q, fmt.Errorf("invalid --option-type value %q", s)
		}
		q.OptionType = &v
	}
	if s, _ := cmd.Flags().GetString("entitlement"); s != "" {
		v := client.OptionsEntitlement(s)
		if !v.Valid() {
			return q, fmt.Errorf("invalid --entitlement value %q", s)
		}
		q.Entitlement = &v
	}
	return q, nil
}

// buildPriceHistoryQuery converts the get-price-history flags into the
// client's query struct, validating enum-valued flags.
func buildPriceHistoryQuery(cmd *cobra.Command) (client.PriceHistoryQuery, error) {
	var q client.PriceHistoryQuery
	var err error
	if s, _ := cmd.Flags().GetString("period-type"); s != "" {
		v := client.PriceHistoryPeriodType(s)
		if !v.Valid() {
			return q, fmt.Errorf("invalid --period-type value %q", s)
		}
		q.PeriodType = &v
	}
	if v, err := intFlag(cmd, "period"); err != nil {
		return q, err
	} else if v != nil {
		p := client.PriceHistoryPeriod(*v)
		if !p.Valid() {
			return q, fmt.Errorf("invalid --period value %d", *v)
		}
		q.Period = &p
	}
	if s, _ := cmd.Flags().GetString("frequency-type"); s != "" {
		v := client.PriceHistoryFrequencyType(s)
		if !v.Valid() {
			return q, fmt.Errorf("invalid --frequency-type value %q", s)
		}
		q.FrequencyType = &v
	}
	if v, err := intFlag(cmd, "frequency"); err != nil {
		return q, err
	} else if v != nil {
		f := client.PriceHistoryFrequency(*v)
		if !f.Valid() {
			return q, fmt.Errorf("invalid --frequency value %d", *v)
		}
		q.Frequency = &f
	}
	if s, _ := cmd.Flags().GetString("start-datetime"); s != "" {
		t, err := parseTimeFlag(s)
		if err != nil {
			return q, err
		}
		q.StartDateTime = &t
	}
	if s, _ := cmd.Flags().GetString("end-datetime"); s != "" {
		t, err := parseTimeFlag(s)
		if err != nil {
			return q, err
		}
		q.EndDateTime = &t
	}
	if q.NeedExtendedHoursData, err = boolFlag(cmd, "need-extended-hours-data"); err != nil {
		return q, err
	}
	if q.NeedPreviousClose, err = boolFlag(cmd, "need-previous-close"); err != nil {
		return q, err
	}
	return q, nil
}
