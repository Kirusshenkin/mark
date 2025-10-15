package domain

import "errors"

var (
	// ErrNotFound возвращается когда запись не найдена
	ErrNotFound = errors.New("not found")

	// ErrInvalidInput возвращается при некорректных входных данных
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized возвращается при ошибке авторизации
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInsufficientBalance возвращается при недостаточном балансе
	ErrInsufficientBalance = errors.New("insufficient balance")

	// ErrRiskLimitExceeded возвращается при превышении лимитов риска
	ErrRiskLimitExceeded = errors.New("risk limit exceeded")

	// ErrEmergencyStop возвращается когда активирован emergency stop
	ErrEmergencyStop = errors.New("emergency stop activated")

	// ErrExchangeAPI возвращается при ошибке API биржи
	ErrExchangeAPI = errors.New("exchange API error")

	// ErrDatabaseConnection возвращается при ошибке подключения к БД
	ErrDatabaseConnection = errors.New("database connection error")
)
