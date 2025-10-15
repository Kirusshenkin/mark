package strategy

import (
	"fmt"
	"math"
	"time"

	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
	"github.com/kirillm/dca-bot/pkg/utils"
)

// GridStrategy реализует Grid торговую стратегию
type GridStrategy struct {
	storage  *storage.PostgresStorage
	exchange *exchange.BybitClient
}

func NewGridStrategy(storage *storage.PostgresStorage, exchange *exchange.BybitClient) *GridStrategy {
	return &GridStrategy{
		storage:  storage,
		exchange: exchange,
	}
}

// InitializeGrid создает начальную сетку ордеров
func (g *GridStrategy) InitializeGrid(asset *storage.Asset) error {
	utils.LogInfo(fmt.Sprintf("Инициализация Grid для %s с %d уровнями", asset.Symbol, asset.GridLevels))

	// Получаем текущую цену
	currentPrice, err := g.exchange.GetCurrentPrice(asset.Symbol)
	if err != nil {
		return fmt.Errorf("не удалось получить текущую цену: %w", err)
	}

	utils.LogInfo(fmt.Sprintf("Текущая цена %s: %.8f", asset.Symbol, currentPrice))

	// Отменяем все существующие grid ордера
	if err := g.storage.CancelGridOrders(asset.Symbol); err != nil {
		return fmt.Errorf("не удалось отменить существующие ордера: %w", err)
	}

	// Рассчитываем уровни сетки
	gridLevels := g.calculateGridLevels(currentPrice, asset.GridLevels, asset.GridSpacingPercent)

	// Размещаем buy ордера ниже текущей цены
	for i := 0; i < len(gridLevels)/2; i++ {
		level := -i - 1
		price := gridLevels[i]

		order := &storage.GridOrder{
			Symbol:    asset.Symbol,
			Level:     level,
			Side:      "BUY",
			Price:     price,
			Quantity:  asset.GridOrderSize / price, // Количество монет по цене
			Status:    "PENDING",
			CreatedAt: time.Now(),
		}

		if err := g.storage.SaveGridOrder(order); err != nil {
			utils.LogError(fmt.Sprintf("Не удалось сохранить buy ордер: %v", err))
			continue
		}

		utils.LogInfo(fmt.Sprintf("Создан buy ордер: уровень %d, цена %.8f, количество %.8f", level, price, order.Quantity))
	}

	// Размещаем sell ордера выше текущей цены
	for i := len(gridLevels) / 2; i < len(gridLevels); i++ {
		level := i - len(gridLevels)/2 + 1
		price := gridLevels[i]

		order := &storage.GridOrder{
			Symbol:    asset.Symbol,
			Level:     level,
			Side:      "SELL",
			Price:     price,
			Quantity:  asset.GridOrderSize / price,
			Status:    "PENDING",
			CreatedAt: time.Now(),
		}

		if err := g.storage.SaveGridOrder(order); err != nil {
			utils.LogError(fmt.Sprintf("Не удалось сохранить sell ордер: %v", err))
			continue
		}

		utils.LogInfo(fmt.Sprintf("Создан sell ордер: уровень %d, цена %.8f, количество %.8f", level, price, order.Quantity))
	}

	utils.LogInfo(fmt.Sprintf("Grid инициализирована для %s: %d уровней", asset.Symbol, asset.GridLevels))
	return nil
}

// calculateGridLevels рассчитывает ценовые уровни для сетки
func (g *GridStrategy) calculateGridLevels(currentPrice float64, numLevels int, spacingPercent float64) []float64 {
	levels := make([]float64, numLevels)
	halfLevels := numLevels / 2

	// Уровни ниже текущей цены (buy)
	for i := 0; i < halfLevels; i++ {
		step := float64(i + 1)
		levels[i] = currentPrice * (1 - (spacingPercent/100)*step)
	}

	// Уровни выше текущей цены (sell)
	for i := halfLevels; i < numLevels; i++ {
		step := float64(i - halfLevels + 1)
		levels[i] = currentPrice * (1 + (spacingPercent/100)*step)
	}

	return levels
}

