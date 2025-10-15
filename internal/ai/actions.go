package ai

import (
	"encoding/json"
	"fmt"

	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
	"github.com/kirillm/dca-bot/internal/strategy"
	"github.com/kirillm/dca-bot/pkg/utils"
)

// ActionExecutor выполняет действия, запрошенные AI
type ActionExecutor struct {
	storage          *storage.PostgresStorage
	exchange         *exchange.BybitClient
	portfolioManager *strategy.PortfolioManager
	riskManager      *strategy.RiskManager
	gridStrategy     *strategy.GridStrategy
}

func NewActionExecutor(
	storage *storage.PostgresStorage,
	exchange *exchange.BybitClient,
	portfolioManager *strategy.PortfolioManager,
	riskManager *strategy.RiskManager,
	gridStrategy *strategy.GridStrategy,
) *ActionExecutor {
	return &ActionExecutor{
		storage:          storage,
		exchange:         exchange,
		portfolioManager: portfolioManager,
		riskManager:      riskManager,
		gridStrategy:     gridStrategy,
	}
}

// ExecuteAction выполняет действие AI
func (e *ActionExecutor) ExecuteAction(action AIAction) (string, error) {
	utils.LogInfo(fmt.Sprintf("Выполнение AI действия: %s", action.Type))

	switch action.Type {
	// ===== MULTI-ASSET УПРАВЛЕНИЕ =====
	case "add_asset":
		return e.addAsset(action.Parameters)
	case "remove_asset":
		return e.removeAsset(action.Parameters)
	case "update_asset":
		return e.updateAsset(action.Parameters)
	case "enable_asset":
		return e.enableAsset(action.Parameters)
	case "disable_asset":
		return e.disableAsset(action.Parameters)
	case "list_assets":
		return e.listAssets()

	// ===== GRID СТРАТЕГИЯ =====
	case "init_grid":
		return e.initGrid(action.Parameters)
	case "stop_grid":
		return e.stopGrid(action.Parameters)
	case "grid_status":
		return e.gridStatus(action.Parameters)

	// ===== ПОРТФЕЛЬ =====
	case "portfolio_summary":
		return e.portfolioSummary()
	case "asset_allocation":
		return e.assetAllocation()
	case "allocate_capital":
		return e.allocateCapital(action.Parameters)
	case "rebalance_portfolio":
		return e.rebalancePortfolio()

	// ===== РИСК МЕНЕДЖМЕНТ =====
	case "set_stop_loss":
		return e.setStopLoss(action.Parameters)
	case "set_take_profit":
		return e.setTakeProfit(action.Parameters)
	case "update_risk_limits":
		return e.updateRiskLimits(action.Parameters)
	case "risk_status":
		return e.riskStatus()
	case "emergency_stop":
		return e.emergencyStop(action.Parameters)

	// ===== АНАЛИТИКА =====
	case "performance_metrics":
		return e.performanceMetrics(action.Parameters)
	case "pnl_history":
		return e.pnlHistory(action.Parameters)

	// ===== LEGACY (для обратной совместимости) =====
	case "get_status":
		return e.getStatus(action.Parameters)
	case "update_dca_amount":
		return e.updateDCAAmount(action.Parameters)
	case "update_autosell_trigger":
		return e.updateAutoSellTrigger(action.Parameters)
	case "enable_autosell":
		return e.enableAutoSell(action.Parameters)
	case "disable_autosell":
		return e.disableAutoSell(action.Parameters)
	case "get_history":
		return e.getHistory(action.Parameters)
	case "get_price":
		return e.getPrice(action.Parameters)

	default:
		return "", fmt.Errorf("неизвестное действие: %s", action.Type)
	}
}

// ===== MULTI-ASSET УПРАВЛЕНИЕ =====

