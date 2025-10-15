package strategy

import (
	"fmt"
	"time"

	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
	"github.com/kirillm/dca-bot/pkg/utils"
)

type AutoSellStrategy struct {
	exchange           *exchange.BybitClient
	storage            *storage.PostgresStorage
	logger             *utils.Logger
	symbol             string
	triggerPercent     float64  // процент роста для активации продажи
	sellAmountPercent  float64  // процент позиции для продажи
	checkInterval      time.Duration
	enabled            bool
	stopChan           chan bool
	notifyFunc         func(string)
}

func NewAutoSellStrategy(
	ex *exchange.BybitClient,
	st *storage.PostgresStorage,
	logger *utils.Logger,
	symbol string,
	triggerPercent float64,
	sellAmountPercent float64,
	checkInterval time.Duration,
	notifyFunc func(string),
) *AutoSellStrategy {
	return &AutoSellStrategy{
		exchange:          ex,
		storage:           st,
		logger:            logger,
		symbol:            symbol,
		triggerPercent:    triggerPercent,
		sellAmountPercent: sellAmountPercent,
		checkInterval:     checkInterval,
		enabled:           true,
		stopChan:          make(chan bool),
		notifyFunc:        notifyFunc,
	}
}

// Start запускает Auto-Sell стратегию
func (a *AutoSellStrategy) Start() {
	a.logger.Info("Auto-Sell strategy started for %s with trigger %.2f%% and sell amount %.2f%%",
		a.symbol, a.triggerPercent, a.sellAmountPercent)

	ticker := time.NewTicker(a.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if a.enabled {
				if err := a.checkAndExecuteSell(); err != nil {
					a.logger.Error("Auto-Sell check failed: %v", err)
				}
			}
		case <-a.stopChan:
			a.logger.Info("Auto-Sell strategy stopped")
			return
		}
	}
}

// Stop останавливает Auto-Sell стратегию
func (a *AutoSellStrategy) Stop() {
	a.stopChan <- true
}

// Enable включает Auto-Sell
func (a *AutoSellStrategy) Enable() {
	a.enabled = true
	a.logger.Info("Auto-Sell enabled")
	if a.notifyFunc != nil {
		a.notifyFunc("✅ Auto-Sell enabled")
	}
}

// Disable выключает Auto-Sell
func (a *AutoSellStrategy) Disable() {
	a.enabled = false
	a.logger.Info("Auto-Sell disabled")
	if a.notifyFunc != nil {
		a.notifyFunc("⏸ Auto-Sell disabled")
	}
}

// IsEnabled возвращает статус Auto-Sell
func (a *AutoSellStrategy) IsEnabled() bool {
	return a.enabled
}

// checkAndExecuteSell проверяет условия и выполняет продажу
func (a *AutoSellStrategy) checkAndExecuteSell() error {
	// Получаем баланс
	balance, err := a.storage.GetBalance(a.symbol)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	// Если нет позиции, ничего не делаем
	if balance.AvailableQty <= 0 {
		return nil
	}

	// Получаем текущую цену
	currentPrice, err := a.exchange.GetPrice(a.symbol)
	if err != nil {
		return fmt.Errorf("failed to get price: %w", err)
	}

	// Рассчитываем процент прибыли
	profitPercent := ((currentPrice - balance.AvgEntryPrice) / balance.AvgEntryPrice) * 100

	a.logger.Debug("Current price: %.2f, Avg entry: %.2f, Profit: %.2f%%",
		currentPrice, balance.AvgEntryPrice, profitPercent)

	// Проверяем условие для продажи
	if profitPercent >= a.triggerPercent {
		return a.executeSell(balance, currentPrice, profitPercent)
	}

	return nil
}

