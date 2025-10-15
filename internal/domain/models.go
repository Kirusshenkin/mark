package domain

import "time"

// Trade представляет сделку (покупку или продажу)
type Trade struct {
	ID           int64     `db:"id"`
	Symbol       string    `db:"symbol"`
	Side         string    `db:"side"` // "BUY" or "SELL"
	Quantity     float64   `db:"quantity"`
	Price        float64   `db:"price"`
	Amount       float64   `db:"amount"`
	OrderID      string    `db:"order_id"`
	Status       string    `db:"status"`
	StrategyType string    `db:"strategy_type"` // "DCA", "GRID", "AUTO_SELL"
	GridLevel    int       `db:"grid_level"`    // для Grid стратегии
	CreatedAt    time.Time `db:"created_at"`
}

// Balance представляет баланс актива
type Balance struct {
	ID             int64     `db:"id"`
	Symbol         string    `db:"symbol"`
	TotalQuantity  float64   `db:"total_quantity"`
	AvailableQty   float64   `db:"available_qty"`
	AvgEntryPrice  float64   `db:"avg_entry_price"`
	TotalInvested  float64   `db:"total_invested"`
	TotalSold      float64   `db:"total_sold"`
	RealizedProfit float64   `db:"realized_profit"`
	UnrealizedPnL  float64   `db:"unrealized_pnl"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// Asset представляет торгуемый актив с его конфигурацией
type Asset struct {
	ID                     int64     `db:"id"`
	Symbol                 string    `db:"symbol"`
	Enabled                bool      `db:"enabled"`
	StrategyType           string    `db:"strategy_type"` // "DCA", "GRID", "HYBRID"
	AllocatedCapital       float64   `db:"allocated_capital"`
	MaxPositionSize        float64   `db:"max_position_size"`
	DCAAmount              float64   `db:"dca_amount"`
	DCAInterval            int       `db:"dca_interval_minutes"`
	AutoSellEnabled        bool      `db:"auto_sell_enabled"`
	AutoSellTriggerPercent float64   `db:"auto_sell_trigger_percent"`
	AutoSellAmountPercent  float64   `db:"auto_sell_amount_percent"`
	GridLevels             int       `db:"grid_levels"`
	GridSpacingPercent     float64   `db:"grid_spacing_percent"`
	GridOrderSize          float64   `db:"grid_order_size"`
	StopLossPercent        float64   `db:"stop_loss_percent"`
	TakeProfitPercent      float64   `db:"take_profit_percent"`
	CreatedAt              time.Time `db:"created_at"`
	UpdatedAt              time.Time `db:"updated_at"`
}

// GridOrder представляет ордер в Grid стратегии
type GridOrder struct {
	ID          int64     `db:"id"`
	Symbol      string    `db:"symbol"`
	Level       int       `db:"level"`
	Side        string    `db:"side"` // "BUY" or "SELL"
	Price       float64   `db:"price"`
	Quantity    float64   `db:"quantity"`
	OrderID     string    `db:"order_id"`
	Status      string    `db:"status"` // "PENDING", "PLACED", "FILLED", "CANCELLED"
	FilledQty   float64   `db:"filled_qty"`
	FilledPrice float64   `db:"filled_price"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// PnLHistory представляет снапшот PnL для аналитики
type PnLHistory struct {
	ID             int64     `db:"id"`
	Symbol         string    `db:"symbol"`
	RealizedPnL    float64   `db:"realized_pnl"`
	UnrealizedPnL  float64   `db:"unrealized_pnl"`
	TotalPnL       float64   `db:"total_pnl"`
	TotalInvested  float64   `db:"total_invested"`
	CurrentValue   float64   `db:"current_value"`
	ReturnPercent  float64   `db:"return_percent"`
	SnapshotType   string    `db:"snapshot_type"` // "HOURLY", "DAILY", "WEEKLY", "MONTHLY"
	Regime         string    `db:"regime"`        // Stage 4: AI regime
	AutoMode       bool      `db:"auto_mode"`     // Stage 4: was auto-trading active
	RiskScore      float64   `db:"risk_score"`    // Stage 4: risk score at snapshot time
	CreatedAt      time.Time `db:"created_at"`
}

// RiskLimit представляет лимиты риск-менеджмента
type RiskLimit struct {
	ID                  int64     `db:"id"`
	MaxDailyLoss        float64   `db:"max_daily_loss"`
	MaxTotalExposure    float64   `db:"max_total_exposure"`
	MaxPositionSizeUSD  float64   `db:"max_position_size_usd"`
	MaxOrderSizeUSD     float64   `db:"max_order_size_usd"`
	EnableEmergencyStop bool      `db:"enable_emergency_stop"`
	UpdatedAt           time.Time `db:"updated_at"`
}

// ConfigParam представляет параметр конфигурации стратегии
type ConfigParam struct {
	ID        int64     `db:"id"`
	Key       string    `db:"key"`
	Value     string    `db:"value"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Log представляет системное событие
type Log struct {
	ID        int64     `db:"id"`
	Level     string    `db:"level"` // "INFO", "WARN", "ERROR"
	Message   string    `db:"message"`
	Data      string    `db:"data"` // JSON
	CreatedAt time.Time `db:"created_at"`
}

// ==================== STAGE 4: Full-Auto Models ====================

// AIDecision представляет решение AI-ассистента
type AIDecision struct {
	ID              int64     `db:"id"`
	Timestamp       time.Time `db:"timestamp"`
	Regime          string    `db:"regime"` // ACCUMULATE, TREND_FOLLOW, RANGE_GRID, DEFENSE
	Confidence      float64   `db:"confidence"`
	Rationale       string    `db:"rationale"`
	RawResponse     string    `db:"raw_response"` // JSON
	Approved        bool      `db:"approved"`
	RejectionReason string    `db:"rejection_reason"`
	Mode            string    `db:"mode"` // shadow, pilot, full
}

// AIAction представляет действие, запланированное AI
type AIAction struct {
	ID                 int64     `db:"id"`
	DecisionID         int64     `db:"decision_id"`
	ActionType         string    `db:"action_type"` // set_dca, set_grid, set_autosell, rebalance, pause_strategy
	Symbol             string    `db:"symbol"`
	Parameters         string    `db:"parameters"` // JSON
	Status             string    `db:"status"`     // pending, approved, rejected, executed, failed
	RiskScore          float64   `db:"risk_score"`
	PolicyCheckResult  string    `db:"policy_check_result"` // JSON
	ExecutedAt         time.Time `db:"executed_at"`
	ErrorMessage       string    `db:"error_message"`
	CreatedAt          time.Time `db:"created_at"`
}

// NewsSignal представляет новостной сигнал
type NewsSignal struct {
	ID             int64     `db:"id"`
	Timestamp      time.Time `db:"timestamp"`
	Source         string    `db:"source"`
	Headline       string    `db:"headline"`
	URL            string    `db:"url"`
	Sentiment      string    `db:"sentiment"` // positive, negative, neutral
	SentimentScore float64   `db:"sentiment_score"`
	Topics         []string  `db:"topics"` // crypto, macro, regulation, etc.
	Signal         string    `db:"signal"` // BUY, SELL, HOLD
	Symbols        []string  `db:"symbols"`
	Processed      bool      `db:"processed"`
}

// CircuitBreakerEvent представляет событие триггера circuit breaker
type CircuitBreakerEvent struct {
	ID          int64     `db:"id"`
	TriggeredAt time.Time `db:"triggered_at"`
	Reason      string    `db:"reason"` // drawdown, daily_loss, volatility, news_negative
	Details     string    `db:"details"` // JSON
	PausedUntil time.Time `db:"paused_until"`
	ResumedAt   time.Time `db:"resumed_at"`
}

// PolicyViolation представляет нарушение политики рисков
type PolicyViolation struct {
	ID             int64     `db:"id"`
	Timestamp      time.Time `db:"timestamp"`
	ActionID       int64     `db:"action_id"`
	ViolationType  string    `db:"violation_type"`
	LimitName      string    `db:"limit_name"`
	LimitValue     float64   `db:"limit_value"`
	AttemptedValue float64   `db:"attempted_value"`
	Severity       string    `db:"severity"` // warning, critical
}
