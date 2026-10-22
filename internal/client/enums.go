package client

// This file ports the enum classes nested in schwab-py's BaseClient
// (schwab/client/base.py). Each Go string type carries exactly the wire
// values of its Python counterpart, and a Valid method mirroring the enum
// membership check schwab-py performs via EnumEnforcer.

// AccountFields — schwab-py: BaseClient.Account.Fields
type AccountFields string

const (
	AccountFieldsPositions AccountFields = "positions"
)

func (e AccountFields) Valid() bool { return e == AccountFieldsPositions }

// OrderStatus — schwab-py: BaseClient.Order.Status
type OrderStatus string

const (
	OrderStatusAwaitingParentOrder    OrderStatus = "AWAITING_PARENT_ORDER"
	OrderStatusAwaitingCondition      OrderStatus = "AWAITING_CONDITION"
	OrderStatusAwaitingStopCondition  OrderStatus = "AWAITING_STOP_CONDITION"
	OrderStatusAwaitingManualReview   OrderStatus = "AWAITING_MANUAL_REVIEW"
	OrderStatusAccepted               OrderStatus = "ACCEPTED"
	OrderStatusAwaitingUROut          OrderStatus = "AWAITING_UR_OUT"
	OrderStatusPendingActivation      OrderStatus = "PENDING_ACTIVATION"
	OrderStatusQueued                 OrderStatus = "QUEUED"
	OrderStatusWorking                OrderStatus = "WORKING"
	OrderStatusRejected               OrderStatus = "REJECTED"
	OrderStatusPendingCancel          OrderStatus = "PENDING_CANCEL"
	OrderStatusCanceled               OrderStatus = "CANCELED"
	OrderStatusPendingReplace         OrderStatus = "PENDING_REPLACE"
	OrderStatusReplaced               OrderStatus = "REPLACED"
	OrderStatusFilled                 OrderStatus = "FILLED"
	OrderStatusExpired                OrderStatus = "EXPIRED"
	OrderStatusNew                    OrderStatus = "NEW"
	OrderStatusAwaitingReleaseTime    OrderStatus = "AWAITING_RELEASE_TIME"
	OrderStatusPendingAcknowledgement OrderStatus = "PENDING_ACKNOWLEDGEMENT"
	OrderStatusPendingRecall          OrderStatus = "PENDING_RECALL"
	OrderStatusUnknown                OrderStatus = "UNKNOWN"
)

var orderStatuses = map[OrderStatus]bool{
	OrderStatusAwaitingParentOrder: true, OrderStatusAwaitingCondition: true,
	OrderStatusAwaitingStopCondition: true, OrderStatusAwaitingManualReview: true,
	OrderStatusAccepted: true, OrderStatusAwaitingUROut: true,
	OrderStatusPendingActivation: true, OrderStatusQueued: true,
	OrderStatusWorking: true, OrderStatusRejected: true,
	OrderStatusPendingCancel: true, OrderStatusCanceled: true,
	OrderStatusPendingReplace: true, OrderStatusReplaced: true,
	OrderStatusFilled: true, OrderStatusExpired: true, OrderStatusNew: true,
	OrderStatusAwaitingReleaseTime:    true,
	OrderStatusPendingAcknowledgement: true, OrderStatusPendingRecall: true,
	OrderStatusUnknown: true,
}

func (e OrderStatus) Valid() bool { return orderStatuses[e] }

// TransactionType — schwab-py: BaseClient.Transactions.TransactionType
type TransactionType string

const (
	TransactionTypeTrade              TransactionType = "TRADE"
	TransactionTypeReceiveAndDeliver  TransactionType = "RECEIVE_AND_DELIVER"
	TransactionTypeDividendOrInterest TransactionType = "DIVIDEND_OR_INTEREST"
	TransactionTypeACHReceipt         TransactionType = "ACH_RECEIPT"
	TransactionTypeACHDisbursement    TransactionType = "ACH_DISBURSEMENT"
	TransactionTypeCashReceipt        TransactionType = "CASH_RECEIPT"
	TransactionTypeCashDisbursement   TransactionType = "CASH_DISBURSEMENT"
	TransactionTypeElectronicFund     TransactionType = "ELECTRONIC_FUND"
	TransactionTypeWireOut            TransactionType = "WIRE_OUT"
	TransactionTypeWireIn             TransactionType = "WIRE_IN"
	TransactionTypeJournal            TransactionType = "JOURNAL"
	TransactionTypeMemorandum         TransactionType = "MEMORANDUM"
	TransactionTypeMarginCall         TransactionType = "MARGIN_CALL"
	TransactionTypeMoneyMarket        TransactionType = "MONEY_MARKET"
	TransactionTypeSMAAdjustment      TransactionType = "SMA_ADJUSTMENT"
)

