package domain

// Trade sides
const (
	SideBuy  = "BUY"
	SideSell = "SELL"
)

// Trade statuses
const (
	StatusPending   = "PENDING"
	StatusPlaced    = "PLACED"
	StatusFilled    = "FILLED"
	StatusCancelled = "CANCELLED"
)

// Strategy types
const (
	StrategyDCA      = "DCA"
	StrategyGrid     = "GRID"
	StrategyHybrid   = "HYBRID"
	StrategyAutoSell = "AUTO_SELL"
	StrategyManual   = "MANUAL"
)

// Snapshot types for PnL history
const (
	SnapshotHourly  = "HOURLY"
	SnapshotDaily   = "DAILY"
	SnapshotWeekly  = "WEEKLY"
	SnapshotMonthly = "MONTHLY"
)

// Log levels
const (
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
)

// Special symbols
const (
	SymbolAll = "ALL"
)

// Order types
const (
	OrderTypeMarket = "Market"
	OrderTypeLimit  = "Limit"
)

// Bybit constants
const (
	BybitCategorySpot   = "spot"
	BybitAccountUnified = "UNIFIED"
	BybitRecvWindow     = "5000"
)
