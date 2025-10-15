package policy

import "time"

// Policy представляет профиль риск-менеджмента
type Policy struct {
	ProfileName         string            `yaml:"profile_name"`
	MaxOrderUSDT        float64           `yaml:"max_order_usdt"`
	MaxPositionUSDT     float64           `yaml:"max_position_usdt"`
	MaxTotalExposure    float64           `yaml:"max_total_exposure"`
	MaxDailyLossUSDT    float64           `yaml:"max_daily_loss_usdt"`
	TradesPerHour       int               `yaml:"trades_per_hour"`
	SlippageThreshold   float64           `yaml:"slippage_threshold"`
	CircuitBreakers     []CircuitBreaker  `yaml:"circuit_breakers"`
}

// CircuitBreaker описывает автоматический предохранитель
type CircuitBreaker struct {
	Type      string  `yaml:"type"`      // drawdown, daily_loss, volatility, news_negative
	Threshold float64 `yaml:"threshold"` // Пороговое значение
	Action    string  `yaml:"action"`    // pause, conservative, killswitch
}

// ActionRequest представляет запрос на выполнение действия
type ActionRequest struct {
	Type       string                 `json:"type"`       // set_dca, set_grid, set_autosell, rebalance, pause_strategy
	Symbol     string                 `json:"symbol"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ValidationResult результат проверки действия политикой
type ValidationResult struct {
	Approved        bool
	RiskScore       float64
	Violations      []Violation
	FallbackProfile *Policy
	CheckedAt       time.Time
}

// Violation описывает нарушение политики
type Violation struct {
	Type           string  // order_size, position_size, daily_loss, trade_frequency
	LimitName      string
	LimitValue     float64
	AttemptedValue float64
	Severity       string  // warning, critical
	Message        string
}

// RiskMetrics текущие метрики риска
type RiskMetrics struct {
	TotalExposureUSDT float64
	DailyLossUSDT     float64
	DailyTradeCount   int
	CurrentDrawdown   float64
	VolatilityPct     float64
	LastUpdated       time.Time
}
