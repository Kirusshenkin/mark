package strategy

import (
	"fmt"
	"time"

	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
	"github.com/kirillm/dca-bot/pkg/utils"
)

// RiskManager управляет рисками и лимитами
type RiskManager struct {
	storage  *storage.PostgresStorage
	exchange *exchange.BybitClient
}

func NewRiskManager(storage *storage.PostgresStorage, exchange *exchange.BybitClient) *RiskManager {
	return &RiskManager{
		storage:  storage,
		exchange: exchange,
	}
}

// CheckStopLoss проверяет условия stop-loss
func (r *RiskManager) CheckStopLoss(asset *storage.Asset) (bool, error) {
	if asset.StopLossPercent == 0 {
		return false, nil // Stop-loss не установлен
	}

	balance, err := r.storage.GetBalance(asset.Symbol)
	if err != nil {
		return false, err
	}

	if balance.TotalQuantity == 0 {
		return false, nil // Нет позиции
	}

	currentPrice, err := r.exchange.GetCurrentPrice(asset.Symbol)
	if err != nil {
		return false, err
	}

	// Рассчитываем процент убытка от средней цены входа
	priceDiff := balance.AvgEntryPrice - currentPrice
	lossPercent := (priceDiff / balance.AvgEntryPrice) * 100

	if lossPercent >= asset.StopLossPercent {
		utils.LogWarn(fmt.Sprintf("Stop-Loss срабатывает для %s: убыток %.2f%% (лимит %.2f%%)",
			asset.Symbol, lossPercent, asset.StopLossPercent))

		// Продаем всю позицию
		if err := r.executeStopLoss(asset, balance, currentPrice); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

// executeStopLoss исполняет stop-loss
func (r *RiskManager) executeStopLoss(asset *storage.Asset, balance *storage.Balance, currentPrice float64) error {
	utils.LogWarn(fmt.Sprintf("Исполнение Stop-Loss для %s: продажа %.8f по цене %.8f",
		asset.Symbol, balance.AvailableQty, currentPrice))

	// Размещаем рыночный ордер на продажу
	orderInfo, err := r.exchange.PlaceOrder(asset.Symbol, "SELL", balance.AvailableQty)
	if err != nil {
		return fmt.Errorf("не удалось разместить stop-loss ордер: %w", err)
	}

	executedPrice := currentPrice // Используем текущую цену для рыночного ордера

	// Сохраняем сделку
	trade := &storage.Trade{
		Symbol:       asset.Symbol,
		Side:         "SELL",
		Quantity:     balance.AvailableQty,
		Price:        executedPrice,
		Amount:       balance.AvailableQty * executedPrice,
		OrderID:      orderInfo.OrderID,
		Status:       "FILLED",
		StrategyType: "STOP_LOSS",
		CreatedAt:    time.Now(),
	}
	if err := r.storage.SaveTrade(trade); err != nil {
		return fmt.Errorf("не удалось сохранить сделку: %w", err)
	}

	// Обновляем баланс
	totalAmount := balance.AvailableQty * executedPrice
	costBasis := balance.AvailableQty * balance.AvgEntryPrice
	loss := totalAmount - costBasis

	balance.TotalQuantity = 0
	balance.AvailableQty = 0
	balance.TotalSold += totalAmount
	balance.RealizedProfit += loss

	if err := r.storage.UpdateBalance(balance); err != nil {
		return fmt.Errorf("не удалось обновить баланс: %w", err)
	}

	utils.LogInfo(fmt.Sprintf("Stop-Loss исполнен для %s: убыток %.2f USDT", asset.Symbol, loss))
	return nil
}

// CheckTakeProfit проверяет условия take-profit
func (r *RiskManager) CheckTakeProfit(asset *storage.Asset) (bool, error) {
	if asset.TakeProfitPercent == 0 {
		return false, nil // Take-profit не установлен
	}

	balance, err := r.storage.GetBalance(asset.Symbol)
	if err != nil {
		return false, err
	}

	if balance.TotalQuantity == 0 {
		return false, nil // Нет позиции
	}

	currentPrice, err := r.exchange.GetCurrentPrice(asset.Symbol)
	if err != nil {
		return false, err
	}

	// Рассчитываем процент прибыли от средней цены входа
	priceDiff := currentPrice - balance.AvgEntryPrice
	profitPercent := (priceDiff / balance.AvgEntryPrice) * 100

	if profitPercent >= asset.TakeProfitPercent {
		utils.LogInfo(fmt.Sprintf("Take-Profit срабатывает для %s: прибыль %.2f%% (цель %.2f%%)",
			asset.Symbol, profitPercent, asset.TakeProfitPercent))

		// Продаем всю позицию
		if err := r.executeTakeProfit(asset, balance, currentPrice); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

// executeTakeProfit исполняет take-profit
func (r *RiskManager) executeTakeProfit(asset *storage.Asset, balance *storage.Balance, currentPrice float64) error {
	utils.LogInfo(fmt.Sprintf("Исполнение Take-Profit для %s: продажа %.8f по цене %.8f",
		asset.Symbol, balance.AvailableQty, currentPrice))

	// Размещаем рыночный ордер на продажу
	orderInfo, err := r.exchange.PlaceOrder(asset.Symbol, "SELL", balance.AvailableQty)
	if err != nil {
		return fmt.Errorf("не удалось разместить take-profit ордер: %w", err)
	}

	executedPrice := currentPrice // Используем текущую цену для рыночного ордера

	// Сохраняем сделку
	trade := &storage.Trade{
		Symbol:       asset.Symbol,
		Side:         "SELL",
		Quantity:     balance.AvailableQty,
		Price:        executedPrice,
		Amount:       balance.AvailableQty * executedPrice,
		OrderID:      orderInfo.OrderID,
		Status:       "FILLED",
		StrategyType: "TAKE_PROFIT",
		CreatedAt:    time.Now(),
	}
	if err := r.storage.SaveTrade(trade); err != nil {
		return fmt.Errorf("не удалось сохранить сделку: %w", err)
	}

	// Обновляем баланс
	totalAmount := balance.AvailableQty * executedPrice
	costBasis := balance.AvailableQty * balance.AvgEntryPrice
	profit := totalAmount - costBasis

	balance.TotalQuantity = 0
	balance.AvailableQty = 0
	balance.TotalSold += totalAmount
	balance.RealizedProfit += profit

	if err := r.storage.UpdateBalance(balance); err != nil {
		return fmt.Errorf("не удалось обновить баланс: %w", err)
	}

	utils.LogInfo(fmt.Sprintf("Take-Profit исполнен для %s: прибыль %.2f USDT", asset.Symbol, profit))
	return nil
}

// CheckRiskLimits проверяет глобальные лимиты риска
func (r *RiskManager) CheckRiskLimits() (bool, string, error) {
	limits, err := r.storage.GetRiskLimits()
	if err != nil {
		return false, "", err
	}

	if limits.EnableEmergencyStop {
		return false, "Emergency stop активирован", nil
	}

	// Проверяем общую экспозицию
	totalExposure, err := r.calculateTotalExposure()
	if err != nil {
		return false, "", err
	}

	if totalExposure > limits.MaxTotalExposure {
		return false, fmt.Sprintf("Превышен лимит общей экспозиции: %.2f > %.2f", totalExposure, limits.MaxTotalExposure), nil
	}

	// Проверяем дневные убытки
	dailyLoss, err := r.calculateDailyLoss()
	if err != nil {
		return false, "", err
	}

	if dailyLoss > limits.MaxDailyLoss {
		return false, fmt.Sprintf("Превышен лимит дневных убытков: %.2f > %.2f", dailyLoss, limits.MaxDailyLoss), nil
	}

	return true, "", nil
}

// calculateTotalExposure рассчитывает общую экспозицию
func (r *RiskManager) calculateTotalExposure() (float64, error) {
	balances, err := r.storage.GetAllBalances()
	if err != nil {
		return 0, err
	}

	totalExposure := 0.0
	for _, balance := range balances {
		currentPrice, err := r.exchange.GetCurrentPrice(balance.Symbol)
		if err != nil {
			utils.LogError(fmt.Sprintf("Не удалось получить цену для %s: %v", balance.Symbol, err))
			continue
		}
		totalExposure += balance.TotalQuantity * currentPrice
	}

	return totalExposure, nil
}

// calculateDailyLoss рассчитывает дневные убытки
func (r *RiskManager) calculateDailyLoss() (float64, error) {
	balances, err := r.storage.GetAllBalances()
	if err != nil {
		return 0, err
	}

	totalDailyLoss := 0.0
	for _, balance := range balances {
		if balance.RealizedProfit < 0 {
			// Проверяем, что убыток был за последние 24 часа
			// Здесь упрощенная проверка, в реальности нужно проверять timestamp сделок
			totalDailyLoss += -balance.RealizedProfit
		}
	}

	return totalDailyLoss, nil
}

// ValidateOrderSize проверяет размер ордера
func (r *RiskManager) ValidateOrderSize(symbol string, orderSizeUSD float64) (bool, error) {
	limits, err := r.storage.GetRiskLimits()
	if err != nil {
		return false, err
	}

	if orderSizeUSD > limits.MaxOrderSizeUSD {
		return false, fmt.Errorf("размер ордера %.2f превышает лимит %.2f", orderSizeUSD, limits.MaxOrderSizeUSD)
	}

	// Проверяем размер позиции
	balance, err := r.storage.GetBalance(symbol)
	if err != nil {
		return false, err
	}

	currentPrice, err := r.exchange.GetCurrentPrice(symbol)
	if err != nil {
		return false, err
	}

	currentPositionSize := balance.TotalQuantity * currentPrice
	newPositionSize := currentPositionSize + orderSizeUSD

	if newPositionSize > limits.MaxPositionSizeUSD {
		return false, fmt.Errorf("новый размер позиции %.2f превысит лимит %.2f", newPositionSize, limits.MaxPositionSizeUSD)
	}

	return true, nil
}

// MonitorAllRisks проверяет все риски для всех активов
func (r *RiskManager) MonitorAllRisks() error {
	// Проверяем глобальные лимиты
	ok, message, err := r.CheckRiskLimits()
	if err != nil {
		return err
	}
	if !ok {
		utils.LogWarn(fmt.Sprintf("Глобальное ограничение риска: %s", message))
		return fmt.Errorf("риск-лимит: %s", message)
	}

	// Получаем все активные активы
	assets, err := r.storage.GetEnabledAssets()
	if err != nil {
		return err
	}

	// Проверяем stop-loss и take-profit для каждого актива
	for _, asset := range assets {
		// Проверяем stop-loss
		stopLossTriggered, err := r.CheckStopLoss(&asset)
		if err != nil {
			utils.LogError(fmt.Sprintf("Ошибка проверки stop-loss для %s: %v", asset.Symbol, err))
			continue
		}
		if stopLossTriggered {
			utils.LogWarn(fmt.Sprintf("Stop-Loss сработал для %s, актив деактивирован", asset.Symbol))
			// Опционально: деактивируем актив после stop-loss
			if err := r.storage.DisableAsset(asset.Symbol); err != nil {
				utils.LogError(fmt.Sprintf("Не удалось деактивировать актив %s: %v", asset.Symbol, err))
			}
			continue
		}

		// Проверяем take-profit
		takeProfitTriggered, err := r.CheckTakeProfit(&asset)
		if err != nil {
			utils.LogError(fmt.Sprintf("Ошибка проверки take-profit для %s: %v", asset.Symbol, err))
			continue
		}
		if takeProfitTriggered {
			utils.LogInfo(fmt.Sprintf("Take-Profit сработал для %s", asset.Symbol))
		}
	}

	return nil
}

// SetEmergencyStop устанавливает флаг экстренной остановки
func (r *RiskManager) SetEmergencyStop(enabled bool) error {
	limits, err := r.storage.GetRiskLimits()
	if err != nil {
		return err
	}

	limits.EnableEmergencyStop = enabled
	if err := r.storage.UpdateRiskLimits(limits); err != nil {
		return err
	}

	if enabled {
		utils.LogWarn("ЭКСТРЕННАЯ ОСТАНОВКА АКТИВИРОВАНА")
	} else {
		utils.LogInfo("Экстренная остановка деактивирована")
	}

	return nil
}

// GetRiskStatus возвращает текущий статус рисков
func (r *RiskManager) GetRiskStatus() (map[string]interface{}, error) {
	limits, err := r.storage.GetRiskLimits()
	if err != nil {
		return nil, err
	}

	totalExposure, err := r.calculateTotalExposure()
	if err != nil {
		return nil, err
	}

	dailyLoss, err := r.calculateDailyLoss()
	if err != nil {
		return nil, err
	}

	status := map[string]interface{}{
		"emergency_stop_enabled": limits.EnableEmergencyStop,
		"total_exposure":         totalExposure,
		"max_total_exposure":     limits.MaxTotalExposure,
		"exposure_percent":       (totalExposure / limits.MaxTotalExposure) * 100,
		"daily_loss":             dailyLoss,
		"max_daily_loss":         limits.MaxDailyLoss,
		"daily_loss_percent":     (dailyLoss / limits.MaxDailyLoss) * 100,
		"max_position_size_usd":  limits.MaxPositionSizeUSD,
		"max_order_size_usd":     limits.MaxOrderSizeUSD,
	}

	return status, nil
}
