package telegram

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kirillm/dca-bot/internal/ai"
	"github.com/kirillm/dca-bot/internal/ai/agents"
	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
	"github.com/kirillm/dca-bot/internal/strategy"
	"github.com/kirillm/dca-bot/pkg/utils"
)

type Bot struct {
	api              *tgbotapi.BotAPI
	chatID           int64
	logger           *utils.Logger
	exchange         *exchange.BybitClient
	storage          *storage.PostgresStorage
	aiClient         *ai.AIClient // Legacy AI client
	dcaStrategy      *strategy.DCAStrategy
	autoSell         *strategy.AutoSellStrategy
	gridStrategy     *strategy.GridStrategy
	portfolioManager *strategy.PortfolioManager
	orchestrator     Orchestrator // Stage 4
	policyEngine     PolicyEngine // Stage 4

	// Stage 5: Hybrid AI
	agentRouter    *agents.AgentRouter
	chatAgent      *agents.ChatAgent
	analysisAgent  *agents.AnalysisAgent
	decisionAgent  *agents.DecisionAgent
	actionExecutor *ai.ActionExecutor
}

// Orchestrator interface for Stage 4
type Orchestrator interface {
	GetMode() string
	SetMode(mode string) error
	IsRunning() bool
}

// PolicyEngine interface for Stage 4
type PolicyEngine interface {
	GetPolicy() interface{}
	GetMetrics() interface{}
}

func NewBot(
	token string,
	chatID int64,
	logger *utils.Logger,
	ex *exchange.BybitClient,
	st *storage.PostgresStorage,
	aiClient *ai.AIClient,
	dcaStrategy *strategy.DCAStrategy,
	autoSell *strategy.AutoSellStrategy,
	gridStrategy *strategy.GridStrategy,
	portfolioManager *strategy.PortfolioManager,
) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	logger.Info("Telegram bot authorized: @%s", bot.Self.UserName)

	return &Bot{
		api:              bot,
		chatID:           chatID,
		logger:           logger,
		exchange:         ex,
		storage:          st,
		aiClient:         aiClient,
		dcaStrategy:      dcaStrategy,
		autoSell:         autoSell,
		gridStrategy:     gridStrategy,
		portfolioManager: portfolioManager,
	}, nil
}

// Start запускает обработку сообщений
func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	b.SendMessage("🤖 Crypto Trading Bot started!\nUse /help to see available commands.")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		// Проверяем, что сообщение от правильного пользователя
		if update.Message.Chat.ID != b.chatID {
			b.logger.Warn("Unauthorized access attempt from chat ID: %d", update.Message.Chat.ID)
			continue
		}

		go b.handleMessage(update.Message)
	}
}

// handleMessage обрабатывает входящее сообщение
func (b *Bot) handleMessage(message *tgbotapi.Message) {
	b.logger.Info("Received message: %s", message.Text)

	// Обработка команд
	if message.IsCommand() {
		b.handleCommand(message)
		return
	}

	// Обработка текстовых сообщений через AI
	b.handleAIMessage(message.Text)
}

// handleCommand обрабатывает команды
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start", "help":
		b.sendHelp()

	case "status":
		b.handleStatus()

	case "history":
		b.handleHistory()

	case "config":
		b.handleConfig()

	case "buy":
		b.handleManualBuy()

	case "sell":
		b.handleManualSell(message.CommandArguments())

	case "autosell_on":
		b.autoSell.Enable()

	case "autosell_off":
		b.autoSell.Disable()

	case "price":
		b.handlePrice(message.CommandArguments())

	case "analysis":
		b.handleAnalysis()

	case "portfolio":
		b.handlePortfolio()

	case "grid_init":
		b.handleGridInit(message.CommandArguments())

	case "grid_status":
		b.handleGridStatus(message.CommandArguments())

	// Stage 4: Autonomous Trading Commands
	case "mode":
		b.handleMode(message.CommandArguments())

	case "mode_shadow":
		b.handleMode("shadow")

	case "mode_pilot":
		b.handleMode("pilot")

	case "mode_full":
		b.handleMode("full")

	case "decisions":
		b.handleDecisions()

	case "circuit":
		b.handleCircuit()

	case "policy":
		b.handlePolicyStatus()

	case "metrics":
		b.handleMetrics()

	// Stage 5: Hybrid AI Commands
	case "ai_analyze":
		b.handleAIAnalysis(message.CommandArguments())

	case "ai_decision":
		b.handleAIDecision()

	case "ai_metrics":
		b.handleAIMetrics()

	case "ai_mode":
		b.handleAIMode(message.CommandArguments())

	default:
		b.SendMessage("Unknown command. Use /help to see available commands.")
	}
}