// AllTransactionTypes lists every transaction type, in the order schwab-py
// uses when defaulting the types parameter of get_transactions.
func AllTransactionTypes() []TransactionType {
	return []TransactionType{
		TransactionTypeTrade, TransactionTypeReceiveAndDeliver,
		TransactionTypeDividendOrInterest, TransactionTypeACHReceipt,
		TransactionTypeACHDisbursement, TransactionTypeCashReceipt,
		TransactionTypeCashDisbursement, TransactionTypeElectronicFund,
		TransactionTypeWireOut, TransactionTypeWireIn, TransactionTypeJournal,
		TransactionTypeMemorandum, TransactionTypeMarginCall,
		TransactionTypeMoneyMarket, TransactionTypeSMAAdjustment,
	}
}

var transactionTypes = func() map[TransactionType]bool {
	m := map[TransactionType]bool{}
	for _, t := range AllTransactionTypes() {
		m[t] = true
	}
	return m
}()

func (e TransactionType) Valid() bool { return transactionTypes[e] }

// QuoteFields — schwab-py: BaseClient.Quote.Fields
type QuoteFields string

const (
	QuoteFieldsQuote       QuoteFields = "quote"
	QuoteFieldsFundamental QuoteFields = "fundamental"
	QuoteFieldsExtended    QuoteFields = "extended"
	QuoteFieldsReference   QuoteFields = "reference"
	QuoteFieldsRegular     QuoteFields = "regular"
)

var quoteFields = map[QuoteFields]bool{
	QuoteFieldsQuote: true, QuoteFieldsFundamental: true,
	QuoteFieldsExtended: true, QuoteFieldsReference: true,
	QuoteFieldsRegular: true,
}

func (e QuoteFields) Valid() bool { return quoteFields[e] }

// OptionsContractType — schwab-py: BaseClient.Options.ContractType
type OptionsContractType string

const (
	OptionsContractTypeCall OptionsContractType = "CALL"
	OptionsContractTypePut  OptionsContractType = "PUT"
	OptionsContractTypeAll  OptionsContractType = "ALL"
)

func (e OptionsContractType) Valid() bool {
	return e == OptionsContractTypeCall || e == OptionsContractTypePut ||
		e == OptionsContractTypeAll
}

// OptionsStrategy — schwab-py: BaseClient.Options.Strategy
type OptionsStrategy string

const (
	OptionsStrategySingle     OptionsStrategy = "SINGLE"
	OptionsStrategyAnalytical OptionsStrategy = "ANALYTICAL"
	OptionsStrategyCovered    OptionsStrategy = "COVERED"
	OptionsStrategyVertical   OptionsStrategy = "VERTICAL"
	OptionsStrategyCalendar   OptionsStrategy = "CALENDAR"
	OptionsStrategyStrangle   OptionsStrategy = "STRANGLE"
	OptionsStrategyStraddle   OptionsStrategy = "STRADDLE"
	OptionsStrategyButterfly  OptionsStrategy = "BUTTERFLY"
	OptionsStrategyCondor     OptionsStrategy = "CONDOR"
	OptionsStrategyDiagonal   OptionsStrategy = "DIAGONAL"
	OptionsStrategyCollar     OptionsStrategy = "COLLAR"
	OptionsStrategyRoll       OptionsStrategy = "ROLL"
)

var optionsStrategies = map[OptionsStrategy]bool{
	OptionsStrategySingle: true, OptionsStrategyAnalytical: true,
	OptionsStrategyCovered: true, OptionsStrategyVertical: true,
	OptionsStrategyCalendar: true, OptionsStrategyStrangle: true,
	OptionsStrategyStraddle: true, OptionsStrategyButterfly: true,
	OptionsStrategyCondor: true, OptionsStrategyDiagonal: true,
	OptionsStrategyCollar: true, OptionsStrategyRoll: true,
}

func (e OptionsStrategy) Valid() bool { return optionsStrategies[e] }

// OptionsStrikeRange — schwab-py: BaseClient.Options.StrikeRange
type OptionsStrikeRange string

