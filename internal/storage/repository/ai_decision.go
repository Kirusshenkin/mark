package repository

import (
	"database/sql"
	"time"

	"github.com/kirillm/dca-bot/internal/domain"
)

// AIDecisionRepository управляет AI решениями
type AIDecisionRepository struct {
	db *sql.DB
}

// NewAIDecisionRepository создает новый репозиторий
func NewAIDecisionRepository(db *sql.DB) *AIDecisionRepository {
	return &AIDecisionRepository{db: db}
}

// Save сохраняет AI решение
func (r *AIDecisionRepository) Save(decision *domain.AIDecision) error {
	query := `
		INSERT INTO ai_decisions (timestamp, regime, confidence, rationale, raw_response, approved, rejection_reason, mode)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	return r.db.QueryRow(
		query,
		decision.Timestamp,
		decision.Regime,
		decision.Confidence,
		decision.Rationale,
		decision.RawResponse,
		decision.Approved,
		decision.RejectionReason,
		decision.Mode,
	).Scan(&decision.ID)
}

// GetRecent получает последние N решений
func (r *AIDecisionRepository) GetRecent(limit int) ([]domain.AIDecision, error) {
	query := `
		SELECT id, timestamp, regime, confidence, rationale, raw_response, approved, rejection_reason, mode
		FROM ai_decisions
		ORDER BY timestamp DESC
		LIMIT $1
	`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []domain.AIDecision
	for rows.Next() {
		var d domain.AIDecision
		err := rows.Scan(
			&d.ID,
			&d.Timestamp,
			&d.Regime,
			&d.Confidence,
			&d.Rationale,
			&d.RawResponse,
			&d.Approved,
			&d.RejectionReason,
			&d.Mode,
		)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, d)
	}

	return decisions, rows.Err()
}

// GetByID получает решение по ID
func (r *AIDecisionRepository) GetByID(id int64) (*domain.AIDecision, error) {
	query := `
		SELECT id, timestamp, regime, confidence, rationale, raw_response, approved, rejection_reason, mode
		FROM ai_decisions
		WHERE id = $1
	`
	var d domain.AIDecision
	err := r.db.QueryRow(query, id).Scan(
		&d.ID,
		&d.Timestamp,
		&d.Regime,
		&d.Confidence,
		&d.Rationale,
		&d.RawResponse,
		&d.Approved,
		&d.RejectionReason,
		&d.Mode,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &d, err
}

// GetByMode получает решения по режиму
func (r *AIDecisionRepository) GetByMode(mode string, limit int) ([]domain.AIDecision, error) {
	query := `
		SELECT id, timestamp, regime, confidence, rationale, raw_response, approved, rejection_reason, mode
		FROM ai_decisions
		WHERE mode = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	rows, err := r.db.Query(query, mode, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []domain.AIDecision
	for rows.Next() {
		var d domain.AIDecision
		err := rows.Scan(
			&d.ID,
			&d.Timestamp,
			&d.Regime,
			&d.Confidence,
			&d.Rationale,
			&d.RawResponse,
			&d.Approved,
			&d.RejectionReason,
			&d.Mode,
		)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, d)
	}

	return decisions, rows.Err()
}

// GetStats получает статистику решений
func (r *AIDecisionRepository) GetStats(since time.Time) (map[string]int, error) {
	query := `
		SELECT regime, COUNT(*) as count
		FROM ai_decisions
		WHERE timestamp >= $1
		GROUP BY regime
	`
	rows, err := r.db.Query(query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var regime string
		var count int
		if err := rows.Scan(&regime, &count); err != nil {
			return nil, err
		}
		stats[regime] = count
	}

	return stats, rows.Err()
}
