package telegram

import (
	"fmt"

	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
)

// Validator валидирует команды перед выполнением
type Validator struct {
	storage  *storage.PostgresStorage
	exchange *exchange.BybitClient
}

func NewValidator(storage *storage.PostgresStorage, exchange *exchange.BybitClient) *Validator {
	return &Validator{
		storage:  storage,
		exchange: exchange,
	}
}

// ValidateBuy валидирует команду покупки
func (v *Validator) ValidateBuy(symbol string, amount float64) error {
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	// Проверяем баланс USDT
	usdtBalance, err := v.exchange.GetBalance("USDT")
	if err != nil {
		return fmt.Errorf("failed to get USDT balance: %w", err)
	}

	if usdtBalance < amount {
		return fmt.Errorf("insufficient USDT balance: have %.2f, need %.2f", usdtBalance, amount)
	}

	// Проверяем риск-лимиты
	limits, err := v.storage.GetRiskLimits()
	if err != nil {
		return fmt.Errorf("failed to get risk limits: %w", err)
	}

	if limits.EnableEmergencyStop {
		return fmt.Errorf("trading is paused (emergency stop enabled)")
	}

	if amount > limits.MaxOrderSizeUSD {
		return fmt.Errorf("order size %.2f exceeds max limit %.2f USDT", amount, limits.MaxOrderSizeUSD)
	}

	// Проверяем текущую экспозицию
	balances, err := v.storage.GetAllBalances()
	if err != nil {
		return fmt.Errorf("failed to get balances: %w", err)
	}

	totalExposure := 0.0
	for _, bal := range balances {
		price, err := v.exchange.GetPrice(bal.Symbol)
		if err != nil {
			continue
		}
		totalExposure += bal.TotalQuantity * price
	}

	if totalExposure+amount > limits.MaxTotalExposure {
		return fmt.Errorf("total exposure %.2f + %.2f exceeds limit %.2f USDT",
			totalExposure, amount, limits.MaxTotalExposure)
	}

	return nil
}

// ValidateSell валидирует команду продажи
func (v *Validator) ValidateSell(symbol string, percent float64) error {
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if percent <= 0 || percent > 100 {
		return fmt.Errorf("percent must be between 1 and 100")
	}

	// Проверяем наличие позиции
	balance, err := v.storage.GetBalance(symbol)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	if balance.AvailableQty <= 0 {
		return fmt.Errorf("no position in %s to sell", symbol)
	}

	// Проверяем риск-лимиты
	limits, err := v.storage.GetRiskLimits()
	if err != nil {
		return fmt.Errorf("failed to get risk limits: %w", err)
	}

	if limits.EnableEmergencyStop {
		return fmt.Errorf("trading is paused (emergency stop enabled)")
	}

	return nil
}

// ValidateGridInit валидирует инициализацию Grid
func (v *Validator) ValidateGridInit(symbol string, levels int, spacing, orderSize float64) error {
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if levels <= 0 || levels > 100 {
		return fmt.Errorf("levels must be between 1 and 100")
	}

	if spacing <= 0 || spacing > 50 {
		return fmt.Errorf("spacing must be between 0.1%% and 50%%")
	}

	if orderSize <= 0 {
		return fmt.Errorf("order size must be positive")
	}

	// Проверяем баланс USDT для Grid
	usdtBalance, err := v.exchange.GetBalance("USDT")
	if err != nil {
		return fmt.Errorf("failed to get USDT balance: %w", err)
	}

	// Grid требует капитал на все уровни
	totalCapitalNeeded := float64(levels) * orderSize
	if usdtBalance < totalCapitalNeeded {
		return fmt.Errorf("insufficient USDT for Grid: need %.2f, have %.2f",
			totalCapitalNeeded, usdtBalance)
	}

	// Проверяем риск-лимиты
	limits, err := v.storage.GetRiskLimits()
	if err != nil {
		return fmt.Errorf("failed to get risk limits: %w", err)
	}

	if limits.EnableEmergencyStop {
		return fmt.Errorf("trading is paused (emergency stop enabled)")
	}

	if orderSize > limits.MaxOrderSizeUSD {
		return fmt.Errorf("order size %.2f exceeds max limit %.2f USDT",
			orderSize, limits.MaxOrderSizeUSD)
	}

	// Проверяем позиционный лимит
	balance, err := v.storage.GetBalance(symbol)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	currentPrice, err := v.exchange.GetPrice(symbol)
	if err != nil {
		return fmt.Errorf("failed to get price: %w", err)
	}

	currentPositionValue := balance.TotalQuantity * currentPrice
	maxAdditionalCapital := limits.MaxPositionSizeUSD - currentPositionValue

	if totalCapitalNeeded > maxAdditionalCapital {
		return fmt.Errorf("Grid capital %.2f would exceed position limit (current: %.2f, max: %.2f)",
			totalCapitalNeeded, currentPositionValue, limits.MaxPositionSizeUSD)
	}

	return nil
}

// ValidateAutoSell валидирует настройки Auto-Sell
func (v *Validator) ValidateAutoSell(symbol string, trigger, sellPercent float64) error {
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if trigger <= 0 {
		return fmt.Errorf("trigger percent must be positive")
	}

	if sellPercent <= 0 || sellPercent > 100 {
		return fmt.Errorf("sell percent must be between 1 and 100")
	}

	if trigger > 1000 {
		return fmt.Errorf("trigger percent seems too high (%.2f%%), max is 1000%%", trigger)
	}

	// Проверяем наличие позиции
	balance, err := v.storage.GetBalance(symbol)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	if balance.TotalQuantity <= 0 {
		return fmt.Errorf("no position in %s for auto-sell", symbol)
	}

	return nil
}

// ValidateSymbol проверяет корректность символа
func (v *Validator) ValidateSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	// Проверяем существование пары на бирже
	_, err := v.exchange.GetPrice(symbol)
	if err != nil {
		return fmt.Errorf("invalid or unsupported symbol %s: %w", symbol, err)
	}

	return nil
}

// CheckAdminPermission проверяет права администратора
func (v *Validator) CheckAdminPermission(userID int64, allowedIDs []int64) error {
	if len(allowedIDs) == 0 {
		// Если список не настроен, разрешаем всем
		return nil
	}

	for _, id := range allowedIDs {
		if id == userID {
			return nil
		}
	}

	return fmt.Errorf("access denied: admin permission required")
}

// GetDefaultSymbol возвращает дефолтный символ из конфига
func (v *Validator) GetDefaultSymbol() string {
	// Попытаемся получить из конфига или используем BTCUSDT
	return "BTCUSDT"
}
