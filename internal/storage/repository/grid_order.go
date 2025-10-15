package repository

import (
	"database/sql"
	"time"

	"github.com/kirillm/dca-bot/internal/domain"
)

// GridOrderRepository реализует работу с Grid ордерами
type GridOrderRepository struct {
	db *sql.DB
}

// NewGridOrderRepository создает новый репозиторий для Grid ордеров
func NewGridOrderRepository(db *sql.DB) *GridOrderRepository {
	return &GridOrderRepository{db: db}
}

// Save сохраняет новый Grid ордер
func (r *GridOrderRepository) Save(order *domain.GridOrder) error {
	order.UpdatedAt = time.Now()
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO grid_orders (symbol, level, side, price, quantity, order_id, status, filled_qty, filled_price, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`
	return r.db.QueryRow(
		query,
		order.Symbol,
		order.Level,
		order.Side,
		order.Price,
		order.Quantity,
		order.OrderID,
		order.Status,
		order.FilledQty,
		order.FilledPrice,
		order.CreatedAt,
		order.UpdatedAt,
	).Scan(&order.ID)
}

// Update обновляет существующий Grid ордер
func (r *GridOrderRepository) Update(order *domain.GridOrder) error {
	order.UpdatedAt = time.Now()
	query := `
		UPDATE grid_orders
		SET status = $1, filled_qty = $2, filled_price = $3, order_id = $4, updated_at = $5
		WHERE id = $6
	`
	_, err := r.db.Exec(
		query,
		order.Status,
		order.FilledQty,
		order.FilledPrice,
		order.OrderID,
		order.UpdatedAt,
		order.ID,
	)
	return err
}

// GetActive получает активные Grid ордера для символа
func (r *GridOrderRepository) GetActive(symbol string) ([]domain.GridOrder, error) {
	query := `
		SELECT id, symbol, level, side, price, quantity, order_id, status, filled_qty, filled_price, created_at, updated_at
		FROM grid_orders
		WHERE symbol = $1 AND status IN ('PENDING', 'PLACED')
		ORDER BY level
	`
	rows, err := r.db.Query(query, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.GridOrder
	for rows.Next() {
		var order domain.GridOrder
		err := rows.Scan(
			&order.ID,
			&order.Symbol,
			&order.Level,
			&order.Side,
			&order.Price,
			&order.Quantity,
			&order.OrderID,
			&order.Status,
			&order.FilledQty,
			&order.FilledPrice,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// CancelAll отменяет все Grid ордера для символа
func (r *GridOrderRepository) CancelAll(symbol string) error {
	query := `
		UPDATE grid_orders
		SET status = 'CANCELLED', updated_at = $1
		WHERE symbol = $2 AND status IN ('PENDING', 'PLACED')
	`
	_, err := r.db.Exec(query, time.Now(), symbol)
	return err
}
