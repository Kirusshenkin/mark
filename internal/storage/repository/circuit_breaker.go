package repository

import (
	"database/sql"
	"time"

	"github.com/kirillm/dca-bot/internal/domain"
)

// CircuitBreakerRepository управляет событиями circuit breaker
type CircuitBreakerRepository struct {
	db *sql.DB
}

// NewCircuitBreakerRepository создает новый репозиторий
func NewCircuitBreakerRepository(db *sql.DB) *CircuitBreakerRepository {
	return &CircuitBreakerRepository{db: db}
}

// SaveEvent сохраняет событие триггера
func (r *CircuitBreakerRepository) SaveEvent(event *domain.CircuitBreakerEvent) error {
	if event.TriggeredAt.IsZero() {
		event.TriggeredAt = time.Now()
	}

	query := `
		INSERT INTO circuit_breaker_events (triggered_at, reason, details, paused_until, resumed_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	return r.db.QueryRow(
		query,
		event.TriggeredAt,
		event.Reason,
		event.Details,
		event.PausedUntil,
		event.ResumedAt,
	).Scan(&event.ID)
}

// GetActive получает активные события (не resumed)
func (r *CircuitBreakerRepository) GetActive() ([]domain.CircuitBreakerEvent, error) {
	query := `
		SELECT id, triggered_at, reason, details, paused_until, resumed_at
		FROM circuit_breaker_events
		WHERE resumed_at IS NULL AND paused_until > NOW()
		ORDER BY triggered_at DESC
	`
	return r.query(query)
}

// GetRecent получает последние N событий
func (r *CircuitBreakerRepository) GetRecent(limit int) ([]domain.CircuitBreakerEvent, error) {
	query := `
		SELECT id, triggered_at, reason, details, paused_until, resumed_at
		FROM circuit_breaker_events
		ORDER BY triggered_at DESC
		LIMIT $1
	`
	return r.query(query, limit)
}

// Resume возобновляет circuit breaker
func (r *CircuitBreakerRepository) Resume(id int64) error {
	query := `
		UPDATE circuit_breaker_events
		SET resumed_at = $1
		WHERE id = $2
	`
	_, err := r.db.Exec(query, time.Now(), id)
	return err
}

// IsAnyActive проверяет есть ли активные circuit breakers
func (r *CircuitBreakerRepository) IsAnyActive() (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM circuit_breaker_events
			WHERE resumed_at IS NULL AND paused_until > NOW()
		)
	`
	var exists bool
	err := r.db.QueryRow(query).Scan(&exists)
	return exists, err
}

// GetStatsByReason получает статистику по причинам
func (r *CircuitBreakerRepository) GetStatsByReason(since time.Time) (map[string]int, error) {
	query := `
		SELECT reason, COUNT(*) as count
		FROM circuit_breaker_events
		WHERE triggered_at >= $1
		GROUP BY reason
	`
	rows, err := r.db.Query(query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var reason string
		var count int
		if err := rows.Scan(&reason, &count); err != nil {
			return nil, err
		}
		stats[reason] = count
	}

	return stats, rows.Err()
}

// query helper
func (r *CircuitBreakerRepository) query(query string, args ...interface{}) ([]domain.CircuitBreakerEvent, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.CircuitBreakerEvent
	for rows.Next() {
		var e domain.CircuitBreakerEvent
		var resumedAt sql.NullTime
		err := rows.Scan(
			&e.ID,
			&e.TriggeredAt,
			&e.Reason,
			&e.Details,
			&e.PausedUntil,
			&resumedAt,
		)
		if err != nil {
			return nil, err
		}
		if resumedAt.Valid {
			e.ResumedAt = resumedAt.Time
		}
		events = append(events, e)
	}

	return events, rows.Err()
}