// sendHelp отправляет справку
func (b *Bot) sendHelp() {
	help := `🤖 AI Криптотрейдинг Бот

📊 МОНИТОРИНГ
/status - Текущая позиция и статистика
/history - История последних сделок
/portfolio - Обзор портфеля
/price [SYMBOL] - Текущая цена актива

💬 HYBRID AI (Stage 5)
/ai_analyze [SYMBOL] - Рыночный анализ (локальный)
/ai_decision - Стратегическое решение (облачный)
/ai_metrics - Метрики AI агентов
/ai_mode [shadow|pilot|full] - Режим DecisionAgent

Просто пишите сообщение боту для общения с ChatAgent!

🧠 АВТОНОМНЫЙ AI (Stage 4)
/mode - Показать текущий режим Orchestrator
/mode_shadow - Режим Тени (без сделок)
/mode_pilot - Режим Пилота (50% лимиты)
/mode_full - Полная Автоматизация
/decisions - История AI решений
/policy - Текущая политика рисков
/metrics - Метрики рисков

📈 СТРАТЕГИИ
/buy - Ручная покупка DCA
/sell <PERCENT> - Ручная продажа
/autosell_on - Включить Auto-Sell
/autosell_off - Выключить Auto-Sell
/grid_init <SYMBOL> - Инициализировать Grid
/grid_status [SYMBOL] - Статус Grid стратегии

Примечание: AI Orchestrator работает каждые 15 минут автоматически.
DecisionAgent вызывается по команде или по расписанию.
`
	// Send without markdown parsing
	message := tgbotapi.NewMessage(b.chatID, help)
	message.ParseMode = "" // No parsing
	if _, err := b.api.Send(message); err != nil {
		b.logger.Error("Failed to send help message: %v", err)
	}
}

// handleStatus показывает текущий статус
func (b *Bot) handleStatus() {
	dcaStatus, err := b.dcaStrategy.GetStatus()
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Error getting DCA status: %v", err))
		return
	}

	autoSellStatus, err := b.autoSell.GetStatus()
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Error getting Auto-Sell status: %v", err))
		return
	}

	message := fmt.Sprintf("%s\n\n%s", dcaStatus, autoSellStatus)
	b.SendMessage(message)
}

// handleHistory показывает историю сделок
func (b *Bot) handleHistory() {
	symbol := "BTCUSDT" // TODO: get from config
	trades, err := b.storage.GetRecentTrades(symbol, 10)
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Error getting trade history: %v", err))
		return
	}

	if len(trades) == 0 {
		b.SendMessage("No trades yet.")
		return
	}

	message := "📜 Recent Trades:\n\n"
	for i, trade := range trades {
		message += fmt.Sprintf("%d. %s %s\n   Qty: %.8f\n   Price: %.2f USDT\n   Amount: %.2f USDT\n   Time: %s\n\n",
			i+1, trade.Side, trade.Symbol, trade.Quantity, trade.Price, trade.Amount,
			trade.CreatedAt.Format("2006-01-02 15:04"))
	}

	b.SendMessage(message)
}

// handleConfig показывает текущую конфигурацию
func (b *Bot) handleConfig() {
	// TODO: Implement config display
	b.SendMessage("⚙️ Configuration display - coming soon")
}

