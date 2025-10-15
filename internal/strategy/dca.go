package strategy

import (
	"fmt"
	"time"

	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
	"github.com/kirillm/dca-bot/pkg/utils"
)

type DCAStrategy struct {
	exchange      *exchange.BybitClient
	storage       *storage.PostgresStorage
	logger        *utils.Logger
	symbol        string
	amount        float64
	interval      time.Duration
	stopChan      chan bool
	notifyFunc    func(string)
}

func NewDCAStrategy(
	ex *exchange.BybitClient,
	st *storage.PostgresStorage,
	logger *utils.Logger,
	symbol string,
	amount float64,
	interval time.Duration,
	notifyFunc func(string),
) *DCAStrategy {
	return &DCAStrategy{
		exchange:   ex,
		storage:    st,
		logger:     logger,
		symbol:     symbol,
		amount:     amount,
		interval:   interval,
		stopChan:   make(chan bool),
		notifyFunc: notifyFunc,
	}
}

// Start запускает DCA стратегию
func (d *DCAStrategy) Start() {
	d.logger.Info("DCA strategy started for %s with amount %.2f USDT every %s", d.symbol, d.amount, d.interval)

	// Выполняем первую покупку сразу (опционально)
	// if err := d.executeDCA(); err != nil {
	// 	d.logger.Error("Initial DCA execution failed: %v", err)
	// }

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := d.executeDCA(); err != nil {
				d.logger.Error("DCA execution failed: %v", err)
				if d.notifyFunc != nil {
					d.notifyFunc(fmt.Sprintf("❌ DCA failed: %v", err))
				}
			}
		case <-d.stopChan:
			d.logger.Info("DCA strategy stopped")
			return
		}
	}
}

// Stop останавливает DCA стратегию
func (d *DCAStrategy) Stop() {
	d.stopChan <- true
}

// executeDCA выполняет одну DCA покупку
func (d *DCAStrategy) executeDCA() error {
	d.logger.Info("Executing DCA buy for %s", d.symbol)

	// Проверяем баланс USDT
	usdtBalance, err := d.exchange.GetBalance("USDT")
	if err != nil {
		return fmt.Errorf("failed to get USDT balance: %w", err)
	}

	if usdtBalance < d.amount {
		return fmt.Errorf("insufficient USDT balance: have %.2f, need %.2f", usdtBalance, d.amount)
	}

	// Получаем текущую цену
	currentPrice, err := d.exchange.GetPrice(d.symbol)
	if err != nil {
		return fmt.Errorf("failed to get price: %w", err)
	}

	// Рассчитываем количество актива для покупки
	quantity, err := d.exchange.CalculateOrderAmount(d.symbol, d.amount)
	if err != nil {
		return fmt.Errorf("failed to calculate order amount: %w", err)
	}

	d.logger.Info("Buying %.8f %s at price %.2f", quantity, d.symbol, currentPrice)

	// Размещаем рыночный ордер
	orderInfo, err := d.exchange.PlaceOrder(d.symbol, "Buy", quantity)
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	d.logger.Info("Order placed successfully: %s", orderInfo.OrderID)

	// Сохраняем сделку в БД
	trade := &storage.Trade{
		Symbol:    d.symbol,
		Side:      "BUY",
		Quantity:  quantity,
		Price:     currentPrice,
		Amount:    d.amount,
		OrderID:   orderInfo.OrderID,
		Status:    orderInfo.Status,
		CreatedAt: time.Now(),
	}

	if err := d.storage.SaveTrade(trade); err != nil {
		d.logger.Error("Failed to save trade: %v", err)
	}

	// Обновляем баланс
	if err := d.updateBalance(quantity, currentPrice, d.amount); err != nil {
		d.logger.Error("Failed to update balance: %v", err)
	}

	// Отправляем уведомление
	message := fmt.Sprintf(
		"✅ DCA Buy Executed\n\n"+
			"Symbol: %s\n"+
			"Quantity: %.8f\n"+
			"Price: %.2f USDT\n"+
			"Amount: %.2f USDT\n"+
			"Order ID: %s",
		d.symbol, quantity, currentPrice, d.amount, orderInfo.OrderID,
	)

	if d.notifyFunc != nil {
		d.notifyFunc(message)
	}

	return nil
}

// updateBalance обновляет баланс после покупки
func (d *DCAStrategy) updateBalance(quantity, price, amount float64) error {
	balance, err := d.storage.GetBalance(d.symbol)
	if err != nil {
		return err
	}

	// Рассчитываем новую среднюю цену входа
	totalQuantity := balance.TotalQuantity + quantity
	totalInvested := balance.TotalInvested + amount
	avgPrice := totalInvested / totalQuantity

	balance.TotalQuantity = totalQuantity
	balance.AvailableQty = totalQuantity
	balance.AvgEntryPrice = avgPrice
	balance.TotalInvested = totalInvested

	return d.storage.UpdateBalance(balance)
}

// ExecuteManualBuy выполняет ручную покупку
func (d *DCAStrategy) ExecuteManualBuy() error {
	return d.executeDCA()
}

// UpdateAmount обновляет сумму DCA
func (d *DCAStrategy) UpdateAmount(newAmount float64) {
	d.amount = newAmount
	d.logger.Info("DCA amount updated to %.2f USDT", newAmount)
}

// GetStatus возвращает информацию о статусе стратегии
func (d *DCAStrategy) GetStatus() (string, error) {
	balance, err := d.storage.GetBalance(d.symbol)
	if err != nil {
		return "", err
	}

	currentPrice, err := d.exchange.GetPrice(d.symbol)
	if err != nil {
		return "", err
	}

	currentValue := balance.TotalQuantity * currentPrice
	unrealizedProfit := currentValue - balance.TotalInvested
	unrealizedProfitPercent := (unrealizedProfit / balance.TotalInvested) * 100

	status := fmt.Sprintf(
		"📊 DCA Strategy Status\n\n"+
			"Symbol: %s\n"+
			"Total Quantity: %.8f\n"+
			"Avg Entry Price: %.2f USDT\n"+
			"Current Price: %.2f USDT\n"+
			"Total Invested: %.2f USDT\n"+
			"Current Value: %.2f USDT\n"+
			"Unrealized P&L: %.2f USDT (%.2f%%)\n"+
			"Realized Profit: %.2f USDT\n"+
			"DCA Amount: %.2f USDT\n"+
			"DCA Interval: %s",
		d.symbol,
		balance.TotalQuantity,
		balance.AvgEntryPrice,
		currentPrice,
		balance.TotalInvested,
		currentValue,
		unrealizedProfit,
		unrealizedProfitPercent,
		balance.RealizedProfit,
		d.amount,
		d.interval,
	)

	return status, nil
}
