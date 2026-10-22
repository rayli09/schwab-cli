// Package client is a Go port of schwab-py's schwab.client module
// (BaseClient in schwab/client/base.py plus the synchronous transport in
// schwab/client/synchronous.py). Method names, endpoints, query parameters,
// defaults, and enum wire values match schwab-py 1.5.1 exactly.
//
// Where schwab-py returns the raw HTTP response, this client returns a
// *Response carrying the status code, headers, and body.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultBaseURL is the Schwab API host, as hardcoded in schwab-py.
const DefaultBaseURL = "https://api.schwabapi.com"

// DefaultTimeout mirrors schwab-py's default session timeout of 30 seconds.
const DefaultTimeout = 30 * time.Second

// TokenSource provides access tokens, refreshing them as needed.
// *auth.Manager satisfies this interface.
type TokenSource interface {
	AccessToken(ctx context.Context) (string, error)
}

// Response is the result of an API call: the raw response pieces schwab-py
// leaves to the caller.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// IsError reports whether the status code indicates an error, mirroring
// httpx's Response.is_error used by schwab-py's order utilities.
func (r *Response) IsError() bool { return r.StatusCode >= 400 }

// JSON decodes the response body into v.
func (r *Response) JSON(v any) error {
	return json.Unmarshal(r.Body, v)
}

var orderLocationPattern = regexp.MustCompile(
	`https://api\.schwabapi\.com/trader/v1/accounts/(\w+)/orders/(\d+)`)

// ExtractOrderID attempts to extract the order ID from a place-order
// response's Location header, mirroring schwab.utils.Utils.extract_order_id.
// It returns (0, false, nil) when the response carries no usable Location
// header, and an error when the order was unsuccessful or the account hash
// in the Location does not match expectedAccountHash.
func ExtractOrderID(resp *Response, expectedAccountHash string) (int64, bool, error) {
	if resp.IsError() {
		return 0, false, fmt.Errorf("order not successful: status %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if location == "" {
		return 0, false, nil
	}
	m := orderLocationPattern.FindStringSubmatch(location)
	if m == nil {
		return 0, false, nil
	}
	if m[1] != expectedAccountHash {
		return 0, false, fmt.Errorf(
			"order request account hash != expected account hash")
	}
	id, err := strconv.ParseInt(m[2], 10, 64)
	if err != nil {
		return 0, false, nil
	}
	return id, true, nil
}

// Client is the Go analogue of schwab-py's Client.
type Client struct {
	BaseURL string // defaults to DefaultBaseURL
	APIKey  string
	Tokens  TokenSource
	HTTP    *http.Client
	Timeout time.Duration // defaults to DefaultTimeout

	// RateLimitPerMinute is the request budget the pacer enforces
	// (0 means DefaultRateLimitPerMinute, Schwab's published 120/min).
	RateLimitPerMinute int
	// Pacer, when set, replaces the pacer built from RateLimitPerMinute.
	// It is the injection point tests use to observe or speed up pacing.
	Pacer Pacer
	// MaxRetries is how many times a 429 response is retried
	// (0 means DefaultMaxRetries; a negative value disables retries).
	MaxRetries int
	// RetryBaseDelay is the first backoff delay after a 429 without a
	// Retry-After header (0 means DefaultRetryBaseDelay).
	RetryBaseDelay time.Duration

	pacerMu    sync.Mutex
	builtPacer Pacer
	pacerErr   error
}

// New builds a Client. tokens may be nil only in tests that also stub HTTP
// via a custom transport that does not require authorization.
func New(apiKey string, tokens TokenSource) *Client {
	return &Client{APIKey: apiKey, Tokens: tokens}
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	timeout := c.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	return &http.Client{Timeout: timeout}
}

// pacer returns the request pacer every call must pass through: the
// injected Pacer if one was set, otherwise a smooth (burst 1) token bucket
// built from RateLimitPerMinute, created lazily on first use.
func (c *Client) pacer() (Pacer, error) {
	if c.Pacer != nil {
		return c.Pacer, nil
	}
	c.pacerMu.Lock()
	defer c.pacerMu.Unlock()
	if c.builtPacer == nil && c.pacerErr == nil {
		rpm := c.RateLimitPerMinute
		if rpm == 0 {
			rpm = DefaultRateLimitPerMinute
		}
		c.builtPacer, c.pacerErr = NewRateLimiter(rpm)
	}
	return c.builtPacer, c.pacerErr
}

func (c *Client) maxRetries() int {
	if c.MaxRetries < 0 {
		return 0
	}
	if c.MaxRetries == 0 {
		return DefaultMaxRetries
	}
	return c.MaxRetries
}

func (c *Client) retryBaseDelay() time.Duration {
	if c.RetryBaseDelay <= 0 {
		return DefaultRetryBaseDelay
	}
	return c.RetryBaseDelay
}

// do performs one authenticated request, mirroring the _get_request /
// _post_request / _put_request / _delete_request helpers in schwab-py,
// with two additions schwab-py leaves to the caller: every attempt is
// paced through the rate limiter, and HTTP 429 responses are retried
// (honoring Retry-After, else exponential backoff) up to MaxRetries times.
func (c *Client) do(ctx context.Context, method, path string,
	params url.Values, jsonBody any) (*Response, error) {
	u := c.baseURL() + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	var bodyBytes []byte
	if jsonBody != nil {
		data, err := json.Marshal(jsonBody)
		if err != nil {
			return nil, err
		}
		bodyBytes = data
	}
	maxRetries := c.maxRetries()
	for attempt := 0; ; attempt++ {
		pacer, err := c.pacer()
		if err != nil {
			return nil, err
		}
		if err := pacer.Wait(ctx); err != nil {
			return nil, err
		}
		resp, err := c.doOnce(ctx, method, u, bodyBytes)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}
		if attempt >= maxRetries {
			rpm := c.RateLimitPerMinute
			if rpm <= 0 {
				rpm = DefaultRateLimitPerMinute
			}
			return nil, fmt.Errorf("schwab API rate limit exceeded: "+
				"HTTP 429 persisted through %d attempt(s) at a configured "+
				"limit of %d requests/minute; wait before retrying, and "+
				"only raise the rate limit if your app's tier allows more "+
				"than Schwab's standard 120 requests/minute",
				attempt+1, rpm)
		}
		delay := retryDelay(resp.Header, attempt, c.retryBaseDelay())
		if err := sleepContext(ctx, delay); err != nil {
			return nil, err
		}
	}
}

