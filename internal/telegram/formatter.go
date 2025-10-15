package telegram

import (
	"fmt"
	"strings"
	"time"

	"github.com/kirillm/dca-bot/internal/storage"
)

// Lang представляет язык
type Lang string

const (
	LangEN Lang = "en"
	LangRU Lang = "ru"
)

// Formatter форматирует ответы для пользователя
type Formatter struct {
	lang Lang
}

// NewFormatter создает новый форматтер
func NewFormatter(lang Lang) *Formatter {
	if lang != LangRU && lang != LangEN {
		lang = LangEN
	}
	return &Formatter{lang: lang}
}

// SetLang устанавливает язык
func (f *Formatter) SetLang(lang Lang) {
	f.lang = lang
}

// GetLang возвращает текущий язык
func (f *Formatter) GetLang() Lang {
	return f.lang
}

// T переводит строку
func (f *Formatter) T(key string) string {
	translations := map[string]map[Lang]string{
		"status":              {LangEN: "Status", LangRU: "Статус"},
		"history":             {LangEN: "Trade History", LangRU: "История сделок"},
		"portfolio":           {LangEN: "Portfolio", LangRU: "Портфель"},
		"config":              {LangEN: "Configuration", LangRU: "Конфигурация"},
		"price":               {LangEN: "Price", LangRU: "Цена"},
		"buy":                 {LangEN: "Buy", LangRU: "Покупка"},
		"sell":                {LangEN: "Sell", LangRU: "Продажа"},
		"autosell":            {LangEN: "Auto-Sell", LangRU: "Авто-продажа"},
		"grid":                {LangEN: "Grid", LangRU: "Сетка"},
		"risk":                {LangEN: "Risk", LangRU: "Риск"},
		"enabled":             {LangEN: "Enabled", LangRU: "Включено"},
		"disabled":            {LangEN: "Disabled", LangRU: "Выключено"},
		"active":              {LangEN: "Active", LangRU: "Активно"},
		"inactive":            {LangEN: "Inactive", LangRU: "Неактивно"},
		"success":             {LangEN: "Success", LangRU: "Успешно"},
		"error":               {LangEN: "Error", LangRU: "Ошибка"},
		"executing":           {LangEN: "Executing", LangRU: "Выполняется"},
		"completed":           {LangEN: "Completed", LangRU: "Завершено"},
		"no_position":         {LangEN: "No position", LangRU: "Нет позиции"},
		"no_trades":           {LangEN: "No trades yet", LangRU: "Нет сделок"},
		"current_price":       {LangEN: "Current Price", LangRU: "Текущая цена"},
		"avg_entry":           {LangEN: "Avg Entry Price", LangRU: "Средняя цена входа"},
		"quantity":            {LangEN: "Quantity", LangRU: "Количество"},
		"total_invested":      {LangEN: "Total Invested", LangRU: "Всего инвестировано"},
		"current_value":       {LangEN: "Current Value", LangRU: "Текущая стоимость"},
		"realized_profit":     {LangEN: "Realized Profit", LangRU: "Реализованная прибыль"},
		"unrealized_pnl":      {LangEN: "Unrealized P&L", LangRU: "Нереализованный P&L"},
		"total_pnl":           {LangEN: "Total P&L", LangRU: "Общий P&L"},
		"return_percent":      {LangEN: "Return", LangRU: "Доходность"},
		"active_orders":       {LangEN: "Active Orders", LangRU: "Активные ордера"},
		"levels":              {LangEN: "Levels", LangRU: "Уровни"},
		"spacing":             {LangEN: "Spacing", LangRU: "Интервал"},
		"order_size":          {LangEN: "Order Size", LangRU: "Размер ордера"},
		"trigger":             {LangEN: "Trigger", LangRU: "Триггер"},
		"sell_amount":         {LangEN: "Sell Amount", LangRU: "Объем продажи"},
		"emergency_stop":      {LangEN: "Emergency Stop", LangRU: "Экстренная остановка"},
		"max_daily_loss":      {LangEN: "Max Daily Loss", LangRU: "Макс. дневной убыток"},
		"max_exposure":        {LangEN: "Max Exposure", LangRU: "Макс. экспозиция"},
		"max_position_size":   {LangEN: "Max Position Size", LangRU: "Макс. размер позиции"},
		"max_order_size":      {LangEN: "Max Order Size", LangRU: "Макс. размер ордера"},
		"confirm_action":      {LangEN: "Please confirm this action:", LangRU: "Пожалуйста, подтвердите действие:"},
		"confirm":             {LangEN: "Confirm", LangRU: "Подтвердить"},
		"cancel":              {LangEN: "Cancel", LangRU: "Отмена"},
		"access_denied":       {LangEN: "Access denied", LangRU: "Доступ запрещен"},
		"admin_required":      {LangEN: "Admin permission required", LangRU: "Требуются права администратора"},
		"rate_limit_exceeded": {LangEN: "Too many requests, please wait", LangRU: "Слишком много запросов, подождите"},
		"invalid_symbol":      {LangEN: "Invalid symbol", LangRU: "Неверный символ"},
		"invalid_amount":      {LangEN: "Invalid amount", LangRU: "Неверная сумма"},
		"invalid_percent":     {LangEN: "Invalid percentage", LangRU: "Неверный процент"},
		"insufficient_balance": {LangEN: "Insufficient balance", LangRU: "Недостаточный баланс"},
	}

	if trans, ok := translations[key]; ok {
		if val, ok := trans[f.lang]; ok {
			return val
		}
	}
	return key
}

