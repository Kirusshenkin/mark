package telegram

import (
	"context"
	"fmt"

	"github.com/kirillm/dca-bot/internal/ai"
	"github.com/kirillm/dca-bot/internal/ai/agents"
)

// ==================== STAGE 5: Hybrid AI Handlers ====================

// SetAIAgents устанавливает AI агенты (Stage 5)
func (b *Bot) SetAIAgents(
	agentRouter *agents.AgentRouter,
	chatAgent *agents.ChatAgent,
	analysisAgent *agents.AnalysisAgent,
	decisionAgent *agents.DecisionAgent,
	actionExecutor *ai.ActionExecutor,
) {
	b.agentRouter = agentRouter
	b.chatAgent = chatAgent
	b.analysisAgent = analysisAgent
	b.decisionAgent = decisionAgent
	b.actionExecutor = actionExecutor
	b.logger.Info("Stage 5 AI agents configured")
}

// handleAIChat обрабатывает сообщение через ChatAgent
func (b *Bot) handleAIChat(text string) {
	if b.agentRouter == nil {
		// Fallback на legacy AI client
		b.handleAIMessage(text)
		return
	}

	b.logger.Info("[AI Chat] User message: %s", text)

	// Получаем контекст
	contextInfo := b.buildContextString()

	// Обрабатываем через AgentRouter
	response, actions, err := b.agentRouter.Process(
		context.Background(),
		text,
		contextInfo,
	)

	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ AI error: %v", err))
		return
	}

	// Выполняем действия через ActionExecutor
	for _, action := range actions {
		result, err := b.actionExecutor.ExecuteAction(action)
		if err != nil {
			b.logger.Error("Failed to execute AI action %s: %v", action.Type, err)
			response += fmt.Sprintf("\n❌ Action %s failed: %v", action.Type, err)
		} else {
			response += "\n" + result
		}
	}

	// Отправляем ответ
	if response != "" {
		b.SendMessage(response)
	}
}

// handleAIAnalysis запрашивает анализ через AnalysisAgent
func (b *Bot) handleAIAnalysis(symbol string) {
	if b.analysisAgent == nil {
		// Fallback на legacy
		b.handleAnalysis()
		return
	}

	if symbol == "" {
		symbol = "BTCUSDT"
	}

	b.logger.Info("[AI Analysis] Symbol: %s", symbol)

	// Получаем данные
	price, err := b.exchange.GetPrice(symbol)
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Error getting price: %v", err))
		return
	}

	balance, err := b.storage.GetBalance(symbol)
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Error getting balance: %v", err))
		return
	}

	// Формируем контекст для AnalysisAgent
	contextInfo := fmt.Sprintf(
		"Symbol: %s\nCurrent price: %.2f USDT\nAverage entry: %.2f USDT\nQuantity: %.8f",
		symbol,
		price,
		balance.AvgEntryPrice,
		balance.TotalQuantity,
	)

	// Запрашиваем анализ
	b.SendMessage("🤔 Analyzing market with local AI...")

	analysis, _, err := b.analysisAgent.Process(
		context.Background(),
		fmt.Sprintf("Проанализируй %s", symbol),
		contextInfo,
	)

	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Analysis failed: %v", err))
		return
	}

	b.SendMessage(fmt.Sprintf("🧠 AI Analysis:\n\n%s", analysis))
}

// handleAIDecision запрашивает стратегическое решение через DecisionAgent
func (b *Bot) handleAIDecision() {
	if b.decisionAgent == nil {
		b.SendMessage("⚠️ DecisionAgent not configured. Check Cloud AI settings.")
		return
	}

	b.logger.Info("[AI Decision] Requesting strategic decision...")
	b.SendMessage("🤖 Requesting strategic AI decision (cloud model)...")

	// Собираем данные для DecisionRequest
	portfolio, err := b.buildPortfolioSnapshot()
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Error building portfolio: %v", err))
		return
	}

	market := b.buildMarketConditions()
	news := b.buildNewsSignals()
	risks := b.buildRiskLimits()

	// Создаем запрос
	req := ai.DecisionRequest{
		CurrentPortfolio: portfolio,
		MarketConditions: market,
		RecentNews:       news,
		RiskLimits:       risks,
		Mode:             b.decisionAgent.GetMode(),
	}

	// Запрашиваем решение
	decision, err := b.decisionAgent.RequestDecision(context.Background(), req)
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Decision failed: %v", err))
		return
	}

	// Форматируем и отправляем решение
	formattedDecision := b.decisionAgent.FormatDecision(decision)
	b.SendMessage(formattedDecision)

	// Выполняем действия (если режим не shadow)
	if b.decisionAgent.GetMode() != "shadow" && len(decision.Actions) > 0 {
		b.SendMessage("\n🔄 Executing AI decisions...")

		for _, action := range decision.Actions {
			result, err := b.actionExecutor.ExecuteAction(ai.AIAction{
				Type:       action.Type,
				Parameters: action.Parameters,
			})

			if err != nil {
				b.SendMessage(fmt.Sprintf("❌ Action failed: %v", err))
			} else {
				b.SendMessage(fmt.Sprintf("✅ %s", result))
			}
		}
	}
}

