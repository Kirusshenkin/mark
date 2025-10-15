package telegram

import (
	"strings"
	"testing"
	"time"

	"github.com/kirillm/dca-bot/internal/storage"
)

func TestFormatter_T(t *testing.T) {
	tests := []struct {
		name string
		lang Lang
		key  string
		want string
	}{
		{"english status", LangEN, "status", "Status"},
		{"russian status", LangRU, "status", "Статус"},
		{"english error", LangEN, "error", "Error"},
		{"russian error", LangRU, "error", "Ошибка"},
		{"unknown key", LangEN, "unknown_key", "unknown_key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormatter(tt.lang)
			if got := f.T(tt.key); got != tt.want {
				t.Errorf("T() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatter_SetGetLang(t *testing.T) {
	f := NewFormatter(LangEN)

	if f.GetLang() != LangEN {
		t.Error("Initial language should be English")
	}

	f.SetLang(LangRU)

	if f.GetLang() != LangRU {
		t.Error("Language should be Russian after SetLang")
	}
}

func TestFormatter_FormatStatus(t *testing.T) {
	f := NewFormatter(LangEN)

	data := map[string]interface{}{
		"active_assets":   2,
		"strategies":      []string{"DCA", "Grid"},
		"autosell_status": true,
		"grid_active":     false,
		"uptime":          "1h 30m",
	}

	result := f.FormatStatus(data)

	// Проверяем наличие ключевых элементов
	if !strings.Contains(result, "📊") {
		t.Error("Status should contain emoji")
	}
	if !strings.Contains(result, "2") {
		t.Error("Status should contain number of active assets")
	}
	if !strings.Contains(result, "DCA") || !strings.Contains(result, "Grid") {
		t.Error("Status should contain strategy names")
	}
}

func TestFormatter_FormatHistory(t *testing.T) {
	f := NewFormatter(LangEN)

	trades := []storage.Trade{
		{
			Symbol:       "BTCUSDT",
			Side:         "BUY",
			Quantity:     0.001,
			Price:        50000,
			Amount:       50,
			StrategyType: "DCA",
			CreatedAt:    time.Now(),
		},
		{
			Symbol:       "BTCUSDT",
			Side:         "SELL",
			Quantity:     0.0005,
			Price:        55000,
			Amount:       27.5,
			StrategyType: "AUTO_SELL",
			CreatedAt:    time.Now(),
		},
	}

	result := f.FormatHistory(trades, "BTCUSDT", 10)

	// Проверяем наличие ключевых элементов
	if !strings.Contains(result, "📜") {
		t.Error("History should contain emoji")
	}
	if !strings.Contains(result, "BUY") || !strings.Contains(result, "SELL") {
		t.Error("History should contain trade sides")
	}
	if !strings.Contains(result, "BTCUSDT") {
		t.Error("History should contain symbol")
	}
	if !strings.Contains(result, "🟢") || !strings.Contains(result, "🔴") {
		t.Error("History should contain emojis for buy/sell")
	}
}

func TestFormatter_FormatHistory_Empty(t *testing.T) {
	f := NewFormatter(LangEN)

	result := f.FormatHistory([]storage.Trade{}, "", 10)

	if !strings.Contains(result, "No trades") {
		t.Error("Empty history should contain 'No trades' message")
	}
}

func TestFormatter_FormatPortfolio(t *testing.T) {
	f := NewFormatter(LangEN)

	data := map[string]interface{}{
		"total_invested":   1000.0,
		"current_value":    1200.0,
		"realized_profit":  50.0,
		"unrealized_pnl":   150.0,
		"total_pnl":        200.0,
		"return_percent":   20.0,
		"assets": []map[string]interface{}{
			{
				"symbol":        "BTCUSDT",
				"quantity":      0.002,
				"current_price": 50000.0,
				"pnl":           100.0,
				"pnl_percent":   10.0,
			},
		},
	}

	result := f.FormatPortfolio(data)

	// Проверяем наличие ключевых элементов
	if !strings.Contains(result, "💼") {
		t.Error("Portfolio should contain emoji")
	}
	if !strings.Contains(result, "1000") {
		t.Error("Portfolio should contain total invested")
	}
	if !strings.Contains(result, "1200") {
		t.Error("Portfolio should contain current value")
	}
	if !strings.Contains(result, "BTCUSDT") {
		t.Error("Portfolio should contain asset symbols")
	}
}

func TestFormatter_FormatPrice(t *testing.T) {
	f := NewFormatter(LangEN)

	// Без позиции
	result1 := f.FormatPrice("BTCUSDT", 50000, 0, false)

	if !strings.Contains(result1, "💵") {
		t.Error("Price should contain emoji")
	}
	if !strings.Contains(result1, "50000") {
		t.Error("Price should contain current price")
	}

	// С позицией
	result2 := f.FormatPrice("BTCUSDT", 55000, 50000, true)

	if !strings.Contains(result2, "55000") {
		t.Error("Price should contain current price")
	}
	if !strings.Contains(result2, "50000") {
		t.Error("Price should contain avg entry")
	}
	if !strings.Contains(result2, "10.00") {
		t.Error("Price should contain change percentage")
	}
	if !strings.Contains(result2, "📈") {
		t.Error("Price should contain upward trend emoji")
	}
}

func TestFormatter_FormatGridStatus(t *testing.T) {
	f := NewFormatter(LangEN)

	metrics := map[string]interface{}{
		"active_orders":    5,
		"current_price":    50000.0,
		"total_quantity":   0.01,
		"avg_entry_price":  48000.0,
		"total_invested":   480.0,
		"realized_profit":  20.0,
		"unrealized_pnl":   20.0,
		"total_pnl":        40.0,
		"return_percent":   8.33,
	}

	result := f.FormatGridStatus("ETHUSDT", metrics)

	// Проверяем наличие ключевых элементов
	if !strings.Contains(result, "🔷") {
		t.Error("Grid status should contain emoji")
	}
	if !strings.Contains(result, "ETHUSDT") {
		t.Error("Grid status should contain symbol")
	}
	if !strings.Contains(result, "5") {
		t.Error("Grid status should contain active orders count")
	}
	if !strings.Contains(result, "50000") {
		t.Error("Grid status should contain current price")
	}
}

func TestFormatter_FormatRiskStatus(t *testing.T) {
	f := NewFormatter(LangEN)

	limits := &storage.RiskLimit{
		MaxDailyLoss:        1000.0,
		MaxTotalExposure:    10000.0,
		MaxPositionSizeUSD:  5000.0,
		MaxOrderSizeUSD:     1000.0,
		EnableEmergencyStop: false,
	}

	result := f.FormatRiskStatus(limits, 5000.0, 100.0)

	// Проверяем наличие ключевых элементов
	if !strings.Contains(result, "🛡️") {
		t.Error("Risk status should contain emoji")
	}
	if !strings.Contains(result, "10000") {
		t.Error("Risk status should contain max exposure")
	}
	if !strings.Contains(result, "5000") {
		t.Error("Risk status should contain current exposure")
	}
	if !strings.Contains(result, "🟢") {
		t.Error("Risk status should contain green emoji when emergency stop is disabled")
	}
}

func TestFormatter_FormatRiskStatus_EmergencyStop(t *testing.T) {
	f := NewFormatter(LangEN)

	limits := &storage.RiskLimit{
		MaxDailyLoss:        1000.0,
		MaxTotalExposure:    10000.0,
		MaxPositionSizeUSD:  5000.0,
		MaxOrderSizeUSD:     1000.0,
		EnableEmergencyStop: true,
	}

	result := f.FormatRiskStatus(limits, 5000.0, 100.0)

	// Проверяем наличие emergency emoji
	if !strings.Contains(result, "🚨") {
		t.Error("Risk status should contain emergency emoji when emergency stop is enabled")
	}
}

func TestFormatter_FormatError(t *testing.T) {
	f := NewFormatter(LangEN)

	err := f.FormatError(nil)

	// Даже с nil должен быть emoji
	if !strings.Contains(err, "❌") {
		t.Error("Error should contain emoji")
	}
}

func TestFormatter_FormatSuccess(t *testing.T) {
	f := NewFormatter(LangEN)

	result := f.FormatSuccess("Operation completed")

	if !strings.Contains(result, "✅") {
		t.Error("Success should contain emoji")
	}
	if !strings.Contains(result, "Operation completed") {
		t.Error("Success should contain message")
	}
}

func TestFormatter_FormatExecuting(t *testing.T) {
	f := NewFormatter(LangEN)

	result := f.FormatExecuting("buy order")

	if !strings.Contains(result, "🔄") {
		t.Error("Executing should contain emoji")
	}
	if !strings.Contains(result, "buy order") {
		t.Error("Executing should contain action")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds", 30 * time.Second, "30s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours", 2 * time.Hour, "2h 0m"},
		{"hours and minutes", 2*time.Hour + 30*time.Minute, "2h 30m"},
		{"days", 25 * time.Hour, "1d 1h"},
		{"days and hours", 50 * time.Hour, "2d 2h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatDuration(tt.duration); got != tt.want {
				t.Errorf("FormatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatter_Languages(t *testing.T) {
	// Тестируем несколько ключевых переводов для обоих языков
	enFormatter := NewFormatter(LangEN)
	ruFormatter := NewFormatter(LangRU)

	keys := []string{"status", "history", "portfolio", "buy", "sell", "error", "success"}

	for _, key := range keys {
		enTranslation := enFormatter.T(key)
		ruTranslation := ruFormatter.T(key)

		// Убеждаемся, что переводы разные
		if enTranslation == ruTranslation {
			t.Errorf("Translation for key '%s' is the same in both languages: %s", key, enTranslation)
		}

		// Убеждаемся, что не возвращается сам ключ (есть перевод)
		if enTranslation == key {
			t.Errorf("No English translation for key '%s'", key)
		}
		if ruTranslation == key {
			t.Errorf("No Russian translation for key '%s'", key)
		}
	}
}
