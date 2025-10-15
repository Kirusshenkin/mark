package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
	"github.com/kirillm/dca-bot/internal/strategy"
)

// Handlers содержит все обработчики команд
type Handlers struct {
	exchange         *exchange.BybitClient
	storage          *storage.PostgresStorage
	validator        *Validator
	formatter        *Formatter
	dcaStrategy      *strategy.DCAStrategy
	autoSell         *strategy.AutoSellStrategy
	gridStrategy     *strategy.GridStrategy
	portfolioManager *strategy.PortfolioManager
	riskManager      *strategy.RiskManager
	defaultSymbol    string
	startTime        time.Time
}

// NewHandlers создает новый набор обработчиков
func NewHandlers(
	exchange *exchange.BybitClient,
	storage *storage.PostgresStorage,
	validator *Validator,
	formatter *Formatter,
	dcaStrategy *strategy.DCAStrategy,
	autoSell *strategy.AutoSellStrategy,
	gridStrategy *strategy.GridStrategy,
	portfolioManager *strategy.PortfolioManager,
	riskManager *strategy.RiskManager,
	defaultSymbol string,
) *Handlers {
	return &Handlers{
		exchange:         exchange,
		storage:          storage,
		validator:        validator,
		formatter:        formatter,
		dcaStrategy:      dcaStrategy,
		autoSell:         autoSell,
		gridStrategy:     gridStrategy,
		portfolioManager: portfolioManager,
		riskManager:      riskManager,
		defaultSymbol:    defaultSymbol,
		startTime:        time.Now(),
	}
}

// HandleStatus обрабатывает команду /status
func (h *Handlers) HandleStatus(ctx context.Context, args *CommandArgs) (string, error) {
	// Получаем активные активы
	assets, err := h.storage.GetEnabledAssets()
	if err != nil {
		return "", fmt.Errorf("failed to get active assets: %w", err)
	}

	// Собираем информацию о стратегиях
	strategies := []string{}
	if h.dcaStrategy != nil {
		strategies = append(strategies, "DCA")
	}
	if h.autoSell != nil {
		strategies = append(strategies, "Auto-Sell")
	}
	if h.gridStrategy != nil {
		strategies = append(strategies, "Grid")
	}

	// Проверяем статус Auto-Sell
	autoSellActive := false
	for _, asset := range assets {
		if asset.AutoSellEnabled {
			autoSellActive = true
			break
		}
	}

	// Проверяем активность Grid
	gridActive := false
	for _, asset := range assets {
		gridOrders, err := h.storage.GetActiveGridOrders(asset.Symbol)
		if err == nil && len(gridOrders) > 0 {
			gridActive = true
			break
		}
	}

	data := map[string]interface{}{
		"active_assets":   len(assets),
		"strategies":      strategies,
		"autosell_status": autoSellActive,
		"grid_active":     gridActive,
		"uptime":          FormatDuration(time.Since(h.startTime)),
	}

	return h.formatter.FormatStatus(data), nil
}

// HandleHistory обрабатывает команду /history
func (h *Handlers) HandleHistory(ctx context.Context, args *CommandArgs) (string, error) {
	symbol := args.Symbol
	if symbol == "" {
		symbol = h.defaultSymbol
	}

	limit := args.Count
	if limit == 0 {
		limit = 10
	}

	var trades []storage.Trade
	var err error

	if symbol == "" || symbol == "ALL" {
		trades, err = h.storage.GetAllRecentTrades(limit)
	} else {
		trades, err = h.storage.GetRecentTrades(symbol, limit)
	}

	if err != nil {
		return "", fmt.Errorf("failed to get trade history: %w", err)
	}

	return h.formatter.FormatHistory(trades, symbol, limit), nil
}

