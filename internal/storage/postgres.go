package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/kirillm/dca-bot/internal/domain"
	"github.com/kirillm/dca-bot/internal/storage/repository"
	_ "github.com/lib/pq"
)

// Переопределяем типы из domain для обратной совместимости
type (
	Trade       = domain.Trade
	Balance     = domain.Balance
	Asset       = domain.Asset
	GridOrder   = domain.GridOrder
	PnLHistory  = domain.PnLHistory
	RiskLimit   = domain.RiskLimit
	ConfigParam = domain.ConfigParam
	Log         = domain.Log
)

// PostgresStorage является фасадом для работы с PostgreSQL через репозитории
type PostgresStorage struct {
	db           *sql.DB
	trades       *repository.TradeRepository
	balances     *repository.BalanceRepository
	assets       *repository.AssetRepository
	gridOrders   *repository.GridOrderRepository
	pnl          *repository.PnLRepository
	risk         *repository.RiskRepository
	config       *repository.ConfigRepository
	logs         *repository.LogRepository
}

func NewPostgresStorage(host string, port int, user, password, dbname, sslmode string, maxOpenConns, maxIdleConns int, connMaxLifetime time.Duration) (*PostgresStorage, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Настройка connection pool из конфигурации
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	storage := &PostgresStorage{
		db:         db,
		trades:     repository.NewTradeRepository(db),
		balances:   repository.NewBalanceRepository(db),
		assets:     repository.NewAssetRepository(db),
		gridOrders: repository.NewGridOrderRepository(db),
		pnl:        repository.NewPnLRepository(db),
		risk:       repository.NewRiskRepository(db),
		config:     repository.NewConfigRepository(db),
		logs:       repository.NewLogRepository(db),
	}

	// Запускаем миграции
	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return storage, nil
}