// MonitorGrid проверяет и обновляет состояние сетки
func (g *GridStrategy) MonitorGrid(asset *storage.Asset) error {
	currentPrice, err := g.exchange.GetCurrentPrice(asset.Symbol)
	if err != nil {
		return fmt.Errorf("не удалось получить текущую цену: %w", err)
	}

	// Получаем активные ордера
	activeOrders, err := g.storage.GetActiveGridOrders(asset.Symbol)
	if err != nil {
		return fmt.Errorf("не удалось получить активные ордера: %w", err)
	}

	if len(activeOrders) == 0 {
		utils.LogInfo(fmt.Sprintf("Нет активных Grid ордеров для %s, инициализация новой сетки", asset.Symbol))
		return g.InitializeGrid(asset)
	}

	// Проверяем каждый ордер
	for _, order := range activeOrders {
		// Проверяем, должен ли ордер быть исполнен
		shouldExecute := false
		if order.Side == "BUY" && currentPrice <= order.Price {
			shouldExecute = true
		} else if order.Side == "SELL" && currentPrice >= order.Price {
			shouldExecute = true
		}

		if shouldExecute {
			if err := g.executeGridOrder(&order, asset, currentPrice); err != nil {
				utils.LogError(fmt.Sprintf("Ошибка выполнения Grid ордера: %v", err))
				continue
			}
		}
	}

	return nil
}

// executeGridOrder исполняет Grid ордер
func (g *GridStrategy) executeGridOrder(order *storage.GridOrder, asset *storage.Asset, currentPrice float64) error {
	utils.LogInfo(fmt.Sprintf("Исполнение Grid ордера: %s %s %.8f @ %.8f", order.Symbol, order.Side, order.Quantity, currentPrice))

	// Выполняем сделку через биржу
	orderInfo, err := g.exchange.PlaceOrder(order.Symbol, order.Side, order.Quantity)
	if err != nil {
		return fmt.Errorf("не удалось разместить ордер: %w", err)
	}

	executedPrice := currentPrice // Используем текущую цену как приближение для рыночного ордера

	// Обновляем статус ордера
	order.Status = "FILLED"
	order.OrderID = orderInfo.OrderID
	order.FilledQty = order.Quantity
	order.FilledPrice = executedPrice
	if err := g.storage.UpdateGridOrder(order); err != nil {
		return fmt.Errorf("не удалось обновить ордер: %w", err)
	}

	// Сохраняем сделку в историю
	trade := &storage.Trade{
		Symbol:       order.Symbol,
		Side:         order.Side,
		Quantity:     order.Quantity,
		Price:        executedPrice,
		Amount:       order.Quantity * executedPrice,
		OrderID:      orderInfo.OrderID,
		Status:       "FILLED",
		StrategyType: "GRID",
		GridLevel:    order.Level,
		CreatedAt:    time.Now(),
	}
	if err := g.storage.SaveTrade(trade); err != nil {
		return fmt.Errorf("не удалось сохранить сделку: %w", err)
	}

	// Обновляем баланс
	if err := g.updateBalanceAfterGridTrade(order, executedPrice); err != nil {
		return fmt.Errorf("не удалось обновить баланс: %w", err)
	}

	// Создаем противоположный ордер
	if err := g.createCounterOrder(order, asset, executedPrice); err != nil {
		return fmt.Errorf("не удалось создать противоположный ордер: %w", err)
	}

	utils.LogInfo(fmt.Sprintf("Grid ордер успешно исполнен: %s %s", order.Symbol, order.Side))
	return nil
}

// createCounterOrder создает противоположный ордер после исполнения
func (g *GridStrategy) createCounterOrder(executedOrder *storage.GridOrder, asset *storage.Asset, executedPrice float64) error {
	var newSide string
	var newPrice float64
	var newLevel int

	if executedOrder.Side == "BUY" {
		// После покупки создаем ордер на продажу выше
		newSide = "SELL"
		newPrice = executedPrice * (1 + asset.GridSpacingPercent/100)
		newLevel = executedOrder.Level + 1
	} else {
		// После продажи создаем ордер на покупку ниже
		newSide = "BUY"
		newPrice = executedPrice * (1 - asset.GridSpacingPercent/100)
		newLevel = executedOrder.Level - 1
	}

	newOrder := &storage.GridOrder{
		Symbol:    executedOrder.Symbol,
		Level:     newLevel,
		Side:      newSide,
		Price:     newPrice,
		Quantity:  asset.GridOrderSize / newPrice,
		Status:    "PENDING",
		CreatedAt: time.Now(),
	}

	if err := g.storage.SaveGridOrder(newOrder); err != nil {
		return fmt.Errorf("не удалось создать новый ордер: %w", err)
	}

	utils.LogInfo(fmt.Sprintf("Создан новый Grid ордер: уровень %d, %s %.8f @ %.8f", newLevel, newSide, newOrder.Quantity, newPrice))
	return nil
}