// HandleConfig обрабатывает команду /config
func (h *Handlers) HandleConfig(ctx context.Context, args *CommandArgs) (string, error) {
	// Получаем конфигурацию из активов
	assets, err := h.storage.GetEnabledAssets()
	if err != nil {
		return "", err
	}

	if len(assets) == 0 {
		return "⚙️ No active assets configured", nil
	}

	var response string
	response += "⚙️ Configuration\n\n"

	for _, asset := range assets {
		response += fmt.Sprintf("🔹 %s (%s)\n", asset.Symbol, asset.StrategyType)
		response += fmt.Sprintf("  Allocated Capital: $%.2f\n", asset.AllocatedCapital)
		response += fmt.Sprintf("  Max Position Size: $%.2f\n", asset.MaxPositionSize)

		if asset.StrategyType == "DCA" || asset.StrategyType == "HYBRID" {
			response += fmt.Sprintf("  DCA Amount: $%.2f\n", asset.DCAAmount)
			response += fmt.Sprintf("  DCA Interval: %d min\n", asset.DCAInterval)
		}

		if asset.AutoSellEnabled {
			response += fmt.Sprintf("  Auto-Sell: %.2f%% trigger, %.2f%% sell\n",
				asset.AutoSellTriggerPercent, asset.AutoSellAmountPercent)
		}

		if asset.StrategyType == "GRID" || asset.StrategyType == "HYBRID" {
			response += fmt.Sprintf("  Grid: %d levels, %.2f%% spacing, $%.2f per order\n",
				asset.GridLevels, asset.GridSpacingPercent, asset.GridOrderSize)
		}

		if asset.StopLossPercent > 0 {
			response += fmt.Sprintf("  Stop-Loss: %.2f%%\n", asset.StopLossPercent)
		}

		if asset.TakeProfitPercent > 0 {
			response += fmt.Sprintf("  Take-Profit: %.2f%%\n", asset.TakeProfitPercent)
		}

		response += "\n"
	}

	// Risk limits
	limits, err := h.storage.GetRiskLimits()
	if err == nil {
		response += "🛡️ Risk Limits:\n"
		response += fmt.Sprintf("  Max Order Size: $%.2f\n", limits.MaxOrderSizeUSD)
		response += fmt.Sprintf("  Max Position Size: $%.2f\n", limits.MaxPositionSizeUSD)
		response += fmt.Sprintf("  Max Total Exposure: $%.2f\n", limits.MaxTotalExposure)
		response += fmt.Sprintf("  Max Daily Loss: $%.2f\n", limits.MaxDailyLoss)
	}

	return response, nil
}

// HandlePrice обрабатывает команду /price
func (h *Handlers) HandlePrice(ctx context.Context, args *CommandArgs) (string, error) {
	symbol := args.Symbol
	if symbol == "" {
		symbol = h.defaultSymbol
	}

	// Валидируем символ
	if err := h.validator.ValidateSymbol(symbol); err != nil {
		return "", err
	}

	price, err := h.exchange.GetCurrentPrice(symbol)
	if err != nil {
		return "", fmt.Errorf("failed to get price for %s: %w", symbol, err)
	}

	// Получаем баланс если есть позиция
	balance, err := h.storage.GetBalance(symbol)
	hasPosition := err == nil && balance != nil && balance.TotalQuantity > 0

	avgEntry := 0.0
	if hasPosition {
		avgEntry = balance.AvgEntryPrice
	}

	return h.formatter.FormatPrice(symbol, price, avgEntry, hasPosition), nil
}

// HandlePortfolio обрабатывает команду /portfolio
func (h *Handlers) HandlePortfolio(ctx context.Context, args *CommandArgs) (string, error) {
	if h.portfolioManager == nil {
		return "Portfolio manager not available", nil
	}

	summary, err := h.portfolioManager.GetPortfolioSummary()
	if err != nil {
		return "", fmt.Errorf("failed to get portfolio: %w", err)
	}

	return h.formatter.FormatPortfolio(summary), nil
}