// FormatStatus форматирует статус системы
func (f *Formatter) FormatStatus(data map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString("📊 ")
	sb.WriteString(f.T("status"))
	sb.WriteString("\n\n")

	if activeAssets, ok := data["active_assets"].(int); ok {
		if f.lang == LangRU {
			sb.WriteString(fmt.Sprintf("Активных активов: %d\n", activeAssets))
		} else {
			sb.WriteString(fmt.Sprintf("Active Assets: %d\n", activeAssets))
		}
	}

	if strategies, ok := data["strategies"].([]string); ok {
		if f.lang == LangRU {
			sb.WriteString(fmt.Sprintf("Стратегии: %s\n", strings.Join(strategies, ", ")))
		} else {
			sb.WriteString(fmt.Sprintf("Strategies: %s\n", strings.Join(strategies, ", ")))
		}
	}

	if autoSellStatus, ok := data["autosell_status"].(bool); ok {
		status := f.T("disabled")
		if autoSellStatus {
			status = f.T("enabled")
		}
		sb.WriteString(fmt.Sprintf("Auto-Sell: %s\n", status))
	}

	if gridActive, ok := data["grid_active"].(bool); ok {
		status := f.T("inactive")
		if gridActive {
			status = f.T("active")
		}
		sb.WriteString(fmt.Sprintf("Grid: %s\n", status))
	}

	if uptime, ok := data["uptime"].(string); ok {
		if f.lang == LangRU {
			sb.WriteString(fmt.Sprintf("Время работы: %s\n", uptime))
		} else {
			sb.WriteString(fmt.Sprintf("Uptime: %s\n", uptime))
		}
	}

	return sb.String()
}

// FormatHistory форматирует историю сделок
func (f *Formatter) FormatHistory(trades []storage.Trade, symbol string, limit int) string {
	var sb strings.Builder

	sb.WriteString("📜 ")
	sb.WriteString(f.T("history"))
	if symbol != "" {
		sb.WriteString(fmt.Sprintf(" - %s", symbol))
	}
	sb.WriteString("\n\n")

	if len(trades) == 0 {
		sb.WriteString(f.T("no_trades"))
		return sb.String()
	}

	for i, trade := range trades {
		emoji := "🟢"
		if trade.Side == "SELL" {
			emoji = "🔴"
		}

		sb.WriteString(fmt.Sprintf("%s %d. %s %s\n", emoji, i+1, trade.Side, trade.Symbol))
		sb.WriteString(fmt.Sprintf("   %s: %.8f\n", f.T("quantity"), trade.Quantity))
		sb.WriteString(fmt.Sprintf("   %s: $%.2f\n", f.T("price"), trade.Price))
		sb.WriteString(fmt.Sprintf("   %s: $%.2f\n", f.T("total_invested"), trade.Amount))
		sb.WriteString(fmt.Sprintf("   %s: %s\n", "Strategy", trade.StrategyType))
		sb.WriteString(fmt.Sprintf("   %s\n\n", trade.CreatedAt.Format("2006-01-02 15:04")))
	}

	return sb.String()
}