func (e *ActionExecutor) addAsset(params map[string]interface{}) (string, error) {
	symbol, ok := params["symbol"].(string)
	if !ok {
		return "", fmt.Errorf("параметр symbol обязателен")
	}

	strategyType, _ := params["strategy_type"].(string)
	if strategyType == "" {
		strategyType = "DCA"
	}

	asset := &storage.Asset{
		Symbol:                symbol,
		Enabled:               true,
		StrategyType:          strategyType,
		AllocatedCapital:      getFloatParam(params, "allocated_capital", 1000),
		MaxPositionSize:       getFloatParam(params, "max_position_size", 5000),
		DCAAmount:             getFloatParam(params, "dca_amount", 10),
		DCAInterval:           int(getFloatParam(params, "dca_interval_hours", 24) * 60),
		AutoSellEnabled:       getBoolParam(params, "auto_sell_enabled", false),
		AutoSellTriggerPercent: getFloatParam(params, "auto_sell_trigger_percent", 10),
		AutoSellAmountPercent:  getFloatParam(params, "auto_sell_amount_percent", 50),
		GridLevels:            int(getFloatParam(params, "grid_levels", 0)),
		GridSpacingPercent:    getFloatParam(params, "grid_spacing_percent", 0),
		GridOrderSize:         getFloatParam(params, "grid_order_size", 0),
		StopLossPercent:       getFloatParam(params, "stop_loss_percent", 0),
		TakeProfitPercent:     getFloatParam(params, "take_profit_percent", 0),
	}

	if err := e.storage.CreateOrUpdateAsset(asset); err != nil {
		return "", fmt.Errorf("не удалось добавить актив: %w", err)
	}

	// Если Grid стратегия, инициализируем сетку
	if strategyType == "GRID" && asset.GridLevels > 0 {
		if err := e.gridStrategy.InitializeGrid(asset); err != nil {
			utils.LogError(fmt.Sprintf("Не удалось инициализировать Grid для %s: %v", symbol, err))
		}
	}

	return fmt.Sprintf("Актив %s успешно добавлен со стратегией %s", symbol, strategyType), nil
}

func (e *ActionExecutor) removeAsset(params map[string]interface{}) (string, error) {
	symbol, ok := params["symbol"].(string)
	if !ok {
		return "", fmt.Errorf("параметр symbol обязателен")
	}

	if err := e.storage.DisableAsset(symbol); err != nil {
		return "", fmt.Errorf("не удалось удалить актив: %w", err)
	}

	// Отменяем все Grid ордера
	if err := e.storage.CancelGridOrders(symbol); err != nil {
		utils.LogError(fmt.Sprintf("Не удалось отменить Grid ордера для %s: %v", symbol, err))
	}

	return fmt.Sprintf("Актив %s деактивирован", symbol), nil
}

func (e *ActionExecutor) updateAsset(params map[string]interface{}) (string, error) {
	symbol, ok := params["symbol"].(string)
	if !ok {
		return "", fmt.Errorf("параметр symbol обязателен")
	}

	asset, err := e.storage.GetAsset(symbol)
	if err != nil || asset == nil {
		return "", fmt.Errorf("актив %s не найден", symbol)
	}

	// Обновляем параметры, если они предоставлены
	if val, ok := params["dca_amount"]; ok {
		asset.DCAAmount = val.(float64)
	}
	if val, ok := params["auto_sell_trigger_percent"]; ok {
		asset.AutoSellTriggerPercent = val.(float64)
	}
	if val, ok := params["auto_sell_amount_percent"]; ok {
		asset.AutoSellAmountPercent = val.(float64)
	}
	if val, ok := params["auto_sell_enabled"]; ok {
		asset.AutoSellEnabled = val.(bool)
	}
	if val, ok := params["stop_loss_percent"]; ok {
		asset.StopLossPercent = val.(float64)
	}
	if val, ok := params["take_profit_percent"]; ok {
		asset.TakeProfitPercent = val.(float64)
	}
	if val, ok := params["grid_spacing_percent"]; ok {
		asset.GridSpacingPercent = val.(float64)
	}

	if err := e.storage.CreateOrUpdateAsset(asset); err != nil {
		return "", fmt.Errorf("не удалось обновить актив: %w", err)
	}

	return fmt.Sprintf("Актив %s успешно обновлен", symbol), nil
}

func (e *ActionExecutor) enableAsset(params map[string]interface{}) (string, error) {
	symbol, ok := params["symbol"].(string)
	if !ok {
		return "", fmt.Errorf("параметр symbol обязателен")
	}

	if err := e.storage.EnableAsset(symbol); err != nil {
		return "", err
	}

	return fmt.Sprintf("Актив %s активирован", symbol), nil
}