// HandleBuy обрабатывает команду /buy
func (h *Handlers) HandleBuy(ctx context.Context, args *CommandArgs) (string, error) {
	symbol := args.Symbol
	if symbol == "" {
		symbol = h.defaultSymbol
	}

	amount := args.Amount
	if amount == 0 {
		// Используем дефолтную сумму из конфига актива
		asset, err := h.storage.GetAsset(symbol)
		if err != nil || asset == nil {
			return "", fmt.Errorf("asset %s not configured", symbol)
		}
		amount = asset.DCAAmount
	}

	// Валидация
	if err := h.validator.ValidateBuy(symbol, amount); err != nil {
		return "", err
	}

	// Выполняем покупку
	currentPrice, err := h.exchange.GetCurrentPrice(symbol)
	if err != nil {
		return "", fmt.Errorf("failed to get price: %w", err)
	}

	quantity := amount / currentPrice

	orderInfo, err := h.exchange.PlaceOrder(symbol, "BUY", quantity)
	if err != nil {
		return "", fmt.Errorf("failed to place order: %w", err)
	}

	// Сохраняем сделку
	trade := &storage.Trade{
		Symbol:       symbol,
		Side:         "BUY",
		Quantity:     quantity,
		Price:        currentPrice,
		Amount:       amount,
		OrderID:      orderInfo.OrderID,
		Status:       "FILLED",
		StrategyType: "MANUAL",
		CreatedAt:    time.Now(),
	}

	if err := h.storage.SaveTrade(trade); err != nil {
		return "", fmt.Errorf("failed to save trade: %w", err)
	}

	// Обновляем баланс
	balance, _ := h.storage.GetBalance(symbol)
	if balance == nil {
		balance = &storage.Balance{Symbol: symbol}
	}

	newTotalQty := balance.TotalQuantity + quantity
	newInvested := balance.TotalInvested + amount
	balance.AvgEntryPrice = newInvested / newTotalQty
	balance.TotalQuantity = newTotalQty
	balance.AvailableQty = newTotalQty
	balance.TotalInvested = newInvested

	if err := h.storage.UpdateBalance(balance); err != nil {
		return "", fmt.Errorf("failed to update balance: %w", err)
	}

	return h.formatter.FormatSuccess(fmt.Sprintf("Bought %.8f %s at $%.2f (total: $%.2f)",
		quantity, symbol, currentPrice, amount)), nil
}

// HandleSell обрабатывает команду /sell
func (h *Handlers) HandleSell(ctx context.Context, args *CommandArgs) (string, error) {
	symbol := args.Symbol
	if symbol == "" {
		symbol = h.defaultSymbol
	}

	percent := args.Percent
	if percent <= 0 || percent > 100 {
		return "", fmt.Errorf("percent must be between 1 and 100")
	}

	// Валидация
	if err := h.validator.ValidateSell(symbol, percent); err != nil {
		return "", err
	}

	// Получаем баланс
	balance, err := h.storage.GetBalance(symbol)
	if err != nil {
		return "", err
	}

	// Вычисляем количество для продажи
	sellQuantity := balance.AvailableQty * (percent / 100.0)

	currentPrice, err := h.exchange.GetCurrentPrice(symbol)
	if err != nil {
		return "", fmt.Errorf("failed to get price: %w", err)
	}

	// Размещаем ордер
	orderInfo, err := h.exchange.PlaceOrder(symbol, "SELL", sellQuantity)
	if err != nil {
		return "", fmt.Errorf("failed to place order: %w", err)
	}

	sellAmount := sellQuantity * currentPrice

	// Сохраняем сделку
	trade := &storage.Trade{
		Symbol:       symbol,
		Side:         "SELL",
		Quantity:     sellQuantity,
		Price:        currentPrice,
		Amount:       sellAmount,
		OrderID:      orderInfo.OrderID,
		Status:       "FILLED",
		StrategyType: "MANUAL",
		CreatedAt:    time.Now(),
	}

	if err := h.storage.SaveTrade(trade); err != nil {
		return "", fmt.Errorf("failed to save trade: %w", err)
	}

	// Обновляем баланс
	costBasis := sellQuantity * balance.AvgEntryPrice
	profit := sellAmount - costBasis

	balance.TotalQuantity -= sellQuantity
	balance.AvailableQty -= sellQuantity
	balance.TotalSold += sellAmount
	balance.RealizedProfit += profit

	if err := h.storage.UpdateBalance(balance); err != nil {
		return "", fmt.Errorf("failed to update balance: %w", err)
	}

	return h.formatter.FormatSuccess(fmt.Sprintf("Sold %.8f %s (%.0f%%) at $%.2f\nProfit: $%.2f",
		sellQuantity, symbol, percent, currentPrice, profit)), nil
}

