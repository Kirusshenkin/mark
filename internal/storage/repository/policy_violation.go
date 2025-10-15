package repository

import (
	"database/sql"
	"time"

	"github.com/kirillm/dca-bot/internal/domain"
)

// PolicyViolationRepository управляет нарушениями политики
type PolicyViolationRepository struct {
	db *sql.DB
}

// NewPolicyViolationRepository создает новый репозиторий
func NewPolicyViolationRepository(db *sql.DB) *PolicyViolationRepository {
	return &PolicyViolationRepository{db: db}
}

// Save сохраняет нарушение политики
func (r *PolicyViolationRepository) Save(violation *domain.PolicyViolation) error {
	if violation.Timestamp.IsZero() {
		violation.Timestamp = time.Now()
	}

	query := `
		INSERT INTO policy_violations (
			timestamp, action_id, violation_type, limit_name,
			limit_value, attempted_value, severity
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	return r.db.QueryRow(
		query,
		violation.Timestamp,
		violation.ActionID,
		violation.ViolationType,
		violation.LimitName,
		violation.LimitValue,
		violation.AttemptedValue,
		violation.Severity,
	).Scan(&violation.ID)
}

// GetRecent получает последние N нарушений
func (r *PolicyViolationRepository) GetRecent(limit int) ([]domain.PolicyViolation, error) {
	query := `
		SELECT id, timestamp, action_id, violation_type, limit_name,
		       limit_value, attempted_value, severity
		FROM policy_violations
		ORDER BY timestamp DESC
		LIMIT $1
	`
	return r.query(query, limit)
}

// GetByActionID получает нарушения по ID действия
func (r *PolicyViolationRepository) GetByActionID(actionID int64) ([]domain.PolicyViolation, error) {
	query := `
		SELECT id, timestamp, action_id, violation_type, limit_name,
		       limit_value, attempted_value, severity
		FROM policy_violations
		WHERE action_id = $1
		ORDER BY timestamp DESC
	`
	return r.query(query, actionID)
}

// GetBySeverity получает нарушения по severity
func (r *PolicyViolationRepository) GetBySeverity(severity string, limit int) ([]domain.PolicyViolation, error) {
	query := `
		SELECT id, timestamp, action_id, violation_type, limit_name,
		       limit_value, attempted_value, severity
		FROM policy_violations
		WHERE severity = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	return r.query(query, severity, limit)
}

// GetRecentByHours получает нарушения за последние N часов
func (r *PolicyViolationRepository) GetRecentByHours(hours int) ([]domain.PolicyViolation, error) {
	query := `
		SELECT id, timestamp, action_id, violation_type, limit_name,
		       limit_value, attempted_value, severity
		FROM policy_violations
		WHERE timestamp >= NOW() - INTERVAL '1 hour' * $1
		ORDER BY timestamp DESC
	`
	return r.query(query, hours)
}

// GetStatsByType получает статистику по типам нарушений
func (r *PolicyViolationRepository) GetStatsByType(since time.Time) (map[string]int, error) {
	query := `
		SELECT violation_type, COUNT(*) as count
		FROM policy_violations
		WHERE timestamp >= $1
		GROUP BY violation_type
	`
	rows, err := r.db.Query(query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var violationType string
		var count int
		if err := rows.Scan(&violationType, &count); err != nil {
			return nil, err
		}
		stats[violationType] = count
	}

	return stats, rows.Err()
}

// CountCritical считает критические нарушения за период
func (r *PolicyViolationRepository) CountCritical(since time.Time) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM policy_violations
		WHERE severity = 'critical' AND timestamp >= $1
	`
	var count int
	err := r.db.QueryRow(query, since).Scan(&count)
	return count, err
}

// query helper
func (r *PolicyViolationRepository) query(query string, args ...interface{}) ([]domain.PolicyViolation, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var violations []domain.PolicyViolation
	for rows.Next() {
		var v domain.PolicyViolation
		err := rows.Scan(
			&v.ID,
			&v.Timestamp,
			&v.ActionID,
			&v.ViolationType,
			&v.LimitName,
			&v.LimitValue,
			&v.AttemptedValue,
			&v.Severity,
		)
		if err != nil {
			return nil, err
		}
		violations = append(violations, v)
	}

	return violations, rows.Err()
}