// handleManualBuy выполняет ручную покупку
func (b *Bot) handleManualBuy() {
	b.SendMessage("🔄 Executing manual buy...")
	if err := b.dcaStrategy.ExecuteManualBuy(); err != nil {
		b.SendMessage(fmt.Sprintf("❌ Manual buy failed: %v", err))
		return
	}
}

// handleManualSell выполняет ручную продажу
func (b *Bot) handleManualSell(args string) {
	if args == "" {
		b.SendMessage("❌ Please specify percentage to sell. Example: /sell 50")
		return
	}

	percent, err := strconv.ParseFloat(args, 64)
	if err != nil || percent <= 0 || percent > 100 {
		b.SendMessage("❌ Invalid percentage. Use a number between 1 and 100.")
		return
	}

	b.SendMessage(fmt.Sprintf("🔄 Executing manual sell of %.0f%% of position...", percent))
	if err := b.autoSell.ExecuteManualSell(percent); err != nil {
		b.SendMessage(fmt.Sprintf("❌ Manual sell failed: %v", err))
		return
	}
}

// handlePrice показывает текущую цену
func (b *Bot) handlePrice(args string) {
	symbol := args
	if symbol == "" {
		symbol = "BTCUSDT" // default
	}

	price, err := b.exchange.GetPrice(symbol)
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Error getting price for %s: %v", symbol, err))
		return
	}

	balance, err := b.storage.GetBalance(symbol)
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Error getting balance for %s: %v", symbol, err))
		return
	}

	profitPercent := 0.0
	if balance.AvgEntryPrice > 0 {
		profitPercent = ((price - balance.AvgEntryPrice) / balance.AvgEntryPrice) * 100
	}

	message := fmt.Sprintf(
		"💵 %s\n\n"+
			"Current Price: %.2f USDT\n"+
			"Avg Entry: %.2f USDT\n"+
			"Change: %.2f%%",
		symbol, price, balance.AvgEntryPrice, profitPercent,
	)

	b.SendMessage(message)
}

// handleAnalysis запрашивает AI анализ рынка (legacy)
func (b *Bot) handleAnalysis() {
	// Если есть AnalysisAgent, используем его
	if b.analysisAgent != nil {
		b.handleAIAnalysis("BTCUSDT")
		return
	}

	// Legacy fallback
	if b.aiClient == nil {
		b.SendMessage("AI client not configured")
		return
	}

	symbol := "BTCUSDT" // TODO: get from config

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

	b.SendMessage("🤔 Analyzing market...")

	analysis, err := b.aiClient.GetMarketAnalysis(symbol, price, balance.AvgEntryPrice)
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ AI analysis failed: %v", err))
		return
	}

	b.SendMessage(fmt.Sprintf("🧠 AI Analysis:\n\n%s", analysis))
}

// handleAIMessage обрабатывает сообщение через AI (используется для не-команд)
func (b *Bot) handleAIMessage(text string) {
	// Если есть AgentRouter, используем его
	if b.agentRouter != nil {
		b.handleAIChat(text)
		return
	}

	// Иначе используем legacy AI client
	if b.aiClient == nil {
		b.SendMessage("AI client not configured")
		return
	}

	// Получаем контекст
	context, err := b.buildContext()
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Error building context: %v", err))
		return
	}

	// Отправляем AI
	reply, actions, err := b.aiClient.ProcessMessage(text, context)
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ AI error: %v", err))
		return
	}

	// Выполняем действия
	for _, action := range actions {
		if err := b.executeAIAction(action); err != nil {
			b.logger.Error("Failed to execute AI action %s: %v", action.Type, err)
		}
	}

	// Отправляем ответ
	if reply != "" {
		b.SendMessage(reply)
	}
}

// buildContext строит контекст для AI
func (b *Bot) buildContext() (string, error) {
	status, err := b.dcaStrategy.GetStatus()
	if err != nil {
		return "", err
	}

	autoSellStatus, err := b.autoSell.GetStatus()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s\n\n%s", status, autoSellStatus), nil
}

