// Package orders ports the order-related enumerations from schwab-py's
// schwab/orders/common.py. Each Go string type carries exactly the wire
// values of its Python counterpart; because they are string types they
// marshal to JSON as those values, so order specifications built with them
// serialize identically to schwab-py's order builders.
package orders

// Duration — schwab-py: orders.common.Duration
type Duration string

const (
	DurationDay               Duration = "DAY"
	DurationGoodTillCancel    Duration = "GOOD_TILL_CANCEL"
	DurationFillOrKill        Duration = "FILL_OR_KILL"
	DurationImmediateOrCancel Duration = "IMMEDIATE_OR_CANCEL"
	DurationEndOfWeek         Duration = "END_OF_WEEK"
	DurationEndOfMonth        Duration = "END_OF_MONTH"
	DurationNextEndOfMonth    Duration = "NEXT_END_OF_MONTH"
)

// Session — schwab-py: orders.common.Session
type Session string

const (
	SessionNormal   Session = "NORMAL"
	SessionAM       Session = "AM"
	SessionPM       Session = "PM"
	SessionSeamless Session = "SEAMLESS"
)

// OrderType — schwab-py: orders.common.OrderType
type OrderType string

const (
	OrderTypeMarket            OrderType = "MARKET"
	OrderTypeLimit             OrderType = "LIMIT"
	OrderTypeStop              OrderType = "STOP"
	OrderTypeStopLimit         OrderType = "STOP_LIMIT"
	OrderTypeTrailingStop      OrderType = "TRAILING_STOP"
	OrderTypeCabinet           OrderType = "CABINET"
	OrderTypeNonMarketable     OrderType = "NON_MARKETABLE"
	OrderTypeMarketOnClose     OrderType = "MARKET_ON_CLOSE"
	OrderTypeExercise          OrderType = "EXERCISE"
	OrderTypeTrailingStopLimit OrderType = "TRAILING_STOP_LIMIT"
	OrderTypeNetDebit          OrderType = "NET_DEBIT"
	OrderTypeNetCredit         OrderType = "NET_CREDIT"
	OrderTypeNetZero           OrderType = "NET_ZERO"
	OrderTypeLimitOnClose      OrderType = "LIMIT_ON_CLOSE"
)

// ComplexOrderStrategyType — schwab-py: orders.common.ComplexOrderStrategyType
type ComplexOrderStrategyType string

const (
	ComplexOrderStrategyNone                 ComplexOrderStrategyType = "NONE"
	ComplexOrderStrategyCovered              ComplexOrderStrategyType = "COVERED"
	ComplexOrderStrategyVertical             ComplexOrderStrategyType = "VERTICAL"
	ComplexOrderStrategyBackRatio            ComplexOrderStrategyType = "BACK_RATIO"
	ComplexOrderStrategyCalendar             ComplexOrderStrategyType = "CALENDAR"
	ComplexOrderStrategyDiagonal             ComplexOrderStrategyType = "DIAGONAL"
	ComplexOrderStrategyStraddle             ComplexOrderStrategyType = "STRADDLE"
	ComplexOrderStrategyStrangle             ComplexOrderStrategyType = "STRANGLE"
	ComplexOrderStrategyCollarSynthetic      ComplexOrderStrategyType = "COLLAR_SYNTHETIC"
	ComplexOrderStrategyButterfly            ComplexOrderStrategyType = "BUTTERFLY"
	ComplexOrderStrategyCondor               ComplexOrderStrategyType = "CONDOR"
	ComplexOrderStrategyIronCondor           ComplexOrderStrategyType = "IRON_CONDOR"
	ComplexOrderStrategyVerticalRoll         ComplexOrderStrategyType = "VERTICAL_ROLL"
	ComplexOrderStrategyCollarWithStock      ComplexOrderStrategyType = "COLLAR_WITH_STOCK"
	ComplexOrderStrategyDoubleDiagonal       ComplexOrderStrategyType = "DOUBLE_DIAGONAL"
	ComplexOrderStrategyUnbalancedButterfly  ComplexOrderStrategyType = "UNBALANCED_BUTTERFLY"
	ComplexOrderStrategyUnbalancedCondor     ComplexOrderStrategyType = "UNBALANCED_CONDOR"
	ComplexOrderStrategyUnbalancedIronCondor ComplexOrderStrategyType = "UNBALANCED_IRON_CONDOR"
	ComplexOrderStrategyUnbalancedVertRoll   ComplexOrderStrategyType = "UNBALANCED_VERTICAL_ROLL"
	ComplexOrderStrategyMutualFundSwap       ComplexOrderStrategyType = "MUTUAL_FUND_SWAP"
	ComplexOrderStrategyCustom               ComplexOrderStrategyType = "CUSTOM"
)

