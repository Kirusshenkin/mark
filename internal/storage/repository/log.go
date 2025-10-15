package repository

import (
	"database/sql"
	"time"
)

// LogRepository реализует работу с логами
type LogRepository struct {
	db *sql.DB
}

// NewLogRepository создает новый репозиторий для логов
func NewLogRepository(db *sql.DB) *LogRepository {
	return &LogRepository{db: db}
}

// Save сохраняет лог
func (r *LogRepository) Save(level, message, data string) error {
	query := `INSERT INTO logs (level, message, data, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(query, level, message, data, time.Now())
	return err
}
