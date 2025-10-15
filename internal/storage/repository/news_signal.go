package repository

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/kirillm/dca-bot/internal/domain"
)

// NewsSignalRepository управляет новостными сигналами
type NewsSignalRepository struct {
	db *sql.DB
}

// NewNewsSignalRepository создает новый репозиторий
func NewNewsSignalRepository(db *sql.DB) *NewsSignalRepository {
	return &NewsSignalRepository{db: db}
}

// Save сохраняет новостной сигнал
func (r *NewsSignalRepository) Save(signal *domain.NewsSignal) error {
	if signal.Timestamp.IsZero() {
		signal.Timestamp = time.Now()
	}

	query := `
		INSERT INTO news_signals (
			timestamp, source, headline, url, sentiment, sentiment_score,
			topics, signal, symbols, processed
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	return r.db.QueryRow(
		query,
		signal.Timestamp,
		signal.Source,
		signal.Headline,
		signal.URL,
		signal.Sentiment,
		signal.SentimentScore,
		pq.Array(signal.Topics),
		signal.Signal,
		pq.Array(signal.Symbols),
		signal.Processed,
	).Scan(&signal.ID)
}

// GetUnprocessed получает необработанные сигналы
func (r *NewsSignalRepository) GetUnprocessed(limit int) ([]domain.NewsSignal, error) {
	query := `
		SELECT id, timestamp, source, headline, url, sentiment, sentiment_score,
		       topics, signal, symbols, processed
		FROM news_signals
		WHERE processed = false
		ORDER BY timestamp DESC
		LIMIT $1
	`
	return r.query(query, limit)
}

// GetRecent получает последние N сигналов
func (r *NewsSignalRepository) GetRecent(limit int) ([]domain.NewsSignal, error) {
	query := `
		SELECT id, timestamp, source, headline, url, sentiment, sentiment_score,
		       topics, signal, symbols, processed
		FROM news_signals
		ORDER BY timestamp DESC
		LIMIT $1
	`
	return r.query(query, limit)
}

// GetBySentiment получает сигналы по сентименту
func (r *NewsSignalRepository) GetBySentiment(sentiment string, limit int) ([]domain.NewsSignal, error) {
	query := `
		SELECT id, timestamp, source, headline, url, sentiment, sentiment_score,
		       topics, signal, symbols, processed
		FROM news_signals
		WHERE sentiment = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	return r.query(query, sentiment, limit)
}

// MarkAsProcessed помечает сигнал как обработанный
func (r *NewsSignalRepository) MarkAsProcessed(id int64) error {
	query := `UPDATE news_signals SET processed = true WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// GetSentimentStats получает статистику сентимента
func (r *NewsSignalRepository) GetSentimentStats(since time.Time) (map[string]int, error) {
	query := `
		SELECT sentiment, COUNT(*) as count
		FROM news_signals
		WHERE timestamp >= $1
		GROUP BY sentiment
	`
	rows, err := r.db.Query(query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var sentiment string
		var count int
		if err := rows.Scan(&sentiment, &count); err != nil {
			return nil, err
		}
		stats[sentiment] = count
	}

	return stats, rows.Err()
}

// query helper для выполнения запросов
func (r *NewsSignalRepository) query(query string, args ...interface{}) ([]domain.NewsSignal, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signals []domain.NewsSignal
	for rows.Next() {
		var s domain.NewsSignal
		err := rows.Scan(
			&s.ID,
			&s.Timestamp,
			&s.Source,
			&s.Headline,
			&s.URL,
			&s.Sentiment,
			&s.SentimentScore,
			pq.Array(&s.Topics),
			&s.Signal,
			pq.Array(&s.Symbols),
			&s.Processed,
		)
		if err != nil {
			return nil, err
		}
		signals = append(signals, s)
	}

	return signals, rows.Err()
}
