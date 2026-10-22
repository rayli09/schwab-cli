package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

type staticTokens struct{ tok string }

func (s staticTokens) AccessToken(context.Context) (string, error) { return s.tok, nil }

type capturedRequest struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	Body   []byte
}

// newTestClient returns a Client pointed at an httptest server that records
// the last request and replies with the given status/body/headers.
func newTestClient(t *testing.T, status int, body string,
	respHeaders map[string]string) (*Client, *capturedRequest) {
	t.Helper()
	cap := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			cap.Method = r.Method
			cap.Path = r.URL.Path
			cap.Query = r.URL.Query()
			cap.Header = r.Header
			cap.Body, _ = io.ReadAll(r.Body)
			for k, v := range respHeaders {
				w.Header().Set(k, v)
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
	t.Cleanup(srv.Close)
	c := New("KEY_PLACEHOLDER", staticTokens{"TOKEN_PLACEHOLDER"})
	c.BaseURL = srv.URL
	return c, cap
}

func ctx() context.Context { return context.Background() }

func TestGetQuoteRequest(t *testing.T) {
	c, cap := newTestClient(t, 200, `{}`, nil)
	if _, err := c.GetQuote(ctx(), "AAPL",
		[]QuoteFields{QuoteFieldsQuote, QuoteFieldsFundamental}); err != nil {
		t.Fatal(err)
	}
	if cap.Path != "/marketdata/v1/AAPL/quotes" {
		t.Errorf("path = %s", cap.Path)
	}
	if got := cap.Query.Get("fields"); got != "quote,fundamental" {
		t.Errorf("fields = %q", got)
	}
	if got := cap.Header.Get("Authorization"); got != "Bearer TOKEN_PLACEHOLDER" {
		t.Errorf("Authorization = %q", got)
	}
}

func TestGetQuotesRequest(t *testing.T) {
	c, cap := newTestClient(t, 200, `{}`, nil)
	tr := true
	if _, err := c.GetQuotes(ctx(), []string{"AAPL", "$SPX", "BRK/B"}, nil, &tr); err != nil {
		t.Fatal(err)
	}
	if cap.Path != "/marketdata/v1/quotes" {
		t.Errorf("path = %s", cap.Path)
	}
	if got := cap.Query.Get("symbols"); got != "AAPL,$SPX,BRK/B" {
		t.Errorf("symbols = %q", got)
	}
	if got := cap.Query.Get("indicative"); got != "true" {
		t.Errorf("indicative = %q", got)
	}
}

func TestGetAccountNumbersPath(t *testing.T) {
	c, cap := newTestClient(t, 200, `[]`, nil)
	if _, err := c.GetAccountNumbers(ctx()); err != nil {
		t.Fatal(err)
	}
	if cap.Path != "/trader/v1/accounts/accountNumbers" {
		t.Errorf("path = %s", cap.Path)
	}
}

func TestGetOrdersForAccountDefaults(t *testing.T) {
	c, cap := newTestClient(t, 200, `[]`, nil)
	before := time.Now().UTC()
	if _, err := c.GetOrdersForAccount(ctx(), "HASH1", OrderQuery{}); err != nil {
		t.Fatal(err)
	}
	after := time.Now().UTC()
	if cap.Path != "/trader/v1/accounts/HASH1/orders" {
		t.Errorf("path = %s", cap.Path)
	}
	from, err := time.Parse("2006-01-02T15:04:05Z", cap.Query.Get("fromEnteredTime"))
	if err != nil {
		t.Fatalf("fromEnteredTime = %q: %v", cap.Query.Get("fromEnteredTime"), err)
	}
	// Default: 60 days before now, as in schwab-py's _make_order_query.
	wantFrom := before.Add(-60 * 24 * time.Hour)
	if from.Before(wantFrom.Add(-time.Minute)) || from.After(after.Add(-60*24*time.Hour).Add(time.Minute)) {
		t.Errorf("fromEnteredTime = %v, want ~%v", from, wantFrom)
	}
	if _, err := time.Parse("2006-01-02T15:04:05Z", cap.Query.Get("toEnteredTime")); err != nil {
		t.Errorf("toEnteredTime malformed: %v", err)
	}
	if cap.Query.Has("maxResults") || cap.Query.Has("status") {
		t.Errorf("unexpected optional params present: %v", cap.Query)
	}
}

func TestGetOrdersForAllLinkedAccounts(t *testing.T) {
	c, cap := newTestClient(t, 200, `[]`, nil)
	max := 25
	status := OrderStatusWorking
	from := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	to := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	q := OrderQuery{MaxResults: &max, Status: &status,
		FromEnteredTime: &from, ToEnteredTime: &to}
	if _, err := c.GetOrdersForAllLinkedAccounts(ctx(), q); err != nil {
		t.Fatal(err)
	}
	if cap.Path != "/trader/v1/orders" {
		t.Errorf("path = %s", cap.Path)
	}
	if cap.Query.Get("maxResults") != "25" ||
		cap.Query.Get("status") != "WORKING" ||
		cap.Query.Get("fromEnteredTime") != "2026-01-02T03:04:05Z" ||
		cap.Query.Get("toEnteredTime") != "2026-02-03T04:05:06Z" {
		t.Errorf("query = %v", cap.Query)
	}
}

func TestGetTransactionsDefaults(t *testing.T) {
	c, cap := newTestClient(t, 200, `[]`, nil)
	if _, err := c.GetTransactions(ctx(), "HASH1", TransactionsQuery{}); err != nil {
		t.Fatal(err)
	}
	if cap.Path != "/trader/v1/accounts/HASH1/transactions" {
		t.Errorf("path = %s", cap.Path)
	}
	// schwab-py defaults to all transaction types, in enum order.
	wantTypes := "TRADE,RECEIVE_AND_DELIVER,DIVIDEND_OR_INTEREST,ACH_RECEIPT," +
		"ACH_DISBURSEMENT,CASH_RECEIPT,CASH_DISBURSEMENT,ELECTRONIC_FUND," +
		"WIRE_OUT,WIRE_IN,JOURNAL,MEMORANDUM,MARGIN_CALL,MONEY_MARKET,SMA_ADJUSTMENT"
	if got := cap.Query.Get("types"); got != wantTypes {
		t.Errorf("types =\n  %q\nwant\n  %q", got, wantTypes)
	}
	if !cap.Query.Has("startDate") || !cap.Query.Has("endDate") {
		t.Errorf("missing date params: %v", cap.Query)
	}
}

func TestPlaceOrderAndExtractID(t *testing.T) {
	loc := "https://api.schwabapi.com/trader/v1/accounts/HASH1/orders/123456789"
	c, cap := newTestClient(t, 201, ``, map[string]string{"Location": loc})
	spec := map[string]any{"orderStrategyType": "SINGLE", "orderType": "MARKET"}
	resp, err := c.PlaceOrder(ctx(), "HASH1", spec)
	if err != nil {
		t.Fatal(err)
	}
	if cap.Method != http.MethodPost ||
		cap.Path != "/trader/v1/accounts/HASH1/orders" {
		t.Errorf("%s %s", cap.Method, cap.Path)
	}
	var sent map[string]any
	if err := json.Unmarshal(cap.Body, &sent); err != nil {
		t.Fatalf("order body not JSON: %v", err)
	}
	if sent["orderType"] != "MARKET" {
		t.Errorf("body = %s", cap.Body)
	}
	id, ok, err := ExtractOrderID(resp, "HASH1")
	if err != nil || !ok || id != 123456789 {
		t.Errorf("ExtractOrderID = (%d, %v, %v)", id, ok, err)
	}
	if _, _, err := ExtractOrderID(resp, "OTHER"); err == nil {
		t.Error("expected account-hash mismatch error")
	}
}

func TestReplaceAndPreviewPaths(t *testing.T) {
	c, cap := newTestClient(t, 200, `{}`, nil)
	spec := map[string]any{"orderStrategyType": "SINGLE"}
	if _, err := c.ReplaceOrder(ctx(), "HASH1", "42", spec); err != nil {
		t.Fatal(err)
	}
	if cap.Method != http.MethodPut ||
		cap.Path != "/trader/v1/accounts/HASH1/orders/42" {
		t.Errorf("replace: %s %s", cap.Method, cap.Path)
	}
	if _, err := c.PreviewOrder(ctx(), "HASH1", spec); err != nil {
		t.Fatal(err)
	}
	if cap.Method != http.MethodPost ||
		cap.Path != "/trader/v1/accounts/HASH1/previewOrder" {
		t.Errorf("preview: %s %s", cap.Method, cap.Path)
	}
}

func TestCancelOrder(t *testing.T) {
	c, cap := newTestClient(t, 200, ``, nil)
	if _, err := c.CancelOrder(ctx(), "42", "HASH1"); err != nil {
		t.Fatal(err)
	}
	if cap.Method != http.MethodDelete ||
		cap.Path != "/trader/v1/accounts/HASH1/orders/42" {
		t.Errorf("%s %s", cap.Method, cap.Path)
	}
}

func TestGetOptionChainParams(t *testing.T) {
	c, cap := newTestClient(t, 200, `{}`, nil)
	ct := OptionsContractTypeCall
	sc := 10
	strategy := OptionsStrategySingle
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	q := OptionChainQuery{ContractType: &ct, StrikeCount: &sc,
		Strategy: &strategy, FromDate: &from, ToDate: &to}
	if _, err := c.GetOptionChain(ctx(), "AAPL", q); err != nil {
		t.Fatal(err)
	}
	if cap.Path != "/marketdata/v1/chains" {
		t.Errorf("path = %s", cap.Path)
	}
	checks := map[string]string{
		"symbol":       "AAPL",
		"contractType": "CALL",
		"strikeCount":  "10",
		"strategy":     "SINGLE",
		"fromDate":     "2026-11-01",
		"toDate":       "2026-12-01",
	}
	for k, want := range checks {
		if got := cap.Query.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestPriceHistoryVariantEveryMinute(t *testing.T) {
	c, cap := newTestClient(t, 200, `{}`, nil)
	start := time.Date(2026, 10, 1, 13, 30, 0, 0, time.UTC)
	end := time.Date(2026, 10, 1, 20, 0, 0, 0, time.UTC)
	if _, err := c.GetPriceHistoryEveryMinute(ctx(), "AAPL", &start, &end, nil, nil); err != nil {
		t.Fatal(err)
	}
	if cap.Path != "/marketdata/v1/pricehistory" {
		t.Errorf("path = %s", cap.Path)
	}
	checks := map[string]string{
		"symbol":        "AAPL",
		"periodType":    "day",
		"period":        "1",
		"frequencyType": "minute",
		"frequency":     "1",
		"startDate":     strconv.FormatInt(start.UnixMilli(), 10),
		"endDate":       strconv.FormatInt(end.UnixMilli(), 10),
	}
	for k, want := range checks {
		if got := cap.Query.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestPriceHistoryVariantDefaults(t *testing.T) {
	c, cap := newTestClient(t, 200, `{}`, nil)
	if _, err := c.GetPriceHistoryEveryDay(ctx(), "AAPL", nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	// schwab-py's variants default the range to start=1971-01-01,
	// end=now+7d, and every_day uses periodType=year, period=20.
	if got := cap.Query.Get("periodType"); got != "year" {
		t.Errorf("periodType = %q", got)
	}
	if got := cap.Query.Get("period"); got != "20" {
		t.Errorf("period = %q", got)
	}
	wantStart := strconv.FormatInt(
		time.Date(1971, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(), 10)
	if got := cap.Query.Get("startDate"); got != wantStart {
		t.Errorf("startDate = %q, want %q", got, wantStart)
	}
	endMs, err := strconv.ParseInt(cap.Query.Get("endDate"), 10, 64)
	if err != nil {
		t.Fatalf("endDate: %v", err)
	}
	if time.UnixMilli(endMs).Before(time.Now().Add(6 * 24 * time.Hour)) {
		t.Errorf("endDate %d is not ~now+7d", endMs)
	}
}

func TestGetMovers(t *testing.T) {
	c, cap := newTestClient(t, 200, `{}`, nil)
	sortOrder := MoversSortOrderPercentChangeUp
	freq := MoversFrequencyOne
	if _, err := c.GetMovers(ctx(), MoversIndexSPX, &sortOrder, &freq); err != nil {
		t.Fatal(err)
	}
	if cap.Path != "/marketdata/v1/movers/$SPX" {
		t.Errorf("path = %s", cap.Path)
	}
	if cap.Query.Get("sort") != "PERCENT_CHANGE_UP" ||
		cap.Query.Get("frequency") != "1" {
		t.Errorf("query = %v", cap.Query)
	}
	if _, err := c.GetMovers(ctx(), MoversIndex("BOGUS"), nil, nil); err == nil {
		t.Error("expected error for invalid index")
	}
}

func TestGetMarketHours(t *testing.T) {
	c, cap := newTestClient(t, 200, `{}`, nil)
	date := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	markets := []MarketHoursMarket{MarketHoursMarketEquity, MarketHoursMarketOption}
	if _, err := c.GetMarketHours(ctx(), markets, &date); err != nil {
		t.Fatal(err)
	}
	if cap.Path != "/marketdata/v1/markets" {
		t.Errorf("path = %s", cap.Path)
	}
	if got := cap.Query.Get("markets"); got != "equity,option" {
		t.Errorf("markets = %q", got)
	}
	if got := cap.Query.Get("date"); got != "2026-10-21" {
		t.Errorf("date = %q", got)
	}
}

func TestGetInstruments(t *testing.T) {
	c, cap := newTestClient(t, 200, `{}`, nil)
	if _, err := c.GetInstruments(ctx(), []string{"AAPL", "MSFT"},
		InstrumentProjectionFundamental); err != nil {
		t.Fatal(err)
	}
	if cap.Path != "/marketdata/v1/instruments" ||
		cap.Query.Get("symbol") != "AAPL,MSFT" ||
		cap.Query.Get("projection") != "fundamental" {
		t.Errorf("%s %v", cap.Path, cap.Query)
	}
	if _, err := c.GetInstrumentByCusip(ctx(), "037833100"); err != nil {
		t.Fatal(err)
	}
	if cap.Path != "/marketdata/v1/instruments/037833100" {
		t.Errorf("path = %s", cap.Path)
	}
}

func TestGetUserPreferencesPath(t *testing.T) {
	c, cap := newTestClient(t, 200, `{}`, nil)
	if _, err := c.GetUserPreferences(ctx()); err != nil {
		t.Fatal(err)
	}
	if cap.Path != "/trader/v1/userPreference" {
		t.Errorf("path = %s", cap.Path)
	}
}

func TestInvalidEnumRejected(t *testing.T) {
	c, _ := newTestClient(t, 200, `{}`, nil)
	_, err := c.GetQuote(ctx(), "AAPL", []QuoteFields{QuoteFields("bogus")})
	if err == nil {
		t.Fatal("expected error for invalid quote field")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should name the bad value: %v", err)
	}
}