// executeSell выполняет продажу
func (a *AutoSellStrategy) executeSell(balance *storage.Balance, currentPrice, profitPercent float64) error {
	// Рассчитываем количество для продажи
	sellQuantity := balance.AvailableQty * (a.sellAmountPercent / 100)

	a.logger.Info("Executing Auto-Sell: %.8f %s at price %.2f (profit: %.2f%%)",
		sellQuantity, a.symbol, currentPrice, profitPercent)

	// Размещаем ордер на продажу
	orderInfo, err := a.exchange.PlaceOrder(a.symbol, "Sell", sellQuantity)
	if err != nil {
		return fmt.Errorf("failed to place sell order: %w", err)
	}

	a.logger.Info("Sell order placed successfully: %s", orderInfo.OrderID)

	// Рассчитываем сумму продажи и прибыль
	sellAmount := sellQuantity * currentPrice
	profit := sellQuantity * (currentPrice - balance.AvgEntryPrice)

	// Сохраняем сделку в БД
	trade := &storage.Trade{
		Symbol:    a.symbol,
		Side:      "SELL",
		Quantity:  sellQuantity,
		Price:     currentPrice,
		Amount:    sellAmount,
		OrderID:   orderInfo.OrderID,
		Status:    orderInfo.Status,
		CreatedAt: time.Now(),
	}

	if err := a.storage.SaveTrade(trade); err != nil {
		a.logger.Error("Failed to save trade: %v", err)
	}

	// Обновляем баланс
	if err := a.updateBalanceAfterSell(balance, sellQuantity, sellAmount, profit); err != nil {
		a.logger.Error("Failed to update balance: %v", err)
	}

	// Отправляем уведомление
	message := fmt.Sprintf(
		"💰 Auto-Sell Executed\n\n"+
			"Symbol: %s\n"+
			"Sold Quantity: %.8f (%.0f%%)\n"+
			"Price: %.2f USDT\n"+
			"Amount: %.2f USDT\n"+
			"Profit: %.2f USDT (%.2f%%)\n"+
			"Order ID: %s\n\n"+
			"Remaining: %.8f %s",
		a.symbol,
		sellQuantity,
		a.sellAmountPercent,
		currentPrice,
		sellAmount,
		profit,
		profitPercent,
		orderInfo.OrderID,
		balance.AvailableQty-sellQuantity,
		a.symbol,
	)

	if a.notifyFunc != nil {
		a.notifyFunc(message)
	}

	return nil
}

// updateBalanceAfterSell обновляет баланс после продажи
func (a *AutoSellStrategy) updateBalanceAfterSell(balance *storage.Balance, soldQty, sellAmount, profit float64) error {
	balance.AvailableQty -= soldQty
	balance.TotalQuantity -= soldQty
	balance.TotalSold += sellAmount
	balance.RealizedProfit += profit

	return a.storage.UpdateBalance(balance)
}

// UpdateTriggerPercent обновляет процент триггера
func (a *AutoSellStrategy) UpdateTriggerPercent(newPercent float64) {
	a.triggerPercent = newPercent
	a.logger.Info("Auto-Sell trigger updated to %.2f%%", newPercent)
	if a.notifyFunc != nil {
		a.notifyFunc(fmt.Sprintf("✅ Auto-Sell trigger updated to %.2f%%", newPercent))
	}
}

// UpdateSellAmountPercent обновляет процент продажи
func (a *AutoSellStrategy) UpdateSellAmountPercent(newPercent float64) {
	a.sellAmountPercent = newPercent
	a.logger.Info("Auto-Sell amount updated to %.2f%%", newPercent)
	if a.notifyFunc != nil {
		a.notifyFunc(fmt.Sprintf("✅ Auto-Sell amount updated to %.2f%%", newPercent))
	}
}

// GetStatus возвращает информацию о статусе Auto-Sell
func (a *AutoSellStrategy) GetStatus() (string, error) {
	balance, err := a.storage.GetBalance(a.symbol)
	if err != nil {
		return "", err
	}

	currentPrice, err := a.exchange.GetPrice(a.symbol)
	if err != nil {
		return "", err
	}

	profitPercent := 0.0
	if balance.AvgEntryPrice > 0 {
		profitPercent = ((currentPrice - balance.AvgEntryPrice) / balance.AvgEntryPrice) * 100
	}

	enabledStatus := "Disabled ⏸"
	if a.enabled {
		enabledStatus = "Enabled ✅"
	}

	status := fmt.Sprintf(
		"💰 Auto-Sell Status\n\n"+
			"Status: %s\n"+
			"Trigger: %.2f%%\n"+
			"Sell Amount: %.2f%%\n"+
			"Current Profit: %.2f%%\n"+
			"Will trigger at: %.2f USDT\n"+
			"Check Interval: %s",
		enabledStatus,
		a.triggerPercent,
		a.sellAmountPercent,
		profitPercent,
		balance.AvgEntryPrice*(1+a.triggerPercent/100),
		a.checkInterval,
	)

	return status, nil
}

// ExecuteManualSell выполняет ручную продажу указанного процента позиции
func (a *AutoSellStrategy) ExecuteManualSell(percent float64) error {
	balance, err := a.storage.GetBalance(a.symbol)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	if balance.AvailableQty <= 0 {
		return fmt.Errorf("no available quantity to sell")
	}

	currentPrice, err := a.exchange.GetPrice(a.symbol)
	if err != nil {
		return fmt.Errorf("failed to get price: %w", err)
	}

	profitPercent := ((currentPrice - balance.AvgEntryPrice) / balance.AvgEntryPrice) * 100

	// Временно меняем процент продажи
	oldPercent := a.sellAmountPercent
	a.sellAmountPercent = percent

	err = a.executeSell(balance, currentPrice, profitPercent)

	// Восстанавливаем процент
	a.sellAmountPercent = oldPercent

	return err
}