// executeAIAction выполняет действие, предложенное AI
func (b *Bot) executeAIAction(action ai.AIAction) error {
	b.logger.Info("Executing AI action: %s", action.Type)

	switch action.Type {
	case "get_status":
		b.handleStatus()

	case "update_dca_amount":
		if amount, ok := action.Parameters["amount"].(float64); ok {
			b.dcaStrategy.UpdateAmount(amount)
		}

	case "update_autosell_trigger":
		if percent, ok := action.Parameters["percent"].(float64); ok {
			b.autoSell.UpdateTriggerPercent(percent)
		}

	case "update_autosell_amount":
		if percent, ok := action.Parameters["percent"].(float64); ok {
			b.autoSell.UpdateSellAmountPercent(percent)
		}

	case "enable_autosell":
		b.autoSell.Enable()

	case "disable_autosell":
		b.autoSell.Disable()

	case "manual_buy":
		if err := b.dcaStrategy.ExecuteManualBuy(); err != nil {
			return err
		}

	case "manual_sell":
		if percent, ok := action.Parameters["percent"].(float64); ok {
			if err := b.autoSell.ExecuteManualSell(percent); err != nil {
				return err
			}
		}

	case "get_history":
		b.handleHistory()

	case "get_price":
		b.handlePrice("")

	case "get_portfolio":
		b.handlePortfolio()

	case "init_grid":
		if symbol, ok := action.Parameters["symbol"].(string); ok {
			b.handleGridInit(symbol)
		}

	case "get_grid_status":
		if symbol, ok := action.Parameters["symbol"].(string); ok {
			b.handleGridStatus(symbol)
		}

	default:
		b.logger.Warn("Unknown AI action: %s", action.Type)
	}

	return nil
}

// SendMessage отправляет сообщение пользователю
func (b *Bot) SendMessage(text string) {
	// Разбиваем длинные сообщения
	const maxLength = 4096
	messages := splitMessage(text, maxLength)

	for _, msg := range messages {
		message := tgbotapi.NewMessage(b.chatID, msg)
		message.ParseMode = "Markdown"
		if _, err := b.api.Send(message); err != nil {
			b.logger.Error("Failed to send telegram message: %v", err)
		}
	}
}

// splitMessage разбивает длинное сообщение на части
func splitMessage(text string, maxLength int) []string {
	if len(text) <= maxLength {
		return []string{text}
	}

	var messages []string
	lines := strings.Split(text, "\n")
	currentMessage := ""

	for _, line := range lines {
		if len(currentMessage)+len(line)+1 > maxLength {
			messages = append(messages, currentMessage)
			currentMessage = line
		} else {
			if currentMessage != "" {
				currentMessage += "\n"
			}
			currentMessage += line
		}
	}

	if currentMessage != "" {
		messages = append(messages, currentMessage)
	}

	return messages
}

// handlePortfolio показывает портфель
func (b *Bot) handlePortfolio() {
	if b.portfolioManager == nil {
		b.SendMessage("Portfolio manager not available")
		return
	}

	status, err := b.portfolioManager.GetStatus()
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Error getting portfolio: %v", err))
		return
	}

	b.SendMessage(status)
}