const (
	OptionsStrikeRangeInTheMoney      OptionsStrikeRange = "ITM"
	OptionsStrikeRangeNearTheMoney    OptionsStrikeRange = "NTM"
	OptionsStrikeRangeOutOfTheMoney   OptionsStrikeRange = "OTM"
	OptionsStrikeRangeStrikesAboveMkt OptionsStrikeRange = "SAK"
	OptionsStrikeRangeStrikesBelowMkt OptionsStrikeRange = "SBK"
	OptionsStrikeRangeStrikesNearMkt  OptionsStrikeRange = "SNK"
	OptionsStrikeRangeAll             OptionsStrikeRange = "ALL"
)

func (e OptionsStrikeRange) Valid() bool {
	switch e {
	case OptionsStrikeRangeInTheMoney, OptionsStrikeRangeNearTheMoney,
		OptionsStrikeRangeOutOfTheMoney, OptionsStrikeRangeStrikesAboveMkt,
		OptionsStrikeRangeStrikesBelowMkt, OptionsStrikeRangeStrikesNearMkt,
		OptionsStrikeRangeAll:
		return true
	}
	return false
}

// OptionsType — schwab-py: BaseClient.Options.Type
type OptionsType string

const (
	OptionsTypeStandard    OptionsType = "S"
	OptionsTypeNonStandard OptionsType = "NS"
	OptionsTypeAll         OptionsType = "ALL"
)

func (e OptionsType) Valid() bool {
	return e == OptionsTypeStandard || e == OptionsTypeNonStandard ||
		e == OptionsTypeAll
}

// OptionsExpirationMonth — schwab-py: BaseClient.Options.ExpirationMonth
type OptionsExpirationMonth string

const (
	OptionsExpirationMonthJanuary   OptionsExpirationMonth = "JAN"
	OptionsExpirationMonthFebruary  OptionsExpirationMonth = "FEB"
	OptionsExpirationMonthMarch     OptionsExpirationMonth = "MAR"
	OptionsExpirationMonthApril     OptionsExpirationMonth = "APR"
	OptionsExpirationMonthMay       OptionsExpirationMonth = "MAY"
	OptionsExpirationMonthJune      OptionsExpirationMonth = "JUN"
	OptionsExpirationMonthJuly      OptionsExpirationMonth = "JUL"
	OptionsExpirationMonthAugust    OptionsExpirationMonth = "AUG"
	OptionsExpirationMonthSeptember OptionsExpirationMonth = "SEP"
	OptionsExpirationMonthOctober   OptionsExpirationMonth = "OCT"
	OptionsExpirationMonthNovember  OptionsExpirationMonth = "NOV"
	OptionsExpirationMonthDecember  OptionsExpirationMonth = "DEC"
	OptionsExpirationMonthAll       OptionsExpirationMonth = "ALL"
)

func (e OptionsExpirationMonth) Valid() bool {
	switch e {
	case OptionsExpirationMonthJanuary, OptionsExpirationMonthFebruary,
		OptionsExpirationMonthMarch, OptionsExpirationMonthApril,
		OptionsExpirationMonthMay, OptionsExpirationMonthJune,
		OptionsExpirationMonthJuly, OptionsExpirationMonthAugust,
		OptionsExpirationMonthSeptember, OptionsExpirationMonthOctober,
		OptionsExpirationMonthNovember, OptionsExpirationMonthDecember,
		OptionsExpirationMonthAll:
		return true
	}
	return false
}

// OptionsEntitlement — schwab-py: BaseClient.Options.Entitlement
type OptionsEntitlement string

const (
	OptionsEntitlementPayingPro    OptionsEntitlement = "PP"
	OptionsEntitlementNonPro       OptionsEntitlement = "NP"
	OptionsEntitlementNonPayingPro OptionsEntitlement = "PN"
)

func (e OptionsEntitlement) Valid() bool {
	return e == OptionsEntitlementPayingPro || e == OptionsEntitlementNonPro ||
		e == OptionsEntitlementNonPayingPro
}

// PriceHistoryPeriodType — schwab-py: BaseClient.PriceHistory.PeriodType
type PriceHistoryPeriodType string

const (
	PriceHistoryPeriodTypeDay        PriceHistoryPeriodType = "day"
	PriceHistoryPeriodTypeMonth      PriceHistoryPeriodType = "month"
	PriceHistoryPeriodTypeYear       PriceHistoryPeriodType = "year"
	PriceHistoryPeriodTypeYearToDate PriceHistoryPeriodType = "ytd"
)

