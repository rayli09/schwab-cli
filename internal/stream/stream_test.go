package stream

import "testing"

func TestAllFieldIDs(t *testing.T) {
	if got := AllFieldIDs(3); got != "0,1,2,3" {
		t.Errorf("AllFieldIDs(3) = %q", got)
	}
	if got := AllFieldIDs(0); got != "0" {
		t.Errorf("AllFieldIDs(0) = %q", got)
	}
}

// TestFeedsMatchSchwabPy pins the streaming feed table to the subscription
// surface of schwab-py's StreamClient: service names and highest field ids
// (field ranges there are contiguous from 0).
func TestFeedsMatchSchwabPy(t *testing.T) {
	want := map[string]struct {
		service  string
		maxField int
	}{
		"account-activity":          {"ACCT_ACTIVITY", 3},
		"chart-equity":              {"CHART_EQUITY", 8},
		"chart-futures":             {"CHART_FUTURES", 6},
		"level-one-equity":          {"LEVELONE_EQUITIES", 51},
		"level-one-option":          {"LEVELONE_OPTIONS", 55},
		"level-one-futures":         {"LEVELONE_FUTURES", 40},
		"level-one-forex":           {"LEVELONE_FOREX", 29},
		"level-one-futures-options": {"LEVELONE_FUTURES_OPTIONS", 31},
		"nyse-book":                 {"NYSE_BOOK", 3},
		"nasdaq-book":               {"NASDAQ_BOOK", 3},
		"options-book":              {"OPTIONS_BOOK", 3},
		"screener-equity":           {"SCREENER_EQUITY", 4},
		"screener-option":           {"SCREENER_OPTION", 4},
	}
	if len(Feeds) != len(want) {
		t.Errorf("Feeds has %d entries, want %d", len(Feeds), len(want))
	}
	for name, w := range want {
		feed, ok := Feeds[name]
		if !ok {
			t.Errorf("missing feed %q", name)
			continue
		}
		if feed.Service != w.service || feed.MaxField != w.maxField {
			t.Errorf("feed %q = {%s, %d}, want {%s, %d}",
				name, feed.Service, feed.MaxField, w.service, w.maxField)
		}
		if !feed.HasFields {
			t.Errorf("feed %q should carry a fields parameter", name)
		}
	}
}