func (e *ActionExecutor) disableAsset(params map[string]interface{}) (string, error) {
	symbol, ok := params["symbol"].(string)
	if !ok {
		return "", fmt.Errorf("параметр symbol обязателен")
	}

	if err := e.storage.DisableAsset(symbol); err != nil {
		return "", err
	}

	return fmt.Sprintf("Актив %s деактивирован", symbol), nil
}

func (e *ActionExecutor) listAssets() (string, error) {
	assets, err := e.storage.GetAllAssets()
	if err != nil {
		return "", err
	}

	if len(assets) == 0 {
		return "Нет активов в системе", nil
	}

	result := "Список активов:\n\n"
	for _, asset := range assets {
		status := "🟢"
		if !asset.Enabled {
			status = "🔴"
		}
		result += fmt.Sprintf("%s %s (%s)\n", status, asset.Symbol, asset.StrategyType)
		result += fmt.Sprintf("  Капитал: $%.2f\n", asset.AllocatedCapital)
		if asset.StrategyType == "DCA" || asset.StrategyType == "HYBRID" {
			result += fmt.Sprintf("  DCA: $%.2f каждые %d мин\n", asset.DCAAmount, asset.DCAInterval)
		}
		if asset.StrategyType == "GRID" || asset.StrategyType == "HYBRID" {
			result += fmt.Sprintf("  Grid: %d уровней, %.2f%% spacing\n", asset.GridLevels, asset.GridSpacingPercent)
		}
		if asset.StopLossPercent > 0 {
			result += fmt.Sprintf("  Stop-Loss: %.2f%%\n", asset.StopLossPercent)
		}
		if asset.TakeProfitPercent > 0 {
			result += fmt.Sprintf("  Take-Profit: %.2f%%\n", asset.TakeProfitPercent)
		}
		result += "\n"
	}

	return result, nil
}

// ===== GRID СТРАТЕГИЯ =====

func (e *ActionExecutor) initGrid(params map[string]interface{}) (string, error) {
	symbol, ok := params["symbol"].(string)
	if !ok {
		return "", fmt.Errorf("параметр symbol обязателен")
	}

	asset, err := e.storage.GetAsset(symbol)
	if err != nil || asset == nil {
		return "", fmt.Errorf("актив %s не найден", symbol)
	}

	if err := e.gridStrategy.InitializeGrid(asset); err != nil {
		return "", fmt.Errorf("не удалось инициализировать Grid: %w", err)
	}

	return fmt.Sprintf("Grid стратегия инициализирована для %s с %d уровнями", symbol, asset.GridLevels), nil
}

func (e *ActionExecutor) stopGrid(params map[string]interface{}) (string, error) {
	symbol, ok := params["symbol"].(string)
	if !ok {
		return "", fmt.Errorf("параметр symbol обязателен")
	}

	if err := e.storage.CancelGridOrders(symbol); err != nil {
		return "", err
	}

	return fmt.Sprintf("Grid стратегия остановлена для %s", symbol), nil
}

func (e *ActionExecutor) gridStatus(params map[string]interface{}) (string, error) {
	symbol, ok := params["symbol"].(string)
	if !ok {
		return "", fmt.Errorf("параметр symbol обязателен")
	}

	metrics, err := e.gridStrategy.CalculateGridMetrics(symbol)
	if err != nil {
		return "", err
	}

	return formatGridMetrics(metrics), nil
}

// ===== ПОРТФЕЛЬ =====

func (e *ActionExecutor) portfolioSummary() (string, error) {
	summary, err := e.portfolioManager.GetPortfolioSummary()
	if err != nil {
		return "", err
	}

	return formatPortfolioSummary(summary), nil
}

func (e *ActionExecutor) assetAllocation() (string, error) {
	allocations, err := e.portfolioManager.GetAssetAllocation()
	if err != nil {
		return "", err
	}

	result := "Распределение активов:\n\n"
	for _, alloc := range allocations {
		result += fmt.Sprintf("%s: %.2f%% ($%.2f)\n",
			alloc["symbol"], alloc["percentage"], alloc["current_value"])
	}

	return result, nil
}