// doOnce issues a single HTTP attempt for do().
func (c *Client) doOnce(ctx context.Context, method, u string,
	bodyBytes []byte) (*Response, error) {
	var body io.Reader
	if bodyBytes != nil {
		body = bytes.NewReader(bodyBytes)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	if c.Tokens != nil {
		tok, err := c.Tokens.AccessToken(ctx)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	req.Header.Set("Accept", "application/json")
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: data}, nil
}

func (c *Client) get(ctx context.Context, path string, params url.Values) (*Response, error) {
	if params == nil {
		params = url.Values{}
	}
	return c.do(ctx, http.MethodGet, path, params, nil)
}

func (c *Client) post(ctx context.Context, path string, body any) (*Response, error) {
	return c.do(ctx, http.MethodPost, path, nil, body)
}

func (c *Client) put(ctx context.Context, path string, body any) (*Response, error) {
	return c.do(ctx, http.MethodPut, path, nil, body)
}

func (c *Client) delete(ctx context.Context, path string) (*Response, error) {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// FormatDateAsISO formats a time as yyyy-MM-dd'T'HH:mm:ss.SSSZ the way
// schwab-py's _format_date_as_iso does (strftime '%Y-%m-%dT%H:%M:%SZ').
// The time is rendered in UTC.
func FormatDateAsISO(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

// FormatDateAsDay formats a time as YYYY-MM-DD (schwab-py's
// _format_date_as_day).
func FormatDateAsDay(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

// FormatDateAsMillis converts a time to epoch milliseconds (schwab-py's
// _format_date_as_millis).
func FormatDateAsMillis(t time.Time) int64 {
	return t.UnixMilli()
}

func boolParam(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func joinStrings[T ~string](vals []T) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = string(v)
	}
	return strings.Join(parts, ",")
}

////////////////////////////////////////////////////////////////////////////
// Accounts — schwab-py: BaseClient.get_account / get_accounts /
// get_account_numbers

// GetAccount returns account balances, positions, and orders for a given
// account hash. (schwab-py: get_account)
func (c *Client) GetAccount(ctx context.Context, accountHash string,
	fields []AccountFields) (*Response, error) {
	params := url.Values{}
	if len(fields) > 0 {
		for _, f := range fields {
			if !f.Valid() {
				return nil, fmt.Errorf("invalid Account.Fields value %q", f)
			}
		}
		params.Set("fields", joinStrings(fields))
	}
	return c.get(ctx, "/trader/v1/accounts/"+accountHash, params)
}

// GetAccountNumbers returns the mapping from account numbers to the account
// hashes used in API calls. (schwab-py: get_account_numbers)
func (c *Client) GetAccountNumbers(ctx context.Context) (*Response, error) {
	return c.get(ctx, "/trader/v1/accounts/accountNumbers", nil)
}

// GetAccounts returns balances, positions, and orders for all linked
// accounts. (schwab-py: get_accounts)
func (c *Client) GetAccounts(ctx context.Context, fields []AccountFields) (*Response, error) {
	params := url.Values{}
	if len(fields) > 0 {
		for _, f := range fields {
			if !f.Valid() {
				return nil, fmt.Errorf("invalid Account.Fields value %q", f)
			}
		}
		params.Set("fields", joinStrings(fields))
	}
	return c.get(ctx, "/trader/v1/accounts", params)
}

////////////////////////////////////////////////////////////////////////////
// Orders — schwab-py: BaseClient order methods

// GetOrder returns a specific order for a specific account.
// (schwab-py: get_order)
func (c *Client) GetOrder(ctx context.Context, orderID, accountHash string) (*Response, error) {
	return c.get(ctx, "/trader/v1/accounts/"+accountHash+"/orders/"+orderID, nil)
}

// CancelOrder cancels a specific order for a specific account.
// (schwab-py: cancel_order)
func (c *Client) CancelOrder(ctx context.Context, orderID, accountHash string) (*Response, error) {
	return c.delete(ctx, "/trader/v1/accounts/"+accountHash+"/orders/"+orderID)
}

// OrderQuery carries the optional parameters shared by
// GetOrdersForAccount and GetOrdersForAllLinkedAccounts, mirroring
// schwab-py's _make_order_query.
type OrderQuery struct {
	MaxResults      *int
	FromEnteredTime *time.Time // defaults to 60 days before now (UTC)
	ToEnteredTime   *time.Time // defaults to now (UTC)
	Status          *OrderStatus
}

func (c *Client) makeOrderQuery(q OrderQuery) (url.Values, error) {
	now := time.Now().UTC()
	from := now.Add(-60 * 24 * time.Hour)
	if q.FromEnteredTime != nil {
		from = *q.FromEnteredTime
	}
	to := now
	if q.ToEnteredTime != nil {
		to = *q.ToEnteredTime
	}
	params := url.Values{}
	params.Set("fromEnteredTime", FormatDateAsISO(from))
	params.Set("toEnteredTime", FormatDateAsISO(to))
	if q.MaxResults != nil {
		params.Set("maxResults", strconv.Itoa(*q.MaxResults))
	}
	if q.Status != nil {
		if !q.Status.Valid() {
			return nil, fmt.Errorf("invalid Order.Status value %q", *q.Status)
		}
		params.Set("status", string(*q.Status))
	}
	return params, nil
}

// GetOrdersForAccount returns orders for a specific account.
// (schwab-py: get_orders_for_account)
func (c *Client) GetOrdersForAccount(ctx context.Context, accountHash string,
	q OrderQuery) (*Response, error) {
	params, err := c.makeOrderQuery(q)
	if err != nil {
		return nil, err
	}
	return c.get(ctx, "/trader/v1/accounts/"+accountHash+"/orders", params)
}

// GetOrdersForAllLinkedAccounts returns orders for all linked accounts.
// (schwab-py: get_orders_for_all_linked_accounts)
func (c *Client) GetOrdersForAllLinkedAccounts(ctx context.Context,
	q OrderQuery) (*Response, error) {
	params, err := c.makeOrderQuery(q)
	if err != nil {
		return nil, err
	}
	return c.get(ctx, "/trader/v1/orders", params)
}

// PlaceOrder places an order for a specific account. orderSpec is the order
// JSON object schwab-py accepts as order_spec (a map, or an order builder's
// built output). (schwab-py: place_order)
func (c *Client) PlaceOrder(ctx context.Context, accountHash string,
	orderSpec any) (*Response, error) {
	return c.post(ctx, "/trader/v1/accounts/"+accountHash+"/orders", orderSpec)
}

// ReplaceOrder replaces an existing order for an account.
// (schwab-py: replace_order)
func (c *Client) ReplaceOrder(ctx context.Context, accountHash, orderID string,
	orderSpec any) (*Response, error) {
	return c.put(ctx, "/trader/v1/accounts/"+accountHash+"/orders/"+orderID, orderSpec)
}

// PreviewOrder previews an order: tests whether it would be accepted and
// shows the structure it would result in. (schwab-py: preview_order)
func (c *Client) PreviewOrder(ctx context.Context, accountHash string,
	orderSpec any) (*Response, error) {
	return c.post(ctx, "/trader/v1/accounts/"+accountHash+"/previewOrder", orderSpec)
}

////////////////////////////////////////////////////////////////////////////
// Transactions — schwab-py: BaseClient.get_transactions / get_transaction

// TransactionsQuery carries the optional parameters of GetTransactions.
type TransactionsQuery struct {
	StartDate *time.Time        // defaults to 60 days before now (UTC)
	EndDate   *time.Time        // defaults to now (UTC)
	Types     []TransactionType // defaults to all types, as in schwab-py
	Symbol    *string
}

// GetTransactions returns transactions for a specific account.
// (schwab-py: get_transactions)
func (c *Client) GetTransactions(ctx context.Context, accountHash string,
	q TransactionsQuery) (*Response, error) {
	types := q.Types
	if len(types) == 0 {
		types = AllTransactionTypes()
	}
	for _, t := range types {
		if !t.Valid() {
			return nil, fmt.Errorf("invalid Transactions.TransactionType value %q", t)
		}
	}
	now := time.Now().UTC()
	start := now.Add(-60 * 24 * time.Hour)
	if q.StartDate != nil {
		start = *q.StartDate
	}
	end := now
	if q.EndDate != nil {
		end = *q.EndDate
	}
	params := url.Values{}
	params.Set("types", joinStrings(types))
	params.Set("startDate", FormatDateAsISO(start))
	params.Set("endDate", FormatDateAsISO(end))
	if q.Symbol != nil {
		params.Set("symbol", *q.Symbol)
	}
	return c.get(ctx, "/trader/v1/accounts/"+accountHash+"/transactions", params)
}

// GetTransaction returns a single transaction for a specific account.
// (schwab-py: get_transaction)
func (c *Client) GetTransaction(ctx context.Context, accountHash,
	transactionID string) (*Response, error) {
	return c.get(ctx,
		"/trader/v1/accounts/"+accountHash+"/transactions/"+transactionID, nil)
}

////////////////////////////////////////////////////////////////////////////
// User preferences — schwab-py: BaseClient.get_user_preferences

// GetUserPreferences returns preferences for the logged-in user, including
// streaming information used by the streaming client.
// (schwab-py: get_user_preferences)
func (c *Client) GetUserPreferences(ctx context.Context) (*Response, error) {
	return c.get(ctx, "/trader/v1/userPreference", nil)
}

////////////////////////////////////////////////////////////////////////////
// Quotes — schwab-py: BaseClient.get_quote / get_quotes

// GetQuote returns a quote for a single symbol. (schwab-py: get_quote)
func (c *Client) GetQuote(ctx context.Context, symbol string,
	fields []QuoteFields) (*Response, error) {
	params := url.Values{}
	if len(fields) > 0 {
		for _, f := range fields {
			if !f.Valid() {
				return nil, fmt.Errorf("invalid Quote.Fields value %q", f)
			}
		}
		params.Set("fields", joinStrings(fields))
	}
	// schwab-py interpolates the symbol into the path verbatim; do the same.
	return c.get(ctx, "/marketdata/v1/"+symbol+"/quotes", params)
}

// GetQuotes returns quotes for one or more symbols. (schwab-py: get_quotes)
func (c *Client) GetQuotes(ctx context.Context, symbols []string,
	fields []QuoteFields, indicative *bool) (*Response, error) {
	params := url.Values{}
	params.Set("symbols", strings.Join(symbols, ","))
	if len(fields) > 0 {
		for _, f := range fields {
			if !f.Valid() {
				return nil, fmt.Errorf("invalid Quote.Fields value %q", f)
			}
		}
		params.Set("fields", joinStrings(fields))
	}
	if indicative != nil {
		params.Set("indicative", boolParam(*indicative))
	}
	return c.get(ctx, "/marketdata/v1/quotes", params)
}

////////////////////////////////////////////////////////////////////////////
// Option chain — schwab-py: BaseClient.get_option_chain /
// get_option_expiration_chain

// OptionChainQuery carries the optional parameters of GetOptionChain,
// mirroring the keyword arguments of schwab-py's get_option_chain.
type OptionChainQuery struct {
	ContractType           *OptionsContractType
	StrikeCount            *int
	IncludeUnderlyingQuote *bool
	Strategy               *OptionsStrategy
	Interval               *float64
	Strike                 *float64
	StrikeRange            *OptionsStrikeRange
	FromDate               *time.Time
	ToDate                 *time.Time
	Volatility             *float64
	UnderlyingPrice        *float64
	InterestRate           *float64
	DaysToExpiration       *int
	ExpMonth               *OptionsExpirationMonth
	OptionType             *OptionsType
	Entitlement            *OptionsEntitlement
}

// GetOptionChain returns the option chain for an optionable symbol.
// (schwab-py: get_option_chain)
func (c *Client) GetOptionChain(ctx context.Context, symbol string,
	q OptionChainQuery) (*Response, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if q.ContractType != nil {
		if !q.ContractType.Valid() {
			return nil, fmt.Errorf("invalid Options.ContractType value %q", *q.ContractType)
		}
		params.Set("contractType", string(*q.ContractType))
	}
	if q.StrikeCount != nil {
		params.Set("strikeCount", strconv.Itoa(*q.StrikeCount))
	}
	if q.IncludeUnderlyingQuote != nil {
		params.Set("includeUnderlyingQuote", boolParam(*q.IncludeUnderlyingQuote))
	}
	if q.Strategy != nil {
		if !q.Strategy.Valid() {
			return nil, fmt.Errorf("invalid Options.Strategy value %q", *q.Strategy)
		}
		params.Set("strategy", string(*q.Strategy))
	}
	if q.Interval != nil {
		params.Set("interval", strconv.FormatFloat(*q.Interval, 'f', -1, 64))
	}
	if q.Strike != nil {
		params.Set("strike", strconv.FormatFloat(*q.Strike, 'f', -1, 64))
	}
	if q.StrikeRange != nil {
		if !q.StrikeRange.Valid() {
			return nil, fmt.Errorf("invalid Options.StrikeRange value %q", *q.StrikeRange)
		}
		params.Set("range", string(*q.StrikeRange))
	}
	if q.FromDate != nil {
		params.Set("fromDate", FormatDateAsDay(*q.FromDate))
	}
	if q.ToDate != nil {
		params.Set("toDate", FormatDateAsDay(*q.ToDate))
	}
	if q.Volatility != nil {
		params.Set("volatility", strconv.FormatFloat(*q.Volatility, 'f', -1, 64))
	}
	if q.UnderlyingPrice != nil {
		params.Set("underlyingPrice", strconv.FormatFloat(*q.UnderlyingPrice, 'f', -1, 64))
	}
	if q.InterestRate != nil {
		params.Set("interestRate", strconv.FormatFloat(*q.InterestRate, 'f', -1, 64))
	}
	if q.DaysToExpiration != nil {
		params.Set("daysToExpiration", strconv.Itoa(*q.DaysToExpiration))
	}
	if q.ExpMonth != nil {
		if !q.ExpMonth.Valid() {
			return nil, fmt.Errorf("invalid Options.ExpirationMonth value %q", *q.ExpMonth)
		}
		params.Set("expMonth", string(*q.ExpMonth))
	}
	if q.OptionType != nil {
		if !q.OptionType.Valid() {
			return nil, fmt.Errorf("invalid Options.Type value %q", *q.OptionType)
		}
		params.Set("optionType", string(*q.OptionType))
	}
	if q.Entitlement != nil {
		if !q.Entitlement.Valid() {
			return nil, fmt.Errorf("invalid Options.Entitlement value %q", *q.Entitlement)
		}
		params.Set("entitlement", string(*q.Entitlement))
	}
	return c.get(ctx, "/marketdata/v1/chains", params)
}

// GetOptionExpirationChain returns the option expiration chain for a symbol.
// (schwab-py: get_option_expiration_chain)
func (c *Client) GetOptionExpirationChain(ctx context.Context,
	symbol string) (*Response, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	return c.get(ctx, "/marketdata/v1/expirationchain", params)
}

////////////////////////////////////////////////////////////////////////////
// Price history — schwab-py: BaseClient.get_price_history and the
// get_price_history_every_* convenience wrappers

// PriceHistoryQuery carries the optional parameters of GetPriceHistory.
type PriceHistoryQuery struct {
	PeriodType            *PriceHistoryPeriodType
	Period                *PriceHistoryPeriod
	FrequencyType         *PriceHistoryFrequencyType
	Frequency             *PriceHistoryFrequency
	StartDateTime         *time.Time
	EndDateTime           *time.Time
	NeedExtendedHoursData *bool
	NeedPreviousClose     *bool
}

// GetPriceHistory returns price history for a symbol.
// (schwab-py: get_price_history)
func (c *Client) GetPriceHistory(ctx context.Context, symbol string,
	q PriceHistoryQuery) (*Response, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if q.PeriodType != nil {
		if !q.PeriodType.Valid() {
			return nil, fmt.Errorf("invalid PriceHistory.PeriodType value %q", *q.PeriodType)
		}
		params.Set("periodType", string(*q.PeriodType))
	}
	if q.Period != nil {
		if !q.Period.Valid() {
			return nil, fmt.Errorf("invalid PriceHistory.Period value %d", *q.Period)
		}
		params.Set("period", strconv.Itoa(int(*q.Period)))
	}
	if q.FrequencyType != nil {
		if !q.FrequencyType.Valid() {
			return nil, fmt.Errorf("invalid PriceHistory.FrequencyType value %q", *q.FrequencyType)
		}
		params.Set("frequencyType", string(*q.FrequencyType))
	}
	if q.Frequency != nil {
		if !q.Frequency.Valid() {
			return nil, fmt.Errorf("invalid PriceHistory.Frequency value %d", *q.Frequency)
		}
		params.Set("frequency", strconv.Itoa(int(*q.Frequency)))
	}
	if q.StartDateTime != nil {
		params.Set("startDate", strconv.FormatInt(FormatDateAsMillis(*q.StartDateTime), 10))
	}
	if q.EndDateTime != nil {
		params.Set("endDate", strconv.FormatInt(FormatDateAsMillis(*q.EndDateTime), 10))
	}
	if q.NeedExtendedHoursData != nil {
		params.Set("needExtendedHoursData", boolParam(*q.NeedExtendedHoursData))
	}
	if q.NeedPreviousClose != nil {
		params.Set("needPreviousClose", boolParam(*q.NeedPreviousClose))
	}
	return c.get(ctx, "/marketdata/v1/pricehistory", params)
}

// normalizePriceHistoryRange mirrors schwab-py's
// __normalize_start_and_end_datetimes: start defaults to 1971-01-01 and end
// to seven days from now.
func normalizePriceHistoryRange(start, end *time.Time) (time.Time, time.Time) {
	s := time.Date(1971, 1, 1, 0, 0, 0, 0, time.UTC)
	if start != nil {
		s = *start
	}
	e := time.Now().UTC().Add(7 * 24 * time.Hour)
	if end != nil {
		e = *end
	}
	return s, e
}

func (c *Client) priceHistoryVariant(ctx context.Context, symbol string,
	periodType PriceHistoryPeriodType, period PriceHistoryPeriod,
	frequencyType PriceHistoryFrequencyType, frequency PriceHistoryFrequency,
	start, end *time.Time, needExtendedHoursData, needPreviousClose *bool) (*Response, error) {
	s, e := normalizePriceHistoryRange(start, end)
	return c.GetPriceHistory(ctx, symbol, PriceHistoryQuery{
		PeriodType:            &periodType,
		Period:                &period,
		FrequencyType:         &frequencyType,
		Frequency:             &frequency,
		StartDateTime:         &s,
		EndDateTime:           &e,
		NeedExtendedHoursData: needExtendedHoursData,
		NeedPreviousClose:     needPreviousClose,
	})
}

// GetPriceHistoryEveryMinute — schwab-py: get_price_history_every_minute
func (c *Client) GetPriceHistoryEveryMinute(ctx context.Context, symbol string,
	start, end *time.Time, needExtendedHoursData, needPreviousClose *bool) (*Response, error) {
	return c.priceHistoryVariant(ctx, symbol,
		PriceHistoryPeriodTypeDay, PriceHistoryPeriodOneDay,
		PriceHistoryFrequencyTypeMinute, PriceHistoryFrequencyEveryMinute,
		start, end, needExtendedHoursData, needPreviousClose)
}

// GetPriceHistoryEveryFiveMinutes — schwab-py: get_price_history_every_five_minutes
func (c *Client) GetPriceHistoryEveryFiveMinutes(ctx context.Context, symbol string,
	start, end *time.Time, needExtendedHoursData, needPreviousClose *bool) (*Response, error) {
	return c.priceHistoryVariant(ctx, symbol,
		PriceHistoryPeriodTypeDay, PriceHistoryPeriodOneDay,
		PriceHistoryFrequencyTypeMinute, PriceHistoryFrequencyEveryFiveMinutes,
		start, end, needExtendedHoursData, needPreviousClose)
}

// GetPriceHistoryEveryTenMinutes — schwab-py: get_price_history_every_ten_minutes
func (c *Client) GetPriceHistoryEveryTenMinutes(ctx context.Context, symbol string,
	start, end *time.Time, needExtendedHoursData, needPreviousClose *bool) (*Response, error) {
	return c.priceHistoryVariant(ctx, symbol,
		PriceHistoryPeriodTypeDay, PriceHistoryPeriodOneDay,
		PriceHistoryFrequencyTypeMinute, PriceHistoryFrequencyEveryTenMinutes,
		start, end, needExtendedHoursData, needPreviousClose)
}

// GetPriceHistoryEveryFifteenMinutes — schwab-py: get_price_history_every_fifteen_minutes
func (c *Client) GetPriceHistoryEveryFifteenMinutes(ctx context.Context, symbol string,
	start, end *time.Time, needExtendedHoursData, needPreviousClose *bool) (*Response, error) {
	return c.priceHistoryVariant(ctx, symbol,
		PriceHistoryPeriodTypeDay, PriceHistoryPeriodOneDay,
		PriceHistoryFrequencyTypeMinute, PriceHistoryFrequencyEveryFifteenMin,
		start, end, needExtendedHoursData, needPreviousClose)
}

// GetPriceHistoryEveryThirtyMinutes — schwab-py: get_price_history_every_thirty_minutes
func (c *Client) GetPriceHistoryEveryThirtyMinutes(ctx context.Context, symbol string,
	start, end *time.Time, needExtendedHoursData, needPreviousClose *bool) (*Response, error) {
	return c.priceHistoryVariant(ctx, symbol,
		PriceHistoryPeriodTypeDay, PriceHistoryPeriodOneDay,
		PriceHistoryFrequencyTypeMinute, PriceHistoryFrequencyEveryThirtyMin,
		start, end, needExtendedHoursData, needPreviousClose)
}

// GetPriceHistoryEveryDay — schwab-py: get_price_history_every_day
func (c *Client) GetPriceHistoryEveryDay(ctx context.Context, symbol string,
	start, end *time.Time, needExtendedHoursData, needPreviousClose *bool) (*Response, error) {
	return c.priceHistoryVariant(ctx, symbol,
		PriceHistoryPeriodTypeYear, PriceHistoryPeriodTwentyYears,
		PriceHistoryFrequencyTypeDaily, PriceHistoryFrequencyDaily,
		start, end, needExtendedHoursData, needPreviousClose)
}

// GetPriceHistoryEveryWeek — schwab-py: get_price_history_every_week
func (c *Client) GetPriceHistoryEveryWeek(ctx context.Context, symbol string,
	start, end *time.Time, needExtendedHoursData, needPreviousClose *bool) (*Response, error) {
	return c.priceHistoryVariant(ctx, symbol,
		PriceHistoryPeriodTypeYear, PriceHistoryPeriodTwentyYears,
		PriceHistoryFrequencyTypeWeekly, PriceHistoryFrequencyWeekly,
		start, end, needExtendedHoursData, needPreviousClose)
}

////////////////////////////////////////////////////////////////////////////
// Movers — schwab-py: BaseClient.get_movers

// GetMovers returns the top movers for a given index.
// (schwab-py: get_movers)
func (c *Client) GetMovers(ctx context.Context, index MoversIndex,
	sortOrder *MoversSortOrder, frequency *MoversFrequency) (*Response, error) {
	if !index.Valid() {
		return nil, fmt.Errorf("invalid Movers.Index value %q", index)
	}
	params := url.Values{}
	if sortOrder != nil {
		if !sortOrder.Valid() {
			return nil, fmt.Errorf("invalid Movers.SortOrder value %q", *sortOrder)
		}
		params.Set("sort", string(*sortOrder))
	}
	if frequency != nil {
		if !frequency.Valid() {
			return nil, fmt.Errorf("invalid Movers.Frequency value %d", *frequency)
		}
		params.Set("frequency", strconv.Itoa(int(*frequency)))
	}
	return c.get(ctx, "/marketdata/v1/movers/"+string(index), params)
}

////////////////////////////////////////////////////////////////////////////
// Market hours — schwab-py: BaseClient.get_market_hours

// GetMarketHours returns market hours for the specified markets (one or
// more). A single market is simply a one-element list, exactly as in
// schwab-py. (schwab-py: get_market_hours)
func (c *Client) GetMarketHours(ctx context.Context, markets []MarketHoursMarket,
	date *time.Time) (*Response, error) {
	for _, m := range markets {
		if !m.Valid() {
			return nil, fmt.Errorf("invalid MarketHours.Market value %q", m)
		}
	}
	params := url.Values{}
	params.Set("markets", joinStrings(markets))
	if date != nil {
		params.Set("date", FormatDateAsDay(*date))
	}
	return c.get(ctx, "/marketdata/v1/markets", params)
}

////////////////////////////////////////////////////////////////////////////
// Instruments — schwab-py: BaseClient.get_instruments /
// get_instrument_by_cusip

// GetInstruments searches instruments or returns fundamentals, depending on
// the projection. (schwab-py: get_instruments)
func (c *Client) GetInstruments(ctx context.Context, symbols []string,
	projection InstrumentProjection) (*Response, error) {
	if !projection.Valid() {
		return nil, fmt.Errorf("invalid Instrument.Projection value %q", projection)
	}
	params := url.Values{}
	params.Set("symbol", strings.Join(symbols, ","))
	params.Set("projection", string(projection))
	return c.get(ctx, "/marketdata/v1/instruments", params)
}

// GetInstrumentByCusip returns instrument information for a single CUSIP.
// (schwab-py: get_instrument_by_cusip)
func (c *Client) GetInstrumentByCusip(ctx context.Context, cusip string) (*Response, error) {
	return c.get(ctx, "/marketdata/v1/instruments/"+cusip, nil)
}