func (e PriceHistoryPeriodType) Valid() bool {
	switch e {
	case PriceHistoryPeriodTypeDay, PriceHistoryPeriodTypeMonth,
		PriceHistoryPeriodTypeYear, PriceHistoryPeriodTypeYearToDate:
		return true
	}
	return false
}

// PriceHistoryPeriod — schwab-py: BaseClient.PriceHistory.Period.
// The valid values depend on the period type; the full set of named values
// is: days 1,2,3,4,5,10; months 1,2,3,6; years 1,2,3,5,10,15,20; ytd 1.
type PriceHistoryPeriod int

const (
	PriceHistoryPeriodOneDay      PriceHistoryPeriod = 1
	PriceHistoryPeriodTwoDays     PriceHistoryPeriod = 2
	PriceHistoryPeriodThreeDays   PriceHistoryPeriod = 3
	PriceHistoryPeriodFourDays    PriceHistoryPeriod = 4
	PriceHistoryPeriodFiveDays    PriceHistoryPeriod = 5
	PriceHistoryPeriodTenDays     PriceHistoryPeriod = 10
	PriceHistoryPeriodOneMonth    PriceHistoryPeriod = 1
	PriceHistoryPeriodTwoMonths   PriceHistoryPeriod = 2
	PriceHistoryPeriodThreeMonths PriceHistoryPeriod = 3
	PriceHistoryPeriodSixMonths   PriceHistoryPeriod = 6
	PriceHistoryPeriodOneYear     PriceHistoryPeriod = 1
	PriceHistoryPeriodTwoYears    PriceHistoryPeriod = 2
	PriceHistoryPeriodThreeYears  PriceHistoryPeriod = 3
	PriceHistoryPeriodFiveYears   PriceHistoryPeriod = 5
	PriceHistoryPeriodTenYears    PriceHistoryPeriod = 10
	PriceHistoryPeriodFifteenYrs  PriceHistoryPeriod = 15
	PriceHistoryPeriodTwentyYears PriceHistoryPeriod = 20
	PriceHistoryPeriodYearToDate  PriceHistoryPeriod = 1
)

// Valid reports whether v is one of the period values named by schwab-py's
// PriceHistory.Period enum.
func (e PriceHistoryPeriod) Valid() bool {
	switch int(e) {
	case 1, 2, 3, 4, 5, 6, 10, 15, 20:
		return true
	}
	return false
}

// PriceHistoryFrequencyType — schwab-py: BaseClient.PriceHistory.FrequencyType
type PriceHistoryFrequencyType string

const (
	PriceHistoryFrequencyTypeMinute  PriceHistoryFrequencyType = "minute"
	PriceHistoryFrequencyTypeDaily   PriceHistoryFrequencyType = "daily"
	PriceHistoryFrequencyTypeWeekly  PriceHistoryFrequencyType = "weekly"
	PriceHistoryFrequencyTypeMonthly PriceHistoryFrequencyType = "monthly"
)

func (e PriceHistoryFrequencyType) Valid() bool {
	switch e {
	case PriceHistoryFrequencyTypeMinute, PriceHistoryFrequencyTypeDaily,
		PriceHistoryFrequencyTypeWeekly, PriceHistoryFrequencyTypeMonthly:
		return true
	}
	return false
}

// PriceHistoryFrequency — schwab-py: BaseClient.PriceHistory.Frequency.
// Minute frequencies are 1, 5, 10, 15, 30; other frequency types use 1.
type PriceHistoryFrequency int

const (
	PriceHistoryFrequencyEveryMinute      PriceHistoryFrequency = 1
	PriceHistoryFrequencyEveryFiveMinutes PriceHistoryFrequency = 5
	PriceHistoryFrequencyEveryTenMinutes  PriceHistoryFrequency = 10
	PriceHistoryFrequencyEveryFifteenMin  PriceHistoryFrequency = 15
	PriceHistoryFrequencyEveryThirtyMin   PriceHistoryFrequency = 30
	PriceHistoryFrequencyDaily            PriceHistoryFrequency = 1
	PriceHistoryFrequencyWeekly           PriceHistoryFrequency = 1
	PriceHistoryFrequencyMonthly          PriceHistoryFrequency = 1
)

func (e PriceHistoryFrequency) Valid() bool {
	switch int(e) {
	case 1, 5, 10, 15, 30:
		return true
	}
	return false
}