func (e *ActionExecutor) allocateCapital(params map[string]interface{}) (string, error) {
	totalCapital, ok := params["total_capital"].(float64)
	if !ok {
		return "", fmt.Errorf("параметр total_capital обязателен")
	}

	if err := e.portfolioManager.AllocateCapital(totalCapital); err != nil {
		return "", err
	}

	return fmt.Sprintf("Капитал $%.2f распределен между активами", totalCapital), nil
}

func (e *ActionExecutor) rebalancePortfolio() (string, error) {
	if err := e.portfolioManager.RebalancePortfolio(); err != nil {
		return "", err
	}

	return "Портфель ребалансирован", nil
}

// ===== РИСК МЕНЕДЖМЕНТ =====

func (e *ActionExecutor) setStopLoss(params map[string]interface{}) (string, error) {
	symbol, _ := params["symbol"].(string)
	percent, ok := params["percent"].(float64)
	if !ok {
		return "", fmt.Errorf("параметр percent обязателен")
	}

	if symbol == "" {
		// Устанавливаем для всех активов
		assets, err := e.storage.GetEnabledAssets()
		if err != nil {
			return "", err
		}
		for _, asset := range assets {
			asset.StopLossPercent = percent
			e.storage.CreateOrUpdateAsset(&asset)
		}
		return fmt.Sprintf("Stop-Loss %.2f%% установлен для всех активов", percent), nil
	} else {
		asset, err := e.storage.GetAsset(symbol)
		if err != nil || asset == nil {
			return "", fmt.Errorf("актив не найден")
		}
		asset.StopLossPercent = percent
		if err := e.storage.CreateOrUpdateAsset(asset); err != nil {
			return "", err
		}
		return fmt.Sprintf("Stop-Loss %.2f%% установлен для %s", percent, symbol), nil
	}
}

func (e *ActionExecutor) setTakeProfit(params map[string]interface{}) (string, error) {
	symbol, _ := params["symbol"].(string)
	percent, ok := params["percent"].(float64)
	if !ok {
		return "", fmt.Errorf("параметр percent обязателен")
	}

	if symbol == "" {
		assets, err := e.storage.GetEnabledAssets()
		if err != nil {
			return "", err
		}
		for _, asset := range assets {
			asset.TakeProfitPercent = percent
			e.storage.CreateOrUpdateAsset(&asset)
		}
		return fmt.Sprintf("Take-Profit %.2f%% установлен для всех активов", percent), nil
	} else {
		asset, err := e.storage.GetAsset(symbol)
		if err != nil || asset == nil {
			return "", fmt.Errorf("актив не найден")
		}
		asset.TakeProfitPercent = percent
		if err := e.storage.CreateOrUpdateAsset(asset); err != nil {
			return "", err
		}
		return fmt.Sprintf("Take-Profit %.2f%% установлен для %s", percent, symbol), nil
	}
}

func (e *ActionExecutor) updateRiskLimits(params map[string]interface{}) (string, error) {
	limits, err := e.storage.GetRiskLimits()
	if err != nil {
		return "", err
	}

	if val, ok := params["max_daily_loss"]; ok {
		limits.MaxDailyLoss = val.(float64)
	}
	if val, ok := params["max_total_exposure"]; ok {
		limits.MaxTotalExposure = val.(float64)
	}
	if val, ok := params["max_position_size_usd"]; ok {
		limits.MaxPositionSizeUSD = val.(float64)
	}
	if val, ok := params["max_order_size_usd"]; ok {
		limits.MaxOrderSizeUSD = val.(float64)
	}

	if err := e.storage.UpdateRiskLimits(limits); err != nil {
		return "", err
	}

	return "Риск-лимиты обновлены", nil
}

func (e *ActionExecutor) riskStatus() (string, error) {
	status, err := e.riskManager.GetRiskStatus()
	if err != nil {
		return "", err
	}

	return formatRiskStatus(status), nil
}

func (e *ActionExecutor) emergencyStop(params map[string]interface{}) (string, error) {
	enabled, ok := params["enabled"].(bool)
	if !ok {
		enabled = true
	}

	if err := e.riskManager.SetEmergencyStop(enabled); err != nil {
		return "", err
	}

	if enabled {
		return "🚨 ЭКСТРЕННАЯ ОСТАНОВКА АКТИВИРОВАНА", nil
	}
	return "Экстренная остановка деактивирована", nil
}