// Destination — schwab-py: orders.common.Destination
type Destination string

const (
	DestinationINET    Destination = "INET"
	DestinationECNArca Destination = "ECN_ARCA"
	DestinationCBOE    Destination = "CBOE"
	DestinationAMEX    Destination = "AMEX"
	DestinationPHLX    Destination = "PHLX"
	DestinationISE     Destination = "ISE"
	DestinationBOX     Destination = "BOX"
	DestinationNYSE    Destination = "NYSE"
	DestinationNASDAQ  Destination = "NASDAQ"
	DestinationBATS    Destination = "BATS"
	DestinationC2      Destination = "C2"
	DestinationAuto    Destination = "AUTO"
)

// StopPriceLinkBasis — schwab-py: orders.common.StopPriceLinkBasis
type StopPriceLinkBasis string

const (
	StopPriceLinkBasisManual  StopPriceLinkBasis = "MANUAL"
	StopPriceLinkBasisBase    StopPriceLinkBasis = "BASE"
	StopPriceLinkBasisTrigger StopPriceLinkBasis = "TRIGGER"
	StopPriceLinkBasisLast    StopPriceLinkBasis = "LAST"
	StopPriceLinkBasisBid     StopPriceLinkBasis = "BID"
	StopPriceLinkBasisAsk     StopPriceLinkBasis = "ASK"
	StopPriceLinkBasisAskBid  StopPriceLinkBasis = "ASK_BID"
	StopPriceLinkBasisMark    StopPriceLinkBasis = "MARK"
	StopPriceLinkBasisAverage StopPriceLinkBasis = "AVERAGE"
)

// StopPriceLinkType — schwab-py: orders.common.StopPriceLinkType
type StopPriceLinkType string

const (
	StopPriceLinkTypeValue   StopPriceLinkType = "VALUE"
	StopPriceLinkTypePercent StopPriceLinkType = "PERCENT"
	StopPriceLinkTypeTick    StopPriceLinkType = "TICK"
)

// StopType — schwab-py: orders.common.StopType
type StopType string

const (
	StopTypeStandard StopType = "STANDARD"
	StopTypeBid      StopType = "BID"
	StopTypeAsk      StopType = "ASK"
	StopTypeLast     StopType = "LAST"
	StopTypeMark     StopType = "MARK"
)

// PriceLinkBasis — schwab-py: orders.common.PriceLinkBasis
type PriceLinkBasis string

const (
	PriceLinkBasisManual  PriceLinkBasis = "MANUAL"
	PriceLinkBasisBase    PriceLinkBasis = "BASE"
	PriceLinkBasisTrigger PriceLinkBasis = "TRIGGER"
	PriceLinkBasisLast    PriceLinkBasis = "LAST"
	PriceLinkBasisBid     PriceLinkBasis = "BID"
	PriceLinkBasisAsk     PriceLinkBasis = "ASK"
	PriceLinkBasisAskBid  PriceLinkBasis = "ASK_BID"
	PriceLinkBasisMark    PriceLinkBasis = "MARK"
	PriceLinkBasisAverage PriceLinkBasis = "AVERAGE"
)

// PriceLinkType — schwab-py: orders.common.PriceLinkType
type PriceLinkType string