// FormatPortfolio форматирует портфель
func (f *Formatter) FormatPortfolio(data map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString("💼 ")
	sb.WriteString(f.T("portfolio"))
	sb.WriteString("\n\n")

	if totalInvested, ok := data["total_invested"].(float64); ok {
		sb.WriteString(fmt.Sprintf("%s: $%.2f\n", f.T("total_invested"), totalInvested))
	}

	if currentValue, ok := data["current_value"].(float64); ok {
		sb.WriteString(fmt.Sprintf("%s: $%.2f\n", f.T("current_value"), currentValue))
	}

	if realizedProfit, ok := data["realized_profit"].(float64); ok {
		sb.WriteString(fmt.Sprintf("%s: $%.2f\n", f.T("realized_profit"), realizedProfit))
	}

	if unrealizedPnL, ok := data["unrealized_pnl"].(float64); ok {
		emoji := "📈"
		if unrealizedPnL < 0 {
			emoji = "📉"
		}
		sb.WriteString(fmt.Sprintf("%s %s: $%.2f\n", emoji, f.T("unrealized_pnl"), unrealizedPnL))
	}

	if totalPnL, ok := data["total_pnl"].(float64); ok {
		emoji := "💰"
		if totalPnL < 0 {
			emoji = "💸"
		}
		sb.WriteString(fmt.Sprintf("%s %s: $%.2f", emoji, f.T("total_pnl"), totalPnL))

		if returnPercent, ok := data["return_percent"].(float64); ok {
			sb.WriteString(fmt.Sprintf(" (%.2f%%)", returnPercent))
		}
		sb.WriteString("\n")
	}

	// Assets breakdown
	if assets, ok := data["assets"].([]map[string]interface{}); ok && len(assets) > 0 {
		sb.WriteString("\n")
		if f.lang == LangRU {
			sb.WriteString("Активы:\n")
		} else {
			sb.WriteString("Assets:\n")
		}

		for _, asset := range assets {
			symbol, _ := asset["symbol"].(string)
			qty, _ := asset["quantity"].(float64)
			price, _ := asset["current_price"].(float64)
			pnl, _ := asset["pnl"].(float64)
			pnlPercent, _ := asset["pnl_percent"].(float64)

			emoji := "🟢"
			if pnl < 0 {
				emoji = "🔴"
			}

			sb.WriteString(fmt.Sprintf("\n%s %s\n", emoji, symbol))
			sb.WriteString(fmt.Sprintf("  %s: %.8f\n", f.T("quantity"), qty))
			sb.WriteString(fmt.Sprintf("  %s: $%.2f\n", f.T("current_price"), price))
			sb.WriteString(fmt.Sprintf("  P&L: $%.2f (%.2f%%)\n", pnl, pnlPercent))
		}
	}

	return sb.String()
}

// FormatPrice форматирует информацию о цене
func (f *Formatter) FormatPrice(symbol string, currentPrice, avgEntry float64, hasPosition bool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("💵 %s\n\n", symbol))
	sb.WriteString(fmt.Sprintf("%s: $%.2f\n", f.T("current_price"), currentPrice))

	if hasPosition && avgEntry > 0 {
		sb.WriteString(fmt.Sprintf("%s: $%.2f\n", f.T("avg_entry"), avgEntry))

		priceDiff := currentPrice - avgEntry
		changePercent := (priceDiff / avgEntry) * 100

		emoji := "📈"
		if changePercent < 0 {
			emoji = "📉"
		}

		if f.lang == LangRU {
			sb.WriteString(fmt.Sprintf("%s Изменение: %.2f%%", emoji, changePercent))
		} else {
			sb.WriteString(fmt.Sprintf("%s Change: %.2f%%", emoji, changePercent))
		}
	}

	return sb.String()
}

