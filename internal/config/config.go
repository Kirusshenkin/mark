package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config содержит все настройки приложения
type Config struct {
	Telegram TelegramConfig
	Bybit    BybitConfig
	Database DatabaseConfig
	AI       AIConfig
	Strategy StrategyConfig
	LogLevel string
}

type TelegramConfig struct {
	BotToken string
	ChatID   int64
}

type BybitConfig struct {
	APIKey    string
	APISecret string
	BaseURL   string
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type AIConfig struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
}

type StrategyConfig struct {
	TradingSymbol          string
	DCAAmount              float64
	DCAInterval            time.Duration
	AutoSellEnabled        bool
	AutoSellTriggerPercent float64
	AutoSellAmountPercent  float64
	PriceCheckInterval     time.Duration
}

// Load загружает конфигурацию из .env файла
func Load() (*Config, error) {
	// Загружаем .env файл (если есть)
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found, using environment variables")
	}

	chatID, err := strconv.ParseInt(getEnv("TELEGRAM_CHAT_ID", "0"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid TELEGRAM_CHAT_ID: %w", err)
	}

	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	maxOpenConns, err := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_OPEN_CONNS: %w", err)
	}

	maxIdleConns, err := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "5"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_IDLE_CONNS: %w", err)
	}

	connMaxLifetime, err := time.ParseDuration(getEnv("DB_CONN_MAX_LIFETIME", "5m"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_CONN_MAX_LIFETIME: %w", err)
	}

	dcaAmount, err := strconv.ParseFloat(getEnv("DCA_AMOUNT", "10"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid DCA_AMOUNT: %w", err)
	}

	dcaInterval, err := time.ParseDuration(getEnv("DCA_INTERVAL", "24h"))
	if err != nil {
		return nil, fmt.Errorf("invalid DCA_INTERVAL: %w", err)
	}

	autoSellEnabled, err := strconv.ParseBool(getEnv("AUTO_SELL_ENABLED", "true"))
	if err != nil {
		return nil, fmt.Errorf("invalid AUTO_SELL_ENABLED: %w", err)
	}

	autoSellTrigger, err := strconv.ParseFloat(getEnv("AUTO_SELL_TRIGGER_PERCENT", "10"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid AUTO_SELL_TRIGGER_PERCENT: %w", err)
	}

	autoSellAmount, err := strconv.ParseFloat(getEnv("AUTO_SELL_AMOUNT_PERCENT", "50"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid AUTO_SELL_AMOUNT_PERCENT: %w", err)
	}

	priceCheckInterval, err := time.ParseDuration(getEnv("PRICE_CHECK_INTERVAL", "5m"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRICE_CHECK_INTERVAL: %w", err)
	}

	config := &Config{
		Telegram: TelegramConfig{
			BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
			ChatID:   chatID,
		},
		Bybit: BybitConfig{
			APIKey:    getEnv("BYBIT_API_KEY", ""),
			APISecret: getEnv("BYBIT_API_SECRET", ""),
			BaseURL:   getEnv("BYBIT_BASE_URL", "https://api.bybit.com"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            dbPort,
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			DBName:          getEnv("DB_NAME", "crypto_trading_bot"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxLifetime: connMaxLifetime,
		},
		AI: AIConfig{
			Provider: getEnv("AI_PROVIDER", "qwen"),
			APIKey:   getEnv("AI_API_KEY", ""),
			BaseURL:  getEnv("AI_BASE_URL", ""),
			Model:    getEnv("AI_MODEL", "qwen-plus"),
		},
		Strategy: StrategyConfig{
			TradingSymbol:          getEnv("TRADING_SYMBOL", "BTCUSDT"),
			DCAAmount:              dcaAmount,
			DCAInterval:            dcaInterval,
			AutoSellEnabled:        autoSellEnabled,
			AutoSellTriggerPercent: autoSellTrigger,
			AutoSellAmountPercent:  autoSellAmount,
			PriceCheckInterval:     priceCheckInterval,
		},
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate проверяет обязательные поля конфигурации
func (c *Config) Validate() error {
	if c.Telegram.BotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if c.Bybit.APIKey == "" {
		return fmt.Errorf("BYBIT_API_KEY is required")
	}
	if c.Bybit.APISecret == "" {
		return fmt.Errorf("BYBIT_API_SECRET is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
