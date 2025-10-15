package repository

import (
	"database/sql"

	"github.com/kirillm/dca-bot/internal/domain"
)

// TradeRepository реализует работу с торговыми операциями
type TradeRepository struct {
	db *sql.DB
}

// NewTradeRepository создает новый репозиторий для торговых операций
func NewTradeRepository(db *sql.DB) *TradeRepository {
	return &TradeRepository{db: db}
}

// Save сохраняет новую торговую операцию
func (r *TradeRepository) Save(trade *domain.Trade) error {
	query := `
		INSERT INTO trades (symbol, side, quantity, price, amount, order_id, status, strategy_type, grid_level, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	return r.db.QueryRow(
		query,
		trade.Symbol,
		trade.Side,
		trade.Quantity,
		trade.Price,
		trade.Amount,
		trade.OrderID,
		trade.Status,
		trade.StrategyType,
		trade.GridLevel,
		trade.CreatedAt,
	).Scan(&trade.ID)
}

// GetRecent получает последние N торговых операций для символа
func (r *TradeRepository) GetRecent(symbol string, limit int) ([]domain.Trade, error) {
	query := `
		SELECT id, symbol, side, quantity, price, amount, order_id, status,
		       COALESCE(strategy_type, 'DCA'), COALESCE(grid_level, 0), created_at
		FROM trades
		WHERE symbol = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	return r.queryTrades(query, symbol, limit)
}

// GetAllRecent получает последние N торговых операций по всем символам
func (r *TradeRepository) GetAllRecent(limit int) ([]domain.Trade, error) {
	query := `
		SELECT id, symbol, side, quantity, price, amount, order_id, status,
		       COALESCE(strategy_type, 'DCA'), COALESCE(grid_level, 0), created_at
		FROM trades
		ORDER BY created_at DESC
		LIMIT $1
	`
	return r.queryTrades(query, limit)
}

// queryTrades выполняет запрос и возвращает список торговых операций
func (r *TradeRepository) queryTrades(query string, args ...interface{}) ([]domain.Trade, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []domain.Trade
	for rows.Next() {
		var trade domain.Trade
		err := rows.Scan(
			&trade.ID,
			&trade.Symbol,
			&trade.Side,
			&trade.Quantity,
			&trade.Price,
			&trade.Amount,
			&trade.OrderID,
			&trade.Status,
			&trade.StrategyType,
			&trade.GridLevel,
			&trade.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		trades = append(trades, trade)
	}

	return trades, rows.Err()
}