// FormatGridStatus форматирует статус Grid
func (f *Formatter) FormatGridStatus(symbol string, metrics map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("🔷 Grid %s - %s\n\n", f.T("status"), symbol))

	if activeOrders, ok := metrics["active_orders"].(int); ok {
		sb.WriteString(fmt.Sprintf("%s: %d\n", f.T("active_orders"), activeOrders))
	}

	if currentPrice, ok := metrics["current_price"].(float64); ok {
		sb.WriteString(fmt.Sprintf("%s: $%.2f\n", f.T("current_price"), currentPrice))
	}

	if totalQty, ok := metrics["total_quantity"].(float64); ok {
		sb.WriteString(fmt.Sprintf("%s: %.8f\n", f.T("quantity"), totalQty))
	}

	if avgEntry, ok := metrics["avg_entry_price"].(float64); ok {
		sb.WriteString(fmt.Sprintf("%s: $%.2f\n", f.T("avg_entry"), avgEntry))
	}

	if totalInvested, ok := metrics["total_invested"].(float64); ok {
		sb.WriteString(fmt.Sprintf("%s: $%.2f\n", f.T("total_invested"), totalInvested))
	}

	if realizedProfit, ok := metrics["realized_profit"].(float64); ok {
		sb.WriteString(fmt.Sprintf("%s: $%.2f\n", f.T("realized_profit"), realizedProfit))
	}

	if unrealizedPnL, ok := metrics["unrealized_pnl"].(float64); ok {
		emoji := "📈"
		if unrealizedPnL < 0 {
			emoji = "📉"
		}
		sb.WriteString(fmt.Sprintf("%s %s: $%.2f\n", emoji, f.T("unrealized_pnl"), unrealizedPnL))
	}

	if totalPnL, ok := metrics["total_pnl"].(float64); ok {
		if returnPercent, ok := metrics["return_percent"].(float64); ok {
			emoji := "💰"
			if totalPnL < 0 {
				emoji = "💸"
			}
			sb.WriteString(fmt.Sprintf("%s %s: $%.2f (%.2f%%)", emoji, f.T("total_pnl"), totalPnL, returnPercent))
		}
	}

	return sb.String()
}

// FormatRiskStatus форматирует статус рисков
func (f *Formatter) FormatRiskStatus(limits *storage.RiskLimit, currentExposure, dailyLoss float64) string {
	var sb strings.Builder

	sb.WriteString("🛡️ ")
	sb.WriteString(f.T("risk"))
	sb.WriteString(" ")
	sb.WriteString(f.T("status"))
	sb.WriteString("\n\n")

	// Emergency Stop
	emergencyStatus := f.T("disabled")
	emoji := "🟢"
	if limits.EnableEmergencyStop {
		emergencyStatus = f.T("enabled")
		emoji = "🚨"
	}
	sb.WriteString(fmt.Sprintf("%s %s: %s\n\n", emoji, f.T("emergency_stop"), emergencyStatus))

	// Exposure
	exposurePercent := 0.0
	if limits.MaxTotalExposure > 0 {
		exposurePercent = (currentExposure / limits.MaxTotalExposure) * 100
	}
	sb.WriteString(fmt.Sprintf("%s: $%.2f / $%.2f (%.1f%%)\n",
		f.T("max_exposure"), currentExposure, limits.MaxTotalExposure, exposurePercent))

	// Daily Loss
	dailyLossPercent := 0.0
	if limits.MaxDailyLoss > 0 {
		dailyLossPercent = (dailyLoss / limits.MaxDailyLoss) * 100
	}
	sb.WriteString(fmt.Sprintf("%s: $%.2f / $%.2f (%.1f%%)\n",
		f.T("max_daily_loss"), dailyLoss, limits.MaxDailyLoss, dailyLossPercent))

	// Limits
	sb.WriteString(fmt.Sprintf("%s: $%.2f\n", f.T("max_position_size"), limits.MaxPositionSizeUSD))
	sb.WriteString(fmt.Sprintf("%s: $%.2f\n", f.T("max_order_size"), limits.MaxOrderSizeUSD))

	return sb.String()
}

// FormatError форматирует сообщение об ошибке
func (f *Formatter) FormatError(err error) string {
	return fmt.Sprintf("❌ %s: %v", f.T("error"), err)
}

// FormatSuccess форматирует сообщение об успехе
func (f *Formatter) FormatSuccess(message string) string {
	return fmt.Sprintf("✅ %s: %s", f.T("success"), message)
}

// FormatExecuting форматирует сообщение о выполнении
func (f *Formatter) FormatExecuting(action string) string {
	return fmt.Sprintf("🔄 %s %s...", f.T("executing"), action)
}

// FormatDuration форматирует длительность
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
}