// MoversIndex — schwab-py: BaseClient.Movers.Index
type MoversIndex string

const (
	MoversIndexDJI        MoversIndex = "$DJI"
	MoversIndexCOMPX      MoversIndex = "$COMPX"
	MoversIndexSPX        MoversIndex = "$SPX"
	MoversIndexNYSE       MoversIndex = "NYSE"
	MoversIndexNASDAQ     MoversIndex = "NASDAQ"
	MoversIndexOTCBB      MoversIndex = "OTCBB"
	MoversIndexIndexAll   MoversIndex = "INDEX_ALL"
	MoversIndexEquityAll  MoversIndex = "EQUITY_ALL"
	MoversIndexOptionAll  MoversIndex = "OPTION_ALL"
	MoversIndexOptionPut  MoversIndex = "OPTION_PUT"
	MoversIndexOptionCall MoversIndex = "OPTION_CALL"
)

func (e MoversIndex) Valid() bool {
	switch e {
	case MoversIndexDJI, MoversIndexCOMPX, MoversIndexSPX, MoversIndexNYSE,
		MoversIndexNASDAQ, MoversIndexOTCBB, MoversIndexIndexAll,
		MoversIndexEquityAll, MoversIndexOptionAll, MoversIndexOptionPut,
		MoversIndexOptionCall:
		return true
	}
	return false
}

// MoversSortOrder — schwab-py: BaseClient.Movers.SortOrder
type MoversSortOrder string

const (
	MoversSortOrderVolume          MoversSortOrder = "VOLUME"
	MoversSortOrderTrades          MoversSortOrder = "TRADES"
	MoversSortOrderPercentChangeUp MoversSortOrder = "PERCENT_CHANGE_UP"
	MoversSortOrderPercentChangeDn MoversSortOrder = "PERCENT_CHANGE_DOWN"
)

func (e MoversSortOrder) Valid() bool {
	switch e {
	case MoversSortOrderVolume, MoversSortOrderTrades,
		MoversSortOrderPercentChangeUp, MoversSortOrderPercentChangeDn:
		return true
	}
	return false
}

// MoversFrequency — schwab-py: BaseClient.Movers.Frequency
type MoversFrequency int

const (
	MoversFrequencyZero   MoversFrequency = 0
	MoversFrequencyOne    MoversFrequency = 1
	MoversFrequencyFive   MoversFrequency = 5
	MoversFrequencyTen    MoversFrequency = 10
	MoversFrequencyThirty MoversFrequency = 30
	MoversFrequencySixty  MoversFrequency = 60
)

func (e MoversFrequency) Valid() bool {
	switch int(e) {
	case 0, 1, 5, 10, 30, 60:
		return true
	}
	return false
}

// MarketHoursMarket — schwab-py: BaseClient.MarketHours.Market
type MarketHoursMarket string

const (
	MarketHoursMarketEquity MarketHoursMarket = "equity"
	MarketHoursMarketOption MarketHoursMarket = "option"
	MarketHoursMarketBond   MarketHoursMarket = "bond"
	MarketHoursMarketFuture MarketHoursMarket = "future"
	MarketHoursMarketForex  MarketHoursMarket = "forex"
)

func (e MarketHoursMarket) Valid() bool {
	switch e {
	case MarketHoursMarketEquity, MarketHoursMarketOption,
		MarketHoursMarketBond, MarketHoursMarketFuture, MarketHoursMarketForex:
		return true
	}
	return false
}

// InstrumentProjection — schwab-py: BaseClient.Instrument.Projection
type InstrumentProjection string

const (
	InstrumentProjectionSymbolSearch      InstrumentProjection = "symbol-search"
	InstrumentProjectionSymbolRegex       InstrumentProjection = "symbol-regex"
	InstrumentProjectionDescriptionSearch InstrumentProjection = "desc-search"
	InstrumentProjectionDescriptionRegex  InstrumentProjection = "desc-regex"
	InstrumentProjectionSearch            InstrumentProjection = "search"
	InstrumentProjectionFundamental       InstrumentProjection = "fundamental"
)

func (e InstrumentProjection) Valid() bool {
	switch e {
	case InstrumentProjectionSymbolSearch, InstrumentProjectionSymbolRegex,
		InstrumentProjectionDescriptionSearch,
		InstrumentProjectionDescriptionRegex, InstrumentProjectionSearch,
		InstrumentProjectionFundamental:
		return true
	}
	return false
}
