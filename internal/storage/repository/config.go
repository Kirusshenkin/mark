package repository

import (
	"database/sql"
	"time"
)

// ConfigRepository реализует работу с параметрами конфигурации
type ConfigRepository struct {
	db *sql.DB
}

// NewConfigRepository создает новый репозиторий для конфигурации
func NewConfigRepository(db *sql.DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

// Set устанавливает параметр конфигурации
func (r *ConfigRepository) Set(key, value string) error {
	query := `
		INSERT INTO config_params (key, value, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.Exec(query, key, value, time.Now())
	return err
}

// Get получает параметр конфигурации
func (r *ConfigRepository) Get(key string) (string, error) {
	var value string
	query := `SELECT value FROM config_params WHERE key = $1`
	err := r.db.QueryRow(query, key).Scan(&value)

	if err == sql.ErrNoRows {
		return "", nil
	}

	return value, err
}
