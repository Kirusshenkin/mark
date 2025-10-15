package repository

import (
	"database/sql"
	"time"

	"github.com/kirillm/dca-bot/internal/domain"
)

// BalanceRepository реализует работу с балансами
type BalanceRepository struct {
	db *sql.DB
}

// NewBalanceRepository создает новый репозиторий для балансов
func NewBalanceRepository(db *sql.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

// Get получает баланс для символа
func (r *BalanceRepository) Get(symbol string) (*domain.Balance, error) {
	balance := &domain.Balance{}
	query := `
		SELECT id, symbol, total_quantity, available_qty, avg_entry_price,
		       total_invested, total_sold, realized_profit,
		       COALESCE(unrealized_pnl, 0), updated_at
		FROM balances WHERE symbol = $1
	`
	err := r.db.QueryRow(query, symbol).Scan(
		&balance.ID,
		&balance.Symbol,
		&balance.TotalQuantity,
		&balance.AvailableQty,
		&balance.AvgEntryPrice,
		&balance.TotalInvested,
		&balance.TotalSold,
		&balance.RealizedProfit,
		&balance.UnrealizedPnL,
		&balance.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Возвращаем пустой баланс если не найден
		return &domain.Balance{
			Symbol:         symbol,
			TotalQuantity:  0,
			AvailableQty:   0,
			AvgEntryPrice:  0,
			TotalInvested:  0,
			TotalSold:      0,
			RealizedProfit: 0,
			UnrealizedPnL:  0,
			UpdatedAt:      time.Now(),
		}, nil
	}

	if err != nil {
		return nil, err
	}

	return balance, nil
}

// GetAll получает все балансы с позициями
func (r *BalanceRepository) GetAll() ([]domain.Balance, error) {
	query := `
		SELECT id, symbol, total_quantity, available_qty, avg_entry_price,
		       total_invested, total_sold, realized_profit,
		       COALESCE(unrealized_pnl, 0), updated_at
		FROM balances
		WHERE total_quantity > 0 OR total_invested > 0
		ORDER BY symbol
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []domain.Balance
	for rows.Next() {
		var balance domain.Balance
		err := rows.Scan(
			&balance.ID,
			&balance.Symbol,
			&balance.TotalQuantity,
			&balance.AvailableQty,
			&balance.AvgEntryPrice,
			&balance.TotalInvested,
			&balance.TotalSold,
			&balance.RealizedProfit,
			&balance.UnrealizedPnL,
			&balance.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		balances = append(balances, balance)
	}

	return balances, rows.Err()
}

// Update обновляет или создает баланс
func (r *BalanceRepository) Update(balance *domain.Balance) error {
	balance.UpdatedAt = time.Now()
	query := `
		INSERT INTO balances (symbol, total_quantity, available_qty, avg_entry_price,
		                     total_invested, total_sold, realized_profit, unrealized_pnl, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (symbol) DO UPDATE SET
			total_quantity = EXCLUDED.total_quantity,
			available_qty = EXCLUDED.available_qty,
			avg_entry_price = EXCLUDED.avg_entry_price,
			total_invested = EXCLUDED.total_invested,
			total_sold = EXCLUDED.total_sold,
			realized_profit = EXCLUDED.realized_profit,
			unrealized_pnl = EXCLUDED.unrealized_pnl,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.Exec(
		query,
		balance.Symbol,
		balance.TotalQuantity,
		balance.AvailableQty,
		balance.AvgEntryPrice,
		balance.TotalInvested,
		balance.TotalSold,
		balance.RealizedProfit,
		balance.UnrealizedPnL,
		balance.UpdatedAt,
	)
	return err
}