// handleGridInit инициализирует Grid для символа
func (b *Bot) handleGridInit(args string) {
	if args == "" {
		b.SendMessage("❌ Please specify symbol. Example: /grid_init ETHUSDT")
		return
	}

	symbol := args
	if b.gridStrategy == nil {
		b.SendMessage("Grid strategy not available")
		return
	}

	// Параметры по умолчанию
	levels := 10
	spacingPercent := 2.5
	orderSizeQuote := 100.0

	b.SendMessage(fmt.Sprintf("🔄 Initializing Grid for %s...", symbol))

	// Инициализируем Grid через API
	// Пока используем GetAsset для получения настроек
	asset, err := b.storage.GetAsset(symbol)
	if err != nil || asset == nil {
		// Если актив не найден, создаем дефолтный
		asset = &storage.Asset{
			Symbol:             symbol,
			Enabled:            true,
			StrategyType:       "GRID",
			AllocatedCapital:   1000,
			MaxPositionSize:    5000,
			GridLevels:         levels,
			GridSpacingPercent: spacingPercent,
			GridOrderSize:      orderSizeQuote,
		}
		if err := b.storage.CreateOrUpdateAsset(asset); err != nil {
			b.SendMessage(fmt.Sprintf("❌ Failed to create asset: %v", err))
			return
		}
	}

	if err := b.gridStrategy.InitializeGrid(asset); err != nil {
		b.SendMessage(fmt.Sprintf("❌ Grid initialization failed: %v", err))
		return
	}

	b.SendMessage(fmt.Sprintf("✅ Grid initialized for %s\nLevels: %d\nSpacing: %.2f%%\nOrder Size: %.2f USDT",
		symbol, levels, spacingPercent, orderSizeQuote))
}

// handleGridStatus показывает статус Grid
func (b *Bot) handleGridStatus(args string) {
	symbol := args
	if symbol == "" {
		symbol = "BTCUSDT" // default
	}

	if b.gridStrategy == nil {
		b.SendMessage("Grid strategy not available")
		return
	}

	metrics, err := b.gridStrategy.CalculateGridMetrics(symbol)
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Error getting Grid status: %v", err))
		return
	}

	message := fmt.Sprintf(
		"🔷 Grid Status - %s\n\n"+
			"Active Orders: %v\n"+
			"Current Price: %.2f USDT\n"+
			"Total Quantity: %.8f\n"+
			"Avg Entry Price: %.2f USDT\n"+
			"Total Invested: %.2f USDT\n"+
			"Total Sold: %.2f USDT\n"+
			"Realized Profit: %.2f USDT\n"+
			"Unrealized P&L: %.2f USDT\n"+
			"Total P&L: %.2f USDT (%.2f%%)",
		symbol,
		metrics["active_orders"],
		metrics["current_price"],
		metrics["total_quantity"],
		metrics["avg_entry_price"],
		metrics["total_invested"],
		metrics["total_sold"],
		metrics["realized_profit"],
		metrics["unrealized_pnl"],
		metrics["total_pnl"],
		metrics["return_percent"],
	)

	b.SendMessage(message)
}

// ==================== STAGE 4: Autonomous Trading Handlers ====================

// handleMode переключает или показывает режим AI
func (b *Bot) handleMode(args string) {
	if b.orchestrator == nil {
		b.SendMessage("⚠️ Orchestrator not available. Stage 4 not enabled.")
		return
	}

	// Если аргумент не указан - показываем текущий режим
	if args == "" {
		currentMode := b.orchestrator.GetMode()
		running := "🔴 Остановлен"
		if b.orchestrator.IsRunning() {
			running = "🟢 Работает"
		}

		message := fmt.Sprintf(
			"🧠 Режим AI Трейдинга\n\n"+
				"Текущий режим: %s\n"+
				"Статус: %s\n\n"+
				"Доступные режимы:\n"+
				"• shadow - AI решает, но не торгует (логирование)\n"+
				"• pilot - Торговля с 50%% лимитами (тестирование)\n"+
				"• full - Полная автономия в рамках лимитов\n\n"+
				"Команды:\n"+
				"/mode_shadow - Переключить на Тень\n"+
				"/mode_pilot - Переключить на Пилот\n"+
				"/mode_full - Переключить на Полную",
			currentMode, running,
		)
		b.SendMessage(message)
		return
	}

	// Валидация режима
	mode := strings.ToLower(args)
	if mode != "shadow" && mode != "pilot" && mode != "full" {
		b.SendMessage("❌ Invalid mode. Use: shadow, pilot, or full")
		return
	}

	// Переключаем режим
	if err := b.orchestrator.SetMode(mode); err != nil {
		b.SendMessage(fmt.Sprintf("❌ Ошибка переключения режима: %v", err))
		return
	}

	b.SendMessage(fmt.Sprintf("✅ Переключено на режим %s\n\n%s",
		mode,
		getModeDescription(mode),
	))
}