// updateBalanceAfterGridTrade обновляет баланс после исполнения Grid сделки
func (g *GridStrategy) updateBalanceAfterGridTrade(order *storage.GridOrder, executedPrice float64) error {
	balance, err := g.storage.GetBalance(order.Symbol)
	if err != nil {
		return err
	}

	if order.Side == "BUY" {
		// При покупке увеличиваем количество и инвестиции
		totalAmount := order.Quantity * executedPrice
		newTotalQty := balance.TotalQuantity + order.Quantity
		newInvested := balance.TotalInvested + totalAmount

		// Пересчитываем среднюю цену входа
		if newTotalQty > 0 {
			balance.AvgEntryPrice = newInvested / newTotalQty
		}

		balance.TotalQuantity = newTotalQty
		balance.AvailableQty = newTotalQty
		balance.TotalInvested = newInvested
	} else {
		// При продаже уменьшаем количество и фиксируем прибыль
		totalAmount := order.Quantity * executedPrice
		balance.TotalQuantity -= order.Quantity
		balance.AvailableQty = balance.TotalQuantity

		// Рассчитываем прибыль от продажи
		costBasis := order.Quantity * balance.AvgEntryPrice
		profit := totalAmount - costBasis

		balance.TotalSold += totalAmount
		balance.RealizedProfit += profit
	}

	return g.storage.UpdateBalance(balance)
}

// CalculateGridMetrics рассчитывает метрики Grid стратегии
func (g *GridStrategy) CalculateGridMetrics(symbol string) (map[string]interface{}, error) {
	activeOrders, err := g.storage.GetActiveGridOrders(symbol)
	if err != nil {
		return nil, err
	}

	balance, err := g.storage.GetBalance(symbol)
	if err != nil {
		return nil, err
	}

	currentPrice, err := g.exchange.GetCurrentPrice(symbol)
	if err != nil {
		return nil, err
	}

	// Рассчитываем unrealized PnL
	unrealizedPnL := 0.0
	if balance.TotalQuantity > 0 {
		currentValue := balance.TotalQuantity * currentPrice
		costBasis := balance.TotalQuantity * balance.AvgEntryPrice
		unrealizedPnL = currentValue - costBasis
	}

	totalPnL := balance.RealizedProfit + unrealizedPnL
	returnPercent := 0.0
	if balance.TotalInvested > 0 {
		returnPercent = (totalPnL / balance.TotalInvested) * 100
	}

	metrics := map[string]interface{}{
		"symbol":           symbol,
		"current_price":    currentPrice,
		"active_orders":    len(activeOrders),
		"total_quantity":   balance.TotalQuantity,
		"avg_entry_price":  balance.AvgEntryPrice,
		"total_invested":   balance.TotalInvested,
		"total_sold":       balance.TotalSold,
		"realized_profit":  balance.RealizedProfit,
		"unrealized_pnl":   unrealizedPnL,
		"total_pnl":        totalPnL,
		"return_percent":   returnPercent,
		"current_value":    balance.TotalQuantity * currentPrice,
	}

	return metrics, nil
}

// RebalanceGrid перебалансирует сетку при сильном отклонении цены
func (g *GridStrategy) RebalanceGrid(asset *storage.Asset) error {
	currentPrice, err := g.exchange.GetCurrentPrice(asset.Symbol)
	if err != nil {
		return err
	}

	activeOrders, err := g.storage.GetActiveGridOrders(asset.Symbol)
	if err != nil {
		return err
	}

	if len(activeOrders) == 0 {
		return g.InitializeGrid(asset)
	}

	// Находим минимальную и максимальную цену в сетке
	minPrice := math.MaxFloat64
	maxPrice := 0.0

	for _, order := range activeOrders {
		if order.Price < minPrice {
			minPrice = order.Price
		}
		if order.Price > maxPrice {
			maxPrice = order.Price
		}
	}

	// Если цена вышла за пределы сетки на 20%, переинициализируем
	if currentPrice < minPrice*0.8 || currentPrice > maxPrice*1.2 {
		utils.LogInfo(fmt.Sprintf("Цена %s вышла за пределы сетки (%.8f), переинициализация", asset.Symbol, currentPrice))
		return g.InitializeGrid(asset)
	}

	return nil
}