// HandleAutoSellOn обрабатывает команду /autosellon
func (h *Handlers) HandleAutoSellOn(ctx context.Context, args *CommandArgs) (string, error) {
	symbol := args.Symbol
	if symbol == "" {
		symbol = h.defaultSymbol
	}

	asset, err := h.storage.GetAsset(symbol)
	if err != nil || asset == nil {
		return "", fmt.Errorf("asset %s not configured", symbol)
	}

	asset.AutoSellEnabled = true
	if err := h.storage.CreateOrUpdateAsset(asset); err != nil {
		return "", err
	}

	return h.formatter.FormatSuccess(fmt.Sprintf("Auto-Sell enabled for %s", symbol)), nil
}

// HandleAutoSellOff обрабатывает команду /autoselloff
func (h *Handlers) HandleAutoSellOff(ctx context.Context, args *CommandArgs) (string, error) {
	symbol := args.Symbol
	if symbol == "" {
		symbol = h.defaultSymbol
	}

	asset, err := h.storage.GetAsset(symbol)
	if err != nil || asset == nil {
		return "", fmt.Errorf("asset %s not configured", symbol)
	}

	asset.AutoSellEnabled = false
	if err := h.storage.CreateOrUpdateAsset(asset); err != nil {
		return "", err
	}

	return h.formatter.FormatSuccess(fmt.Sprintf("Auto-Sell disabled for %s", symbol)), nil
}

// HandleAutoSell обрабатывает команду /autosell
func (h *Handlers) HandleAutoSell(ctx context.Context, args *CommandArgs) (string, error) {
	symbol := args.Symbol
	if symbol == "" {
		symbol = h.defaultSymbol
	}

	trigger := args.Trigger
	sellPercent := args.Percent

	// Валидация
	if err := h.validator.ValidateAutoSell(symbol, trigger, sellPercent); err != nil {
		return "", err
	}

	asset, err := h.storage.GetAsset(symbol)
	if err != nil || asset == nil {
		return "", fmt.Errorf("asset %s not configured", symbol)
	}

	asset.AutoSellTriggerPercent = trigger
	asset.AutoSellAmountPercent = sellPercent
	asset.AutoSellEnabled = true

	if err := h.storage.CreateOrUpdateAsset(asset); err != nil {
		return "", err
	}

	return h.formatter.FormatSuccess(fmt.Sprintf("Auto-Sell configured for %s: trigger %.2f%%, sell %.2f%%",
		symbol, trigger, sellPercent)), nil
}

// HandleGridInit обрабатывает команду /gridinit
func (h *Handlers) HandleGridInit(ctx context.Context, args *CommandArgs) (string, error) {
	symbol := args.Symbol
	levels := args.Levels
	spacing := args.Spacing
	orderSize := args.Amount

	// Валидация
	if err := h.validator.ValidateGridInit(symbol, levels, spacing, orderSize); err != nil {
		return "", err
	}

	// Получаем или создаем актив
	asset, err := h.storage.GetAsset(symbol)
	if err != nil || asset == nil {
		asset = &storage.Asset{
			Symbol:             symbol,
			Enabled:            true,
			StrategyType:       "GRID",
			AllocatedCapital:   float64(levels) * orderSize * 2,
			MaxPositionSize:    float64(levels) * orderSize * 3,
			GridLevels:         levels,
			GridSpacingPercent: spacing,
			GridOrderSize:      orderSize,
		}
		if err := h.storage.CreateOrUpdateAsset(asset); err != nil {
			return "", err
		}
	} else {
		asset.GridLevels = levels
		asset.GridSpacingPercent = spacing
		asset.GridOrderSize = orderSize
		if err := h.storage.CreateOrUpdateAsset(asset); err != nil {
			return "", err
		}
	}

	// Инициализируем Grid
	if err := h.gridStrategy.InitializeGrid(asset); err != nil {
		return "", fmt.Errorf("failed to initialize grid: %w", err)
	}

	return h.formatter.FormatSuccess(fmt.Sprintf("Grid initialized for %s: %d levels, %.2f%% spacing, $%.2f per order",
		symbol, levels, spacing, orderSize)), nil
}

// HandleGridStatus обрабатывает команду /gridstatus
func (h *Handlers) HandleGridStatus(ctx context.Context, args *CommandArgs) (string, error) {
	symbol := args.Symbol
	if symbol == "" {
		return "", fmt.Errorf("symbol required")
	}

	metrics, err := h.gridStrategy.CalculateGridMetrics(symbol)
	if err != nil {
		return "", fmt.Errorf("failed to get grid status: %w", err)
	}

	return h.formatter.FormatGridStatus(symbol, metrics), nil
}

