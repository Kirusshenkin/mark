package repository

import (
	"database/sql"
	"time"

	"github.com/kirillm/dca-bot/internal/domain"
)

// AIActionRepository управляет AI действиями
type AIActionRepository struct {
	db *sql.DB
}

// NewAIActionRepository создает новый репозиторий
func NewAIActionRepository(db *sql.DB) *AIActionRepository {
	return &AIActionRepository{db: db}
}

// Save сохраняет AI действие
func (r *AIActionRepository) Save(action *domain.AIAction) error {
	if action.CreatedAt.IsZero() {
		action.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO ai_actions (
			decision_id, action_type, symbol, parameters, status,
			risk_score, policy_check_result, executed_at, error_message, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	return r.db.QueryRow(
		query,
		action.DecisionID,
		action.ActionType,
		action.Symbol,
		action.Parameters,
		action.Status,
		action.RiskScore,
		action.PolicyCheckResult,
		action.ExecutedAt,
		action.ErrorMessage,
		action.CreatedAt,
	).Scan(&action.ID)
}

// UpdateStatus обновляет статус действия
func (r *AIActionRepository) UpdateStatus(id int64, status string, errorMsg string) error {
	query := `
		UPDATE ai_actions
		SET status = $1, error_message = $2, executed_at = $3
		WHERE id = $4
	`
	_, err := r.db.Exec(query, status, errorMsg, time.Now(), id)
	return err
}

// GetByDecisionID получает действия по ID решения
func (r *AIActionRepository) GetByDecisionID(decisionID int64) ([]domain.AIAction, error) {
	query := `
		SELECT id, decision_id, action_type, symbol, parameters, status,
		       risk_score, policy_check_result, executed_at, error_message, created_at
		FROM ai_actions
		WHERE decision_id = $1
		ORDER BY created_at
	`
	rows, err := r.db.Query(query, decisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []domain.AIAction
	for rows.Next() {
		var a domain.AIAction
		err := rows.Scan(
			&a.ID,
			&a.DecisionID,
			&a.ActionType,
			&a.Symbol,
			&a.Parameters,
			&a.Status,
			&a.RiskScore,
			&a.PolicyCheckResult,
			&a.ExecutedAt,
			&a.ErrorMessage,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}

	return actions, rows.Err()
}

// GetByStatus получает действия по статусу
func (r *AIActionRepository) GetByStatus(status string, limit int) ([]domain.AIAction, error) {
	query := `
		SELECT id, decision_id, action_type, symbol, parameters, status,
		       risk_score, policy_check_result, executed_at, error_message, created_at
		FROM ai_actions
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.db.Query(query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []domain.AIAction
	for rows.Next() {
		var a domain.AIAction
		err := rows.Scan(
			&a.ID,
			&a.DecisionID,
			&a.ActionType,
			&a.Symbol,
			&a.Parameters,
			&a.Status,
			&a.RiskScore,
			&a.PolicyCheckResult,
			&a.ExecutedAt,
			&a.ErrorMessage,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}

	return actions, rows.Err()
}

// GetRecent получает последние N действий
func (r *AIActionRepository) GetRecent(limit int) ([]domain.AIAction, error) {
	query := `
		SELECT id, decision_id, action_type, symbol, parameters, status,
		       risk_score, policy_check_result, executed_at, error_message, created_at
		FROM ai_actions
		ORDER BY created_at DESC
		LIMIT $1
	`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []domain.AIAction
	for rows.Next() {
		var a domain.AIAction
		err := rows.Scan(
			&a.ID,
			&a.DecisionID,
			&a.ActionType,
			&a.Symbol,
			&a.Parameters,
			&a.Status,
			&a.RiskScore,
			&a.PolicyCheckResult,
			&a.ExecutedAt,
			&a.ErrorMessage,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}

	return actions, rows.Err()
}

// GetSuccessRate получает процент успешных действий
func (r *AIActionRepository) GetSuccessRate(since time.Time) (float64, error) {
	query := `
		SELECT
			COUNT(CASE WHEN status = 'executed' THEN 1 END)::float / NULLIF(COUNT(*)::float, 0) * 100 as success_rate
		FROM ai_actions
		WHERE created_at >= $1 AND status IN ('executed', 'failed')
	`
	var rate sql.NullFloat64
	err := r.db.QueryRow(query, since).Scan(&rate)
	if err != nil {
		return 0, err
	}
	if !rate.Valid {
		return 0, nil
	}
	return rate.Float64, nil
}