func (s *PostgresStorage) migrate() error {
	migrations := []string{
		// Основная таблица сделок с расширениями для Grid
		`CREATE TABLE IF NOT EXISTS trades (
			id SERIAL PRIMARY KEY,
			symbol VARCHAR(20) NOT NULL,
			side VARCHAR(10) NOT NULL,
			quantity DECIMAL(20, 8) NOT NULL,
			price DECIMAL(20, 8) NOT NULL,
			amount DECIMAL(20, 8) NOT NULL,
			order_id VARCHAR(100),
			status VARCHAR(20) NOT NULL,
			strategy_type VARCHAR(20) DEFAULT 'DCA',
			grid_level INTEGER DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		// Балансы с unrealized PnL
		`CREATE TABLE IF NOT EXISTS balances (
			id SERIAL PRIMARY KEY,
			symbol VARCHAR(20) NOT NULL UNIQUE,
			total_quantity DECIMAL(20, 8) NOT NULL DEFAULT 0,
			available_qty DECIMAL(20, 8) NOT NULL DEFAULT 0,
			avg_entry_price DECIMAL(20, 8) NOT NULL DEFAULT 0,
			total_invested DECIMAL(20, 8) NOT NULL DEFAULT 0,
			total_sold DECIMAL(20, 8) NOT NULL DEFAULT 0,
			realized_profit DECIMAL(20, 8) NOT NULL DEFAULT 0,
			unrealized_pnl DECIMAL(20, 8) NOT NULL DEFAULT 0,
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		// Таблица активов (НОВАЯ)
		`CREATE TABLE IF NOT EXISTS assets (
			id SERIAL PRIMARY KEY,
			symbol VARCHAR(20) NOT NULL UNIQUE,
			enabled BOOLEAN NOT NULL DEFAULT true,
			strategy_type VARCHAR(20) NOT NULL DEFAULT 'DCA',
			allocated_capital DECIMAL(20, 8) NOT NULL DEFAULT 0,
			max_position_size DECIMAL(20, 8) NOT NULL DEFAULT 0,
			dca_amount DECIMAL(20, 8) NOT NULL DEFAULT 10,
			dca_interval_minutes INTEGER NOT NULL DEFAULT 1440,
			auto_sell_enabled BOOLEAN NOT NULL DEFAULT false,
			auto_sell_trigger_percent DECIMAL(10, 2) NOT NULL DEFAULT 10,
			auto_sell_amount_percent DECIMAL(10, 2) NOT NULL DEFAULT 50,
			grid_levels INTEGER DEFAULT 0,
			grid_spacing_percent DECIMAL(10, 2) DEFAULT 0,
			grid_order_size DECIMAL(20, 8) DEFAULT 0,
			stop_loss_percent DECIMAL(10, 2) DEFAULT 0,
			take_profit_percent DECIMAL(10, 2) DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		// Grid ордера (НОВАЯ)
		`CREATE TABLE IF NOT EXISTS grid_orders (
			id SERIAL PRIMARY KEY,
			symbol VARCHAR(20) NOT NULL,
			level INTEGER NOT NULL,
			side VARCHAR(10) NOT NULL,
			price DECIMAL(20, 8) NOT NULL,
			quantity DECIMAL(20, 8) NOT NULL,
			order_id VARCHAR(100),
			status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
			filled_qty DECIMAL(20, 8) NOT NULL DEFAULT 0,
			filled_price DECIMAL(20, 8) NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		// История PnL (НОВАЯ)
		`CREATE TABLE IF NOT EXISTS pnl_history (
			id SERIAL PRIMARY KEY,
			symbol VARCHAR(20) NOT NULL,
			realized_pnl DECIMAL(20, 8) NOT NULL,
			unrealized_pnl DECIMAL(20, 8) NOT NULL,
			total_pnl DECIMAL(20, 8) NOT NULL,
			total_invested DECIMAL(20, 8) NOT NULL,
			current_value DECIMAL(20, 8) NOT NULL,
			return_percent DECIMAL(10, 2) NOT NULL,
			snapshot_type VARCHAR(20) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		// Лимиты риска (НОВАЯ)
		`CREATE TABLE IF NOT EXISTS risk_limits (
			id SERIAL PRIMARY KEY,
			max_daily_loss DECIMAL(20, 8) NOT NULL DEFAULT 0,
			max_total_exposure DECIMAL(20, 8) NOT NULL DEFAULT 0,
			max_position_size_usd DECIMAL(20, 8) NOT NULL DEFAULT 0,
			max_order_size_usd DECIMAL(20, 8) NOT NULL DEFAULT 0,
			enable_emergency_stop BOOLEAN NOT NULL DEFAULT false,
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS config_params (
			id SERIAL PRIMARY KEY,
			key VARCHAR(100) NOT NULL UNIQUE,
			value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS logs (
			id SERIAL PRIMARY KEY,
			level VARCHAR(10) NOT NULL,
			message TEXT NOT NULL,
			data TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		// Индексы
		`CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol)`,
		`CREATE INDEX IF NOT EXISTS idx_trades_created_at ON trades(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_trades_strategy ON trades(strategy_type)`,
		`CREATE INDEX IF NOT EXISTS idx_grid_orders_symbol ON grid_orders(symbol)`,
		`CREATE INDEX IF NOT EXISTS idx_grid_orders_status ON grid_orders(status)`,
		`CREATE INDEX IF NOT EXISTS idx_pnl_history_symbol ON pnl_history(symbol)`,
		`CREATE INDEX IF NOT EXISTS idx_pnl_history_created_at ON pnl_history(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_created_at ON logs(created_at)`,
		// Миграции для существующих таблиц
		`ALTER TABLE trades ADD COLUMN IF NOT EXISTS strategy_type VARCHAR(20) DEFAULT 'DCA'`,
		`ALTER TABLE trades ADD COLUMN IF NOT EXISTS grid_level INTEGER DEFAULT 0`,
		`ALTER TABLE balances ADD COLUMN IF NOT EXISTS unrealized_pnl DECIMAL(20, 8) DEFAULT 0`,
		// Создаем risk_limits row по умолчанию, если нет
		`INSERT INTO risk_limits (max_daily_loss, max_total_exposure, max_position_size_usd, max_order_size_usd, enable_emergency_stop, updated_at)
		 SELECT 1000, 10000, 5000, 500, false, NOW()
		 WHERE NOT EXISTS (SELECT 1 FROM risk_limits LIMIT 1)`,
		// STAGE 4: AI Decisions
		`CREATE TABLE IF NOT EXISTS ai_decisions (
			id BIGSERIAL PRIMARY KEY,
			timestamp TIMESTAMPTZ DEFAULT NOW(),
			regime VARCHAR(50) NOT NULL,
			confidence NUMERIC(3,2),
			rationale TEXT,
			raw_response JSONB,
			approved BOOLEAN DEFAULT false,
			rejection_reason TEXT,
			mode VARCHAR(20) NOT NULL
		)`,
		// STAGE 4: AI Actions
		`CREATE TABLE IF NOT EXISTS ai_actions (
			id BIGSERIAL PRIMARY KEY,
			decision_id BIGINT REFERENCES ai_decisions(id),
			action_type VARCHAR(50) NOT NULL,
			symbol VARCHAR(20),
			parameters JSONB NOT NULL,
			status VARCHAR(20) DEFAULT 'pending',
			risk_score NUMERIC(3,2),
			policy_check_result JSONB,
			executed_at TIMESTAMPTZ,
			error_message TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		// STAGE 4: News Signals
		`CREATE TABLE IF NOT EXISTS news_signals (
			id BIGSERIAL PRIMARY KEY,
			timestamp TIMESTAMPTZ DEFAULT NOW(),
			source VARCHAR(100),
			headline TEXT,
			url TEXT,
			sentiment VARCHAR(20),
			sentiment_score NUMERIC(3,2),
			topics TEXT[],
			signal VARCHAR(10),
			symbols TEXT[],
			processed BOOLEAN DEFAULT false
		)`,
		// STAGE 4: Circuit Breaker Events
		`CREATE TABLE IF NOT EXISTS circuit_breaker_events (
			id BIGSERIAL PRIMARY KEY,
			triggered_at TIMESTAMPTZ DEFAULT NOW(),
			reason VARCHAR(100),
			details JSONB,
			paused_until TIMESTAMPTZ,
			resumed_at TIMESTAMPTZ
		)`,
		// STAGE 4: Policy Violations
		`CREATE TABLE IF NOT EXISTS policy_violations (
			id BIGSERIAL PRIMARY KEY,
			timestamp TIMESTAMPTZ DEFAULT NOW(),
			action_id BIGINT REFERENCES ai_actions(id),
			violation_type VARCHAR(50),
			limit_name VARCHAR(50),
			limit_value NUMERIC,
			attempted_value NUMERIC,
			severity VARCHAR(20)
		)`,
		// STAGE 4: Extend pnl_history
		`ALTER TABLE pnl_history ADD COLUMN IF NOT EXISTS regime VARCHAR(50)`,
		`ALTER TABLE pnl_history ADD COLUMN IF NOT EXISTS auto_mode BOOLEAN DEFAULT false`,
		`ALTER TABLE pnl_history ADD COLUMN IF NOT EXISTS risk_score NUMERIC(3,2)`,
		// STAGE 4: Indexes for new tables
		`CREATE INDEX IF NOT EXISTS idx_ai_decisions_timestamp ON ai_decisions(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_decisions_mode ON ai_decisions(mode)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_actions_decision_id ON ai_actions(decision_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_actions_status ON ai_actions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_news_signals_timestamp ON news_signals(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_news_signals_processed ON news_signals(processed)`,
		`CREATE INDEX IF NOT EXISTS idx_circuit_breaker_triggered_at ON circuit_breaker_events(triggered_at)`,
		`CREATE INDEX IF NOT EXISTS idx_policy_violations_timestamp ON policy_violations(timestamp)`,
	}

	for _, migration := range migrations {
		if _, err := s.db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// ==================== TRADES ====================

func (s *PostgresStorage) SaveTrade(trade *Trade) error {
	return s.trades.Save(trade)
}

func (s *PostgresStorage) GetRecentTrades(symbol string, limit int) ([]Trade, error) {
	return s.trades.GetRecent(symbol, limit)
}

func (s *PostgresStorage) GetAllRecentTrades(limit int) ([]Trade, error) {
	return s.trades.GetAllRecent(limit)
}

// ==================== BALANCES ====================

func (s *PostgresStorage) GetBalance(symbol string) (*Balance, error) {
	return s.balances.Get(symbol)
}

func (s *PostgresStorage) GetAllBalances() ([]Balance, error) {
	return s.balances.GetAll()
}

func (s *PostgresStorage) UpdateBalance(balance *Balance) error {
	return s.balances.Update(balance)
}

// ==================== ASSETS ====================

func (s *PostgresStorage) CreateOrUpdateAsset(asset *Asset) error {
	return s.assets.CreateOrUpdate(asset)
}

func (s *PostgresStorage) GetAsset(symbol string) (*Asset, error) {
	return s.assets.Get(symbol)
}

func (s *PostgresStorage) GetEnabledAssets() ([]Asset, error) {
	return s.assets.GetEnabled()
}

func (s *PostgresStorage) GetAllAssets() ([]Asset, error) {
	return s.assets.GetAll()
}

func (s *PostgresStorage) DisableAsset(symbol string) error {
	return s.assets.Disable(symbol)
}

func (s *PostgresStorage) EnableAsset(symbol string) error {
	return s.assets.Enable(symbol)
}

// ==================== GRID ORDERS ====================

func (s *PostgresStorage) SaveGridOrder(order *GridOrder) error {
	return s.gridOrders.Save(order)
}

func (s *PostgresStorage) UpdateGridOrder(order *GridOrder) error {
	return s.gridOrders.Update(order)
}

func (s *PostgresStorage) GetActiveGridOrders(symbol string) ([]GridOrder, error) {
	return s.gridOrders.GetActive(symbol)
}

func (s *PostgresStorage) CancelGridOrders(symbol string) error {
	return s.gridOrders.CancelAll(symbol)
}

// ==================== PNL HISTORY ====================

func (s *PostgresStorage) SavePnLSnapshot(pnl *PnLHistory) error {
	return s.pnl.SaveSnapshot(pnl)
}

func (s *PostgresStorage) GetPnLHistory(symbol string, snapshotType string, limit int) ([]PnLHistory, error) {
	return s.pnl.GetHistory(symbol, snapshotType, limit)
}

// ==================== RISK LIMITS ====================

func (s *PostgresStorage) GetRiskLimits() (*RiskLimit, error) {
	return s.risk.GetLimits()
}

func (s *PostgresStorage) UpdateRiskLimits(limits *RiskLimit) error {
	return s.risk.UpdateLimits(limits)
}

// ==================== CONFIG PARAMS ====================

func (s *PostgresStorage) SetConfigParam(key, value string) error {
	return s.config.Set(key, value)
}

func (s *PostgresStorage) GetConfigParam(key string) (string, error) {
	return s.config.Get(key)
}

// ==================== LOGS ====================

func (s *PostgresStorage) SaveLog(level, message, data string) error {
	return s.logs.Save(level, message, data)
}

// Close закрывает соединение с базой данных
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// DB возвращает указатель на *sql.DB для использования в репозиториях Stage 4
func (s *PostgresStorage) DB() *sql.DB {
	return s.db
}