// HandleGridStop обрабатывает команду /gridstop
func (h *Handlers) HandleGridStop(ctx context.Context, args *CommandArgs) (string, error) {
	symbol := args.Symbol
	if symbol == "" {
		return "", fmt.Errorf("symbol required")
	}

	// Отменяем все Grid ордера
	if err := h.storage.CancelGridOrders(symbol); err != nil {
		return "", fmt.Errorf("failed to cancel grid orders: %w", err)
	}

	return h.formatter.FormatSuccess(fmt.Sprintf("Grid stopped for %s", symbol)), nil
}

// HandleRisk обрабатывает команду /risk
func (h *Handlers) HandleRisk(ctx context.Context, args *CommandArgs) (string, error) {
	if h.riskManager == nil {
		return "Risk manager not available", nil
	}

	status, err := h.riskManager.GetRiskStatus()
	if err != nil {
		return "", fmt.Errorf("failed to get risk status: %w", err)
	}

	limits, err := h.storage.GetRiskLimits()
	if err != nil {
		return "", err
	}

	currentExposure, _ := status["total_exposure"].(float64)
	dailyLoss, _ := status["daily_loss"].(float64)

	return h.formatter.FormatRiskStatus(limits, currentExposure, dailyLoss), nil
}

// HandlePanicStop обрабатывает команду /panicstop
func (h *Handlers) HandlePanicStop(ctx context.Context, args *CommandArgs) (string, error) {
	if h.riskManager == nil {
		return "Risk manager not available", nil
	}

	action := normalizeAction(args.Action)

	if action == "status" || action == "" {
		limits, err := h.storage.GetRiskLimits()
		if err != nil {
			return "", err
		}

		status := "disabled"
		if limits.EnableEmergencyStop {
			status = "🚨 ENABLED"
		}

		return fmt.Sprintf("Emergency Stop: %s", status), nil
	}

	enabled := action == "on"

	if err := h.riskManager.SetEmergencyStop(enabled); err != nil {
		return "", err
	}

	if enabled {
		return "🚨 EMERGENCY STOP ACTIVATED\n\nAll trading is now paused.", nil
	}

	return h.formatter.FormatSuccess("Emergency stop deactivated"), nil
}

// HandleHelp обрабатывает команду /help
func (h *Handlers) HandleHelp(ctx context.Context, args *CommandArgs) (string, error) {
	help := `🤖 Crypto Trading Bot Commands

📊 INFORMATION:
/status - Current status and active strategies
/history [SYMBOL] [N] - Recent trades (default: 10)
/config - Current configuration
/price <SYMBOL> - Current market price
/portfolio - Portfolio overview with P&L

💰 TRADING:
/buy [SYMBOL] [AMOUNT] - Execute buy order
  Example: /buy BTCUSDT 20
/sell <PERCENT> [SYMBOL] - Sell % of position
  Example: /sell 50 BTCUSDT

⚙️ AUTO-SELL:
/autosellon [SYMBOL] - Enable Auto-Sell
/autoselloff [SYMBOL] - Disable Auto-Sell
/autosell [SYMBOL] <TRIGGER%> <SELL%> - Configure
  Example: /autosell BTCUSDT 15 50

🔷 GRID TRADING:
/gridinit <SYMBOL> <LEVELS> <SPACING%> <SIZE> - Init Grid
  Example: /gridinit ETHUSDT 10 2.5 100
/gridstatus <SYMBOL> - Grid status
/gridstop <SYMBOL> - Stop Grid (Admin only)

🛡️ RISK & ADMIN:
/risk - Risk limits and exposure
/panicstop [on|off] - Emergency stop (Admin only)

🧠 AI NATURAL LANGUAGE:
Just send a message:
• "Buy 20 USDT of BTC"
• "Sell 50% of ETH"
• "Set auto-sell at +15%"
• "Show portfolio"
• "Купи BTC на 20 USDT"
• "Продай 30% позиции"

Supports English and Russian! 🇬🇧🇷🇺`

	return help, nil
}
