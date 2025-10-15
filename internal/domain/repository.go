package domain

// TradeRepository определяет интерфейс для работы с торговыми операциями
type TradeRepository interface {
	Save(trade *Trade) error
	GetRecent(symbol string, limit int) ([]Trade, error)
	GetAllRecent(limit int) ([]Trade, error)
}

// BalanceRepository определяет интерфейс для работы с балансами
type BalanceRepository interface {
	Get(symbol string) (*Balance, error)
	GetAll() ([]Balance, error)
	Update(balance *Balance) error
}

// AssetRepository определяет интерфейс для работы с активами
type AssetRepository interface {
	CreateOrUpdate(asset *Asset) error
	Get(symbol string) (*Asset, error)
	GetEnabled() ([]Asset, error)
	GetAll() ([]Asset, error)
	Disable(symbol string) error
	Enable(symbol string) error
}

// GridOrderRepository определяет интерфейс для работы с Grid ордерами
type GridOrderRepository interface {
	Save(order *GridOrder) error
	Update(order *GridOrder) error
	GetActive(symbol string) ([]GridOrder, error)
	CancelAll(symbol string) error
}

// PnLRepository определяет интерфейс для работы с PnL историей
type PnLRepository interface {
	SaveSnapshot(pnl *PnLHistory) error
	GetHistory(symbol string, snapshotType string, limit int) ([]PnLHistory, error)
}

// RiskRepository определяет интерфейс для работы с лимитами риска
type RiskRepository interface {
	GetLimits() (*RiskLimit, error)
	UpdateLimits(limits *RiskLimit) error
}

// ConfigRepository определяет интерфейс для работы с конфигурацией
type ConfigRepository interface {
	Set(key, value string) error
	Get(key string) (string, error)
}

// LogRepository определяет интерфейс для работы с логами
type LogRepository interface {
	Save(level, message, data string) error
}