// handleAIMetrics показывает метрики AgentRouter
func (b *Bot) handleAIMetrics() {
	if b.agentRouter == nil {
		b.SendMessage("⚠️ AgentRouter not configured")
		return
	}

	metrics := b.agentRouter.FormatMetrics()
	b.SendMessage(metrics)
}

// handleAIMode показывает или изменяет режим DecisionAgent
func (b *Bot) handleAIMode(args string) {
	if b.decisionAgent == nil {
		b.SendMessage("⚠️ DecisionAgent not configured")
		return
	}

	// Показываем текущий режим
	if args == "" {
		currentMode := b.decisionAgent.GetMode()
		message := fmt.Sprintf(
			"🤖 AI Decision Mode\n\n"+
				"Current: %s\n\n"+
				"Available modes:\n"+
				"• shadow - AI decides but doesn't execute\n"+
				"• pilot - Conservative limits (50%%)\n"+
				"• full - Full autonomy\n\n"+
				"Commands:\n"+
				"/ai_mode shadow\n"+
				"/ai_mode pilot\n"+
				"/ai_mode full",
			currentMode,
		)
		b.SendMessage(message)
		return
	}

	// Изменяем режим
	if err := b.decisionAgent.SetMode(args); err != nil {
		b.SendMessage(fmt.Sprintf("❌ Invalid mode: %v", err))
		return
	}

	b.SendMessage(fmt.Sprintf("✅ AI Decision mode changed to: %s", args))
}

// ==================== Helper functions ====================

// buildContextString строит контекст для AI
func (b *Bot) buildContextString() string {
	dcaStatus, _ := b.dcaStrategy.GetStatus()
	autoSellStatus, _ := b.autoSell.GetStatus()

	return fmt.Sprintf("%s\n\n%s", dcaStatus, autoSellStatus)
}

// buildPortfolioSnapshot создает снимок портфеля
func (b *Bot) buildPortfolioSnapshot() (ai.PortfolioSnapshot, error) {
	// Получаем все балансы
	balances, err := b.storage.GetAllBalances()
	if err != nil {
		return ai.PortfolioSnapshot{}, err
	}

	var assets []ai.AssetStatus
	var totalValue, totalInvested, totalPnL float64

	for _, bal := range balances {
		if bal.TotalQuantity <= 0 {
			continue
		}

		// Получаем текущую цену
		currentPrice, err := b.exchange.GetPrice(bal.Symbol)
		if err != nil {
			continue
		}

		currentValue := bal.TotalQuantity * currentPrice
		invested := bal.TotalInvested
		pnl := currentValue - invested
		pnlPercent := 0.0
		if invested > 0 {
			pnlPercent = (pnl / invested) * 100
		}

		assets = append(assets, ai.AssetStatus{
			Symbol:        bal.Symbol,
			Quantity:      bal.TotalQuantity,
			AvgEntryPrice: bal.AvgEntryPrice,
			CurrentPrice:  currentPrice,
			InvestedUSDT:  invested,
			CurrentValue:  currentValue,
			PnL:           pnl,
			PnLPercent:    pnlPercent,
		})

		totalValue += currentValue
		totalInvested += invested
		totalPnL += pnl
	}

	totalPnLPercent := 0.0
	if totalInvested > 0 {
		totalPnLPercent = (totalPnL / totalInvested) * 100
	}

	return ai.PortfolioSnapshot{
		Assets:          assets,
		TotalValueUSDT:  totalValue,
		TotalInvested:   totalInvested,
		TotalPnL:        totalPnL,
		TotalPnLPercent: totalPnLPercent,
	}, nil
}

// buildMarketConditions собирает рыночные данные
func (b *Bot) buildMarketConditions() ai.MarketData {
	// Получаем цену BTC
	btcPrice, _ := b.exchange.GetPrice("BTCUSDT")

	// TODO: Получить изменение за 24ч и волатильность
	// Пока используем заглушки
	return ai.MarketData{
		BTCPrice:        btcPrice,
		BTCChange24h:    0.0, // TODO: реализовать
		MarketSentiment: "neutral",
		Volatility:      2.0, // TODO: реализовать
	}
}

// buildNewsSignals собирает новостные сигналы
func (b *Bot) buildNewsSignals() []ai.NewsSignal {
	// TODO: Интеграция с NewsSignalRepository
	// Пока возвращаем пустой массив
	return []ai.NewsSignal{}
}

// buildRiskLimits собирает лимиты рисков
func (b *Bot) buildRiskLimits() ai.RiskLimits {
	// Получаем из storage
	limits, err := b.storage.GetRiskLimits()
	if err != nil {
		// Используем дефолтные
		return ai.RiskLimits{
			MaxOrderUSDT:     100,
			MaxPositionUSDT:  1000,
			MaxTotalExposure: 3000,
			MaxDailyLoss:     100,
		}
	}

	return ai.RiskLimits{
		MaxOrderUSDT:     limits.MaxOrderSizeUSD,
		MaxPositionUSDT:  limits.MaxPositionSizeUSD,
		MaxTotalExposure: limits.MaxTotalExposure,
		MaxDailyLoss:     limits.MaxDailyLoss,
	}
}