// ===== АНАЛИТИКА =====

func (e *ActionExecutor) performanceMetrics(params map[string]interface{}) (string, error) {
	symbol, ok := params["symbol"].(string)
	if !ok {
		return "", fmt.Errorf("параметр symbol обязателен")
	}

	days := int(getFloatParam(params, "days", 30))

	metrics, err := e.portfolioManager.GetPerformanceMetrics(symbol, days)
	if err != nil {
		return "", err
	}

	return formatPerformanceMetrics(metrics), nil
}

func (e *ActionExecutor) pnlHistory(params map[string]interface{}) (string, error) {
	symbol, ok := params["symbol"].(string)
	if !ok {
		return "", fmt.Errorf("параметр symbol обязателен")
	}

	limit := int(getFloatParam(params, "limit", 7))

	history, err := e.storage.GetPnLHistory(symbol, "DAILY", limit)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("История PnL для %s (последние %d дней):\n\n", symbol, limit)
	for _, pnl := range history {
		result += fmt.Sprintf("%s: PnL $%.2f (%.2f%%), Current Value $%.2f\n",
			pnl.CreatedAt.Format("2006-01-02"),
			pnl.TotalPnL,
			pnl.ReturnPercent,
			pnl.CurrentValue,
		)
	}

	return result, nil
}

// ===== LEGACY МЕТОДЫ =====

func (e *ActionExecutor) getStatus(params map[string]interface{}) (string, error) {
	return e.portfolioSummary()
}

func (e *ActionExecutor) updateDCAAmount(params map[string]interface{}) (string, error) {
	amount, ok := params["amount"].(float64)
	if !ok {
		return "", fmt.Errorf("параметр amount обязателен")
	}

	// Обновляем для дефолтного актива или всех DCA активов
	assets, err := e.storage.GetEnabledAssets()
	if err != nil {
		return "", err
	}

	updated := 0
	for _, asset := range assets {
		if asset.StrategyType == "DCA" || asset.StrategyType == "HYBRID" {
			asset.DCAAmount = amount
			e.storage.CreateOrUpdateAsset(&asset)
			updated++
		}
	}

	return fmt.Sprintf("DCA сумма обновлена до $%.2f для %d активов", amount, updated), nil
}

func (e *ActionExecutor) updateAutoSellTrigger(params map[string]interface{}) (string, error) {
	percent, ok := params["percent"].(float64)
	if !ok {
		return "", fmt.Errorf("параметр percent обязателен")
	}

	assets, err := e.storage.GetEnabledAssets()
	if err != nil {
		return "", err
	}

	for _, asset := range assets {
		asset.AutoSellTriggerPercent = percent
		e.storage.CreateOrUpdateAsset(&asset)
	}

	return fmt.Sprintf("Триггер Auto-Sell установлен на %.2f%%", percent), nil
}

func (e *ActionExecutor) enableAutoSell(params map[string]interface{}) (string, error) {
	assets, err := e.storage.GetEnabledAssets()
	if err != nil {
		return "", err
	}

	for _, asset := range assets {
		asset.AutoSellEnabled = true
		e.storage.CreateOrUpdateAsset(&asset)
	}

	return "Auto-Sell включен для всех активов", nil
}

func (e *ActionExecutor) disableAutoSell(params map[string]interface{}) (string, error) {
	assets, err := e.storage.GetEnabledAssets()
	if err != nil {
		return "", err
	}

	for _, asset := range assets {
		asset.AutoSellEnabled = false
		e.storage.CreateOrUpdateAsset(&asset)
	}

	return "Auto-Sell выключен для всех активов", nil
}

func (e *ActionExecutor) getHistory(params map[string]interface{}) (string, error) {
	limit := int(getFloatParam(params, "limit", 10))

	trades, err := e.storage.GetAllRecentTrades(limit)
	if err != nil {
		return "", err
	}

	result := "Последние сделки:\n\n"
	for _, trade := range trades {
		result += fmt.Sprintf("%s %s: %.8f @ $%.2f (%s)\n",
			trade.Symbol, trade.Side, trade.Quantity, trade.Price, trade.StrategyType)
	}

	return result, nil
}

