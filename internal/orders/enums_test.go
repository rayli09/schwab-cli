package orders

import "testing"

// These tests pin the wire values of the order enums to schwab-py's
// schwab/orders/common.py. A full machine-checked comparison of every enum
// in schwab-py against this package was performed during the port; these
// cases guard the values most likely to be typo'd.
func TestOrderEnumWireValues(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"Duration DAY", string(DurationDay), "DAY"},
		{"Duration GTC", string(DurationGoodTillCancel), "GOOD_TILL_CANCEL"},
		{"Session AM", string(SessionAM), "AM"},
		{"OrderType NET_DEBIT", string(OrderTypeNetDebit), "NET_DEBIT"},
		{"OrderType STOP_LIMIT", string(OrderTypeStopLimit), "STOP_LIMIT"},
		{"ComplexOrderStrategyType VERTICAL", string(ComplexOrderStrategyVertical), "VERTICAL"},
		{"ComplexOrderStrategyType IRON_CONDOR", string(ComplexOrderStrategyIronCondor), "IRON_CONDOR"},
		{"Destination C2", string(DestinationC2), "C2"},
		{"Destination NYSE", string(DestinationNYSE), "NYSE"},
		{"EquityInstruction BUY_TO_COVER", string(EquityInstructionBuyToCover), "BUY_TO_COVER"},
		{"OptionInstruction SELL_TO_OPEN", string(OptionInstructionSellToOpen), "SELL_TO_OPEN"},
		{"OrderStrategyType OCO", string(OrderStrategyTypeOCO), "OCO"},
		{"OrderStrategyType TRIGGER", string(OrderStrategyTypeTrigger), "TRIGGER"},
		{"AssetType MUTUAL_FUND", string(AssetTypeMutualFund), "MUTUAL_FUND"},
		{"TaxLotMethod LIFO", string(TaxLotMethodLIFO), "LIFO"},
		{"StopType BID", string(StopTypeBid), "BID"},
		{"SpecialInstruction ALL_OR_NONE", string(SpecialInstructionAllOrNone), "ALL_OR_NONE"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}
