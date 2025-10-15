package repository

import (
	"database/sql"
	"time"

	"github.com/kirillm/dca-bot/internal/domain"
)

// RiskRepository реализует работу с лимитами риска
type RiskRepository struct {
	db *sql.DB
}

// NewRiskRepository создает новый репозиторий для лимитов риска
func NewRiskRepository(db *sql.DB) *RiskRepository {
	return &RiskRepository{db: db}
}

// GetLimits получает текущие лимиты риска
func (r *RiskRepository) GetLimits() (*domain.RiskLimit, error) {
	limits := &domain.RiskLimit{}
	query := `
		SELECT id, max_daily_loss, max_total_exposure, max_position_size_usd, max_order_size_usd, enable_emergency_stop, updated_at
		FROM risk_limits
		ORDER BY id DESC
		LIMIT 1
	`
	err := r.db.QueryRow(query).Scan(
		&limits.ID,
		&limits.MaxDailyLoss,
		&limits.MaxTotalExposure,
		&limits.MaxPositionSizeUSD,
		&limits.MaxOrderSizeUSD,
		&limits.EnableEmergencyStop,
		&limits.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Возвращаем дефолтные лимиты
		return &domain.RiskLimit{
			MaxDailyLoss:        1000,
			MaxTotalExposure:    10000,
			MaxPositionSizeUSD:  5000,
			MaxOrderSizeUSD:     500,
			EnableEmergencyStop: false,
			UpdatedAt:           time.Now(),
		}, nil
	}

	return limits, err
}

// UpdateLimits обновляет лимиты риска
func (r *RiskRepository) UpdateLimits(limits *domain.RiskLimit) error {
	limits.UpdatedAt = time.Now()
	query := `
		UPDATE risk_limits
		SET max_daily_loss = $1, max_total_exposure = $2, max_position_size_usd = $3,
		    max_order_size_usd = $4, enable_emergency_stop = $5, updated_at = $6
		WHERE id = (SELECT id FROM risk_limits ORDER BY id DESC LIMIT 1)
	`
	_, err := r.db.Exec(
		query,
		limits.MaxDailyLoss,
		limits.MaxTotalExposure,
		limits.MaxPositionSizeUSD,
		limits.MaxOrderSizeUSD,
		limits.EnableEmergencyStop,
		limits.UpdatedAt,
	)
	return err
}