const (
	PriceLinkTypeValue   PriceLinkType = "VALUE"
	PriceLinkTypePercent PriceLinkType = "PERCENT"
	PriceLinkTypeTick    PriceLinkType = "TICK"
)

// TaxLotMethod — schwab-py: orders.common.TaxLotMethod
type TaxLotMethod string

const (
	TaxLotMethodFIFO          TaxLotMethod = "FIFO"
	TaxLotMethodLIFO          TaxLotMethod = "LIFO"
	TaxLotMethodHighCost      TaxLotMethod = "HIGH_COST"
	TaxLotMethodLowCost       TaxLotMethod = "LOW_COST"
	TaxLotMethodAverageCost   TaxLotMethod = "AVERAGE_COST"
	TaxLotMethodSpecificLot   TaxLotMethod = "SPECIFIC_LOT"
	TaxLotMethodLossHarvester TaxLotMethod = "LOSS_HARVESTER"
)

// EquityInstruction — schwab-py: orders.common.EquityInstruction
type EquityInstruction string

const (
	EquityInstructionBuy        EquityInstruction = "BUY"
	EquityInstructionSell       EquityInstruction = "SELL"
	EquityInstructionSellShort  EquityInstruction = "SELL_SHORT"
	EquityInstructionBuyToCover EquityInstruction = "BUY_TO_COVER"
)

// OptionInstruction — schwab-py: orders.common.OptionInstruction
type OptionInstruction string

const (
	OptionInstructionBuyToOpen   OptionInstruction = "BUY_TO_OPEN"
	OptionInstructionSellToClose OptionInstruction = "SELL_TO_CLOSE"
	OptionInstructionSellToOpen  OptionInstruction = "SELL_TO_OPEN"
	OptionInstructionBuyToClose  OptionInstruction = "BUY_TO_CLOSE"
)

// SpecialInstruction — schwab-py: orders.common.SpecialInstruction
type SpecialInstruction string

const (
	SpecialInstructionAllOrNone            SpecialInstruction = "ALL_OR_NONE"
	SpecialInstructionDoNotReduce          SpecialInstruction = "DO_NOT_REDUCE"
	SpecialInstructionAllOrNoneDoNotReduce SpecialInstruction = "ALL_OR_NONE_DO_NOT_REDUCE"
)

// OrderStrategyType — schwab-py: orders.common.OrderStrategyType
type OrderStrategyType string

const (
	OrderStrategyTypeSingle     OrderStrategyType = "SINGLE"
	OrderStrategyTypeCancel     OrderStrategyType = "CANCEL"
	OrderStrategyTypeRecall     OrderStrategyType = "RECALL"
	OrderStrategyTypePair       OrderStrategyType = "PAIR"
	OrderStrategyTypeFlatten    OrderStrategyType = "FLATTEN"
	OrderStrategyTypeTwoDaySwap OrderStrategyType = "TWO_DAY_SWAP"
	OrderStrategyTypeBlastAll   OrderStrategyType = "BLAST_ALL"
	OrderStrategyTypeOCO        OrderStrategyType = "OCO"
	OrderStrategyTypeTrigger    OrderStrategyType = "TRIGGER"
)

// AssetType values used in order instrument specifications (the assetType
// field emitted by schwab-py's EquityInstrument/OptionInstrument builders).
type AssetType string

const (
	AssetTypeEquity         AssetType = "EQUITY"
	AssetTypeOption         AssetType = "OPTION"
	AssetTypeIndex          AssetType = "INDEX"
	AssetTypeBond           AssetType = "BOND"
	AssetTypeMutualFund     AssetType = "MUTUAL_FUND"
	AssetTypeCashEquivalent AssetType = "CASH_EQUIVALENT"
	AssetTypeCurrency       AssetType = "CURRENCY"
	AssetTypeFuture         AssetType = "FUTURE"
	AssetTypeForex          AssetType = "FOREX"
	AssetTypeProduct        AssetType = "PRODUCT"
	AssetTypeUnknown        AssetType = "UNKNOWN"
)
