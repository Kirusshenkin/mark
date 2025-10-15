package repository

import (
	"database/sql"
	"time"

	"github.com/kirillm/dca-bot/internal/domain"
)

// AssetRepository реализует работу с активами
type AssetRepository struct {
	db *sql.DB
}

// NewAssetRepository создает новый репозиторий для активов
func NewAssetRepository(db *sql.DB) *AssetRepository {
	return &AssetRepository{db: db}
}

// CreateOrUpdate создает или обновляет актив
func (r *AssetRepository) CreateOrUpdate(asset *domain.Asset) error {
	asset.UpdatedAt = time.Now()
	if asset.CreatedAt.IsZero() {
		asset.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO assets (
			symbol, enabled, strategy_type, allocated_capital, max_position_size,
			dca_amount, dca_interval_minutes, auto_sell_enabled,
			auto_sell_trigger_percent, auto_sell_amount_percent,
			grid_levels, grid_spacing_percent, grid_order_size,
			stop_loss_percent, take_profit_percent, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (symbol) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			strategy_type = EXCLUDED.strategy_type,
			allocated_capital = EXCLUDED.allocated_capital,
			max_position_size = EXCLUDED.max_position_size,
			dca_amount = EXCLUDED.dca_amount,
			dca_interval_minutes = EXCLUDED.dca_interval_minutes,
			auto_sell_enabled = EXCLUDED.auto_sell_enabled,
			auto_sell_trigger_percent = EXCLUDED.auto_sell_trigger_percent,
			auto_sell_amount_percent = EXCLUDED.auto_sell_amount_percent,
			grid_levels = EXCLUDED.grid_levels,
			grid_spacing_percent = EXCLUDED.grid_spacing_percent,
			grid_order_size = EXCLUDED.grid_order_size,
			stop_loss_percent = EXCLUDED.stop_loss_percent,
			take_profit_percent = EXCLUDED.take_profit_percent,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`
	return r.db.QueryRow(
		query,
		asset.Symbol,
		asset.Enabled,
		asset.StrategyType,
		asset.AllocatedCapital,
		asset.MaxPositionSize,
		asset.DCAAmount,
		asset.DCAInterval,
		asset.AutoSellEnabled,
		asset.AutoSellTriggerPercent,
		asset.AutoSellAmountPercent,
		asset.GridLevels,
		asset.GridSpacingPercent,
		asset.GridOrderSize,
		asset.StopLossPercent,
		asset.TakeProfitPercent,
		asset.CreatedAt,
		asset.UpdatedAt,
	).Scan(&asset.ID)
}

// Get получает актив по символу
func (r *AssetRepository) Get(symbol string) (*domain.Asset, error) {
	asset := &domain.Asset{}
	query := `
		SELECT id, symbol, enabled, strategy_type, allocated_capital, max_position_size,
		       dca_amount, dca_interval_minutes, auto_sell_enabled,
		       auto_sell_trigger_percent, auto_sell_amount_percent,
		       grid_levels, grid_spacing_percent, grid_order_size,
		       stop_loss_percent, take_profit_percent, created_at, updated_at
		FROM assets WHERE symbol = $1
	`
	err := r.db.QueryRow(query, symbol).Scan(
		&asset.ID,
		&asset.Symbol,
		&asset.Enabled,
		&asset.StrategyType,
		&asset.AllocatedCapital,
		&asset.MaxPositionSize,
		&asset.DCAAmount,
		&asset.DCAInterval,
		&asset.AutoSellEnabled,
		&asset.AutoSellTriggerPercent,
		&asset.AutoSellAmountPercent,
		&asset.GridLevels,
		&asset.GridSpacingPercent,
		&asset.GridOrderSize,
		&asset.StopLossPercent,
		&asset.TakeProfitPercent,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return asset, err
}

// GetEnabled получает все активные активы
func (r *AssetRepository) GetEnabled() ([]domain.Asset, error) {
	query := `
		SELECT id, symbol, enabled, strategy_type, allocated_capital, max_position_size,
		       dca_amount, dca_interval_minutes, auto_sell_enabled,
		       auto_sell_trigger_percent, auto_sell_amount_percent,
		       grid_levels, grid_spacing_percent, grid_order_size,
		       stop_loss_percent, take_profit_percent, created_at, updated_at
		FROM assets WHERE enabled = true
		ORDER BY symbol
	`
	return r.queryAssets(query)
}

// GetAll получает все активы
func (r *AssetRepository) GetAll() ([]domain.Asset, error) {
	query := `
		SELECT id, symbol, enabled, strategy_type, allocated_capital, max_position_size,
		       dca_amount, dca_interval_minutes, auto_sell_enabled,
		       auto_sell_trigger_percent, auto_sell_amount_percent,
		       grid_levels, grid_spacing_percent, grid_order_size,
		       stop_loss_percent, take_profit_percent, created_at, updated_at
		FROM assets
		ORDER BY symbol
	`
	return r.queryAssets(query)
}

// Disable отключает актив
func (r *AssetRepository) Disable(symbol string) error {
	query := `UPDATE assets SET enabled = false, updated_at = $1 WHERE symbol = $2`
	_, err := r.db.Exec(query, time.Now(), symbol)
	return err
}

// Enable включает актив
func (r *AssetRepository) Enable(symbol string) error {
	query := `UPDATE assets SET enabled = true, updated_at = $1 WHERE symbol = $2`
	_, err := r.db.Exec(query, time.Now(), symbol)
	return err
}

// queryAssets выполняет запрос и возвращает список активов
func (r *AssetRepository) queryAssets(query string, args ...interface{}) ([]domain.Asset, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []domain.Asset
	for rows.Next() {
		var asset domain.Asset
		err := rows.Scan(
			&asset.ID,
			&asset.Symbol,
			&asset.Enabled,
			&asset.StrategyType,
			&asset.AllocatedCapital,
			&asset.MaxPositionSize,
			&asset.DCAAmount,
			&asset.DCAInterval,
			&asset.AutoSellEnabled,
			&asset.AutoSellTriggerPercent,
			&asset.AutoSellAmountPercent,
			&asset.GridLevels,
			&asset.GridSpacingPercent,
			&asset.GridOrderSize,
			&asset.StopLossPercent,
			&asset.TakeProfitPercent,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}

	return assets, rows.Err()
}
