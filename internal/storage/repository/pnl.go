package repository

import (
	"database/sql"
	"time"

	"github.com/kirillm/dca-bot/internal/domain"
)

// PnLRepository реализует работу с историей PnL
type PnLRepository struct {
	db *sql.DB
}

// NewPnLRepository создает новый репозиторий для PnL
func NewPnLRepository(db *sql.DB) *PnLRepository {
	return &PnLRepository{db: db}
}

// SaveSnapshot сохраняет снимок PnL
func (r *PnLRepository) SaveSnapshot(pnl *domain.PnLHistory) error {
	if pnl.CreatedAt.IsZero() {
		pnl.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO pnl_history (symbol, realized_pnl, unrealized_pnl, total_pnl, total_invested, current_value, return_percent, snapshot_type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	return r.db.QueryRow(
		query,
		pnl.Symbol,
		pnl.RealizedPnL,
		pnl.UnrealizedPnL,
		pnl.TotalPnL,
		pnl.TotalInvested,
		pnl.CurrentValue,
		pnl.ReturnPercent,
		pnl.SnapshotType,
		pnl.CreatedAt,
	).Scan(&pnl.ID)
}

// GetHistory получает историю PnL для символа
func (r *PnLRepository) GetHistory(symbol string, snapshotType string, limit int) ([]domain.PnLHistory, error) {
	query := `
		SELECT id, symbol, realized_pnl, unrealized_pnl, total_pnl, total_invested, current_value, return_percent, snapshot_type, created_at
		FROM pnl_history
		WHERE symbol = $1 AND snapshot_type = $2
		ORDER BY created_at DESC
		LIMIT $3
	`
	rows, err := r.db.Query(query, symbol, snapshotType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []domain.PnLHistory
	for rows.Next() {
		var pnl domain.PnLHistory
		err := rows.Scan(
			&pnl.ID,
			&pnl.Symbol,
			&pnl.RealizedPnL,
			&pnl.UnrealizedPnL,
			&pnl.TotalPnL,
			&pnl.TotalInvested,
			&pnl.CurrentValue,
			&pnl.ReturnPercent,
			&pnl.SnapshotType,
			&pnl.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, pnl)
	}

	return history, rows.Err()
}