func (e *ActionExecutor) getPrice(params map[string]interface{}) (string, error) {
	symbol, ok := params["symbol"].(string)
	if !ok {
		// Получаем цены всех активов
		assets, err := e.storage.GetEnabledAssets()
		if err != nil {
			return "", err
		}

		result := "Текущие цены:\n\n"
		for _, asset := range assets {
			price, err := e.exchange.GetCurrentPrice(asset.Symbol)
			if err != nil {
				continue
			}
			result += fmt.Sprintf("%s: $%.2f\n", asset.Symbol, price)
		}
		return result, nil
	} else {
		price, err := e.exchange.GetCurrentPrice(symbol)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Текущая цена %s: $%.2f", symbol, price), nil
	}
}

// ===== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ =====

func getFloatParam(params map[string]interface{}, key string, defaultVal float64) float64 {
	if val, ok := params[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return defaultVal
}

func getBoolParam(params map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := params[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultVal
}

func formatGridMetrics(metrics map[string]interface{}) string {
	data, _ := json.MarshalIndent(metrics, "", "  ")
	return fmt.Sprintf("Grid метрики:\n%s", string(data))
}

func formatPortfolioSummary(summary map[string]interface{}) string {
	result := "📊 Сводка по портфелю:\n\n"
	result += fmt.Sprintf("Всего активов: %v\n", summary["total_assets"])
	result += fmt.Sprintf("Активных позиций: %v\n", summary["active_positions"])
	result += fmt.Sprintf("Инвестировано: $%.2f\n", summary["total_invested"])
	result += fmt.Sprintf("Текущая стоимость: $%.2f\n", summary["total_current_value"])
	result += fmt.Sprintf("Реализованная прибыль: $%.2f\n", summary["total_realized_profit"])
	result += fmt.Sprintf("Нереализованный PnL: $%.2f\n", summary["total_unrealized_pnl"])
	result += fmt.Sprintf("Общий PnL: $%.2f (%.2f%%)\n", summary["total_pnl"], summary["total_return_percent"])

	if assets, ok := summary["assets"].([]map[string]interface{}); ok && len(assets) > 0 {
		result += "\nАктивы:\n"
		for _, asset := range assets {
			result += fmt.Sprintf("\n%s:\n", asset["symbol"])
			result += fmt.Sprintf("  Количество: %.8f\n", asset["quantity"])
			result += fmt.Sprintf("  Средняя цена: $%.2f\n", asset["avg_entry_price"])
			result += fmt.Sprintf("  Текущая цена: $%.2f\n", asset["current_price"])
			result += fmt.Sprintf("  PnL: $%.2f (%.2f%%)\n", asset["total_pnl"], asset["return_percent"])
		}
	}

	return result
}

func formatRiskStatus(status map[string]interface{}) string {
	result := "🛡️ Риск статус:\n\n"
	result += fmt.Sprintf("Emergency Stop: %v\n", status["emergency_stop_enabled"])
	result += fmt.Sprintf("Общая экспозиция: $%.2f / $%.2f (%.1f%%)\n",
		status["total_exposure"], status["max_total_exposure"], status["exposure_percent"])
	result += fmt.Sprintf("Дневной убыток: $%.2f / $%.2f (%.1f%%)\n",
		status["daily_loss"], status["max_daily_loss"], status["daily_loss_percent"])
	result += fmt.Sprintf("Лимит позиции: $%.2f\n", status["max_position_size_usd"])
	result += fmt.Sprintf("Лимит ордера: $%.2f\n", status["max_order_size_usd"])
	return result
}

func formatPerformanceMetrics(metrics map[string]interface{}) string {
	result := fmt.Sprintf("📈 Метрики производительности для %s:\n\n", metrics["symbol"])
	result += fmt.Sprintf("Период: %v дней\n", metrics["period_days"])
	result += fmt.Sprintf("Точек данных: %v\n", metrics["data_points"])
	result += fmt.Sprintf("Общая доходность: %.2f%%\n", metrics["total_return"])
	result += fmt.Sprintf("Максимальная просадка: %.2f%%\n", metrics["max_drawdown"])
	result += fmt.Sprintf("Текущий PnL: $%.2f\n", metrics["current_pnl"])
	return result
}