// getModeDescription возвращает описание режима
func getModeDescription(mode string) string {
	descriptions := map[string]string{
		"shadow": "🔍 Режим Тени:\n• AI генерирует решения\n• Без реального исполнения\n• Все решения логируются\n• Безопасно для тестирования",
		"pilot":  "✈️ Режим Пилота:\n• AI решения исполняются\n• 50% от нормальных лимитов\n• Консервативный риск-профиль\n• Хорошо для валидации",
		"full":   "🚀 Режим Полной Автоматизации:\n• Полная автономия\n• Полные лимиты рисков\n• Минимальное вмешательство\n• Максимальная производительность",
	}
	return descriptions[mode]
}

// handleDecisions показывает последние решения AI
func (b *Bot) handleDecisions() {
	// TODO: Implement with storage repository
	b.SendMessage("🧠 Recent AI Decisions:\n\n_Feature coming soon - requires database integration_\n\nThis command will show:\n• Last 10 AI decisions\n• Regime (ACCUMULATE/TREND_FOLLOW/RANGE_GRID/DEFENSE)\n• Confidence scores\n• Actions taken\n• Approval status")
}

// handleCircuit проверяет статус circuit breakers
func (b *Bot) handleCircuit() {
	// TODO: Implement with storage repository
	b.SendMessage("🛡️ Circuit Breaker Status:\n\n_Feature coming soon - requires database integration_\n\nThis command will show:\n• Active circuit breakers\n• Triggered reasons\n• Pause duration\n• Last trigger time")
}

// handlePolicyStatus показывает текущую политику рисков
func (b *Bot) handlePolicyStatus() {
	if b.policyEngine == nil {
		b.SendMessage("⚠️ Policy Engine not available. Stage 4 not enabled.")
		return
	}

	// TODO: Extract actual values from policy engine once interface is extended
	_ = b.policyEngine.GetPolicy() // Placeholder for future use

	message := fmt.Sprintf(
		"⚙️ Risk Management Policy\n\n"+
			"Profile: *%v*\n\n"+
			"Limits:\n"+
			"• Max Order: $%v USDT\n"+
			"• Max Position: $%v USDT\n"+
			"• Max Exposure: $%v USDT\n"+
			"• Max Daily Loss: $%v USDT\n"+
			"• Trades/Hour: %v\n\n"+
			"_Configured in configs/policy.yaml_",
		"moderate",              // TODO: Get actual profile
		100, 1000, 3000, 100, 5, // TODO: Get actual values from policy
	)

	b.SendMessage(message)
}

// handleMetrics показывает метрики рисков
func (b *Bot) handleMetrics() {
	if b.policyEngine == nil {
		b.SendMessage("⚠️ Policy Engine not available. Stage 4 not enabled.")
		return
	}

	// TODO: Extract actual values from metrics once interface is extended
	_ = b.policyEngine.GetMetrics() // Placeholder for future use

	message := fmt.Sprintf(
		"📊 Risk Metrics\n\n"+
			"Current Metrics:\n"+
			"• Total Exposure: $%.2f USDT\n"+
			"• Daily Loss: $%.2f USDT\n"+
			"• Daily Trades: %d\n"+
			"• Current Drawdown: %.2f%%\n"+
			"• Risk Score: %.2f/1.0\n\n"+
			"_Updated in real-time_",
		0.0, 0.0, 0, 0.0, 0.0, // TODO: Get actual metrics
	)

	b.SendMessage(message)
}

// SetOrchestrator устанавливает orchestrator (для Stage 4)
func (b *Bot) SetOrchestrator(orchestrator Orchestrator) {
	b.orchestrator = orchestrator
}

// SetPolicyEngine устанавливает policy engine (для Stage 4)
func (b *Bot) SetPolicyEngine(policyEngine PolicyEngine) {
	b.policyEngine = policyEngine
}
