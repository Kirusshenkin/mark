package telegram

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kirillm/dca-bot/internal/ai"
	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
	"github.com/kirillm/dca-bot/internal/strategy"
	"github.com/kirillm/dca-bot/pkg/utils"
)

// BotV2 представляет новую версию Telegram бота с полной архитектурой
type BotV2 struct {
	api              *tgbotapi.BotAPI
	logger           *utils.Logger
	router           *Router
	handlers         *Handlers
	authManager      *AuthManager
	formatter        *Formatter
	aiClient         *ai.AIClient
	actionExecutor   *ai.ActionExecutor
	userLangs        map[int64]Lang
	userLangsMu      sync.RWMutex
	previewMode      bool
}

// NewBotV2 создает новый бот с полной архитектурой
func NewBotV2(
	token string,
	logger *utils.Logger,
	exchange *exchange.BybitClient,
	storage *storage.PostgresStorage,
	aiClient *ai.AIClient,
	dcaStrategy *strategy.DCAStrategy,
	autoSell *strategy.AutoSellStrategy,
	gridStrategy *strategy.GridStrategy,
	portfolioManager *strategy.PortfolioManager,
	riskManager *strategy.RiskManager,
	defaultSymbol string,
) (*BotV2, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	logger.Info("Telegram bot authorized: @%s", bot.Self.UserName)

	// Создаем auth manager
	adminIDsStr := os.Getenv("TG_ADMINS")
	whitelistStr := os.Getenv("TG_CHAT_WHITELIST")
	authManager := NewAuthManager(adminIDsStr, whitelistStr)

	// Создаем formatter с дефолтным языком
	defaultLang := LangEN
	if os.Getenv("DEFAULT_LANG") == "ru" {
		defaultLang = LangRU
	}
	formatter := NewFormatter(defaultLang)

	// Создаем validator
	validator := NewValidator(storage, exchange)

	// Создаем handlers
	handlers := NewHandlers(
		exchange,
		storage,
		validator,
		formatter,
		dcaStrategy,
		autoSell,
		gridStrategy,
		portfolioManager,
		riskManager,
		defaultSymbol,
	)

	// Создаем router и регистрируем обработчики
	router := NewRouter(authManager, validator, formatter)
	registerHandlers(router, handlers)

	// Action executor для AI
	actionExecutor := ai.NewActionExecutor(
		storage,
		exchange,
		portfolioManager,
		riskManager,
		gridStrategy,
	)

	// Preview mode
	previewMode := os.Getenv("PREVIEW_MODE") == "true"

	b := &BotV2{
		api:            bot,
		logger:         logger,
		router:         router,
		handlers:       handlers,
		authManager:    authManager,
		formatter:      formatter,
		aiClient:       aiClient,
		actionExecutor: actionExecutor,
		userLangs:      make(map[int64]Lang),
		previewMode:    previewMode,
	}

	// Запускаем периодическую очистку rate limiters
	go b.cleanupRateLimiters()

	return b, nil
}

// registerHandlers регистрирует все обработчики команд
func registerHandlers(router *Router, handlers *Handlers) {
	// Info commands
	router.RegisterHandler("status", handlers.HandleStatus)
	router.RegisterHandler("history", handlers.HandleHistory)
	router.RegisterHandler("config", handlers.HandleConfig)
	router.RegisterHandler("price", handlers.HandlePrice)
	router.RegisterHandler("portfolio", handlers.HandlePortfolio)
	router.RegisterHandler("help", handlers.HandleHelp)
	router.RegisterHandler("start", handlers.HandleHelp)

	// Trading commands
	router.RegisterHandler("buy", handlers.HandleBuy)
	router.RegisterHandler("sell", handlers.HandleSell)

	// Auto-Sell commands
	router.RegisterHandler("autosellon", handlers.HandleAutoSellOn)
	router.RegisterHandler("autosell_on", handlers.HandleAutoSellOn)
	router.RegisterHandler("autoselloff", handlers.HandleAutoSellOff)
	router.RegisterHandler("autosell_off", handlers.HandleAutoSellOff)
	router.RegisterHandler("autosell", handlers.HandleAutoSell)

	// Grid commands
	router.RegisterHandler("gridinit", handlers.HandleGridInit)
	router.RegisterHandler("grid_init", handlers.HandleGridInit)
	router.RegisterHandler("gridstatus", handlers.HandleGridStatus)
	router.RegisterHandler("grid_status", handlers.HandleGridStatus)
	router.RegisterAdminHandler("gridstop", handlers.HandleGridStop)
	router.RegisterAdminHandler("grid_stop", handlers.HandleGridStop)

	// Admin/Risk commands
	router.RegisterAdminHandler("risk", handlers.HandleRisk)
	router.RegisterAdminHandler("panicstop", handlers.HandlePanicStop)

	// AI commands
	router.RegisterHandler("analysis", func(ctx context.Context, args *CommandArgs) (string, error) {
		// Placeholder for analysis
		return "AI analysis not implemented in this handler", nil
	})
}

// Start запускает обработку сообщений
func (b *BotV2) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	b.SendMessage(0, "🤖 Crypto Trading Bot started!\nUse /help to see available commands.")

	for update := range updates {
		if update.Message != nil {
			go b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			go b.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

// handleMessage обрабатывает входящее сообщение
func (b *BotV2) handleMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userID := message.From.ID

	b.logger.Info("Received message from user %d (chat %d): %s", userID, chatID, message.Text)

	// Проверяем доступ
	if !b.authManager.IsAllowed(userID) {
		b.SendMessage(chatID, b.formatter.T("access_denied"))
		b.logger.Warn("Unauthorized access attempt from user ID: %d", userID)
		return
	}

	// Устанавливаем язык пользователя
	b.setUserLang(userID, chatID)

	// Обработка команд
	if message.IsCommand() {
		b.handleCommand(message)
		return
	}

	// Обработка текстовых сообщений через AI
	if b.aiClient != nil {
		b.handleAIMessage(message)
	} else {
		b.SendMessage(chatID, "Please use /help to see available commands.")
	}
}

// handleCommand обрабатывает команду
func (b *BotV2) handleCommand(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userID := message.From.ID

	ctx := context.Background()

	// Preview mode check
	if b.previewMode {
		b.SendMessage(chatID, fmt.Sprintf("🔍 PREVIEW MODE\nCommand: %s\nWould be executed, but preview mode is enabled.",
			message.Text))
		return
	}

	// Обрабатываем через router
	response, needsConfirmation, err := b.router.HandleCommand(ctx, userID, message.Text)

	if err != nil {
		b.logger.Error("Command error: %v", err)
	}

	// Если команда опасная и требует подтверждения
	if needsConfirmation && err == nil {
		// Создаем клавиатуру подтверждения
		msg := tgbotapi.NewMessage(chatID, b.formatter.T("confirm_action")+"\n\n"+response)
		msg.ReplyMarkup = b.router.MakeConfirmationKeyboard(message.Command(), message.CommandArguments())
		b.api.Send(msg)
	} else {
		b.SendMessage(chatID, response)
	}
}

// handleCallbackQuery обрабатывает callback от inline кнопок
func (b *BotV2) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID
	userID := query.From.ID

	ctx := context.Background()

	// Обрабатываем callback
	response, err := b.router.HandleCallback(ctx, userID, query.Data)
	if err != nil {
		b.logger.Error("Callback error: %v", err)
		response = b.formatter.FormatError(err)
	}

	// Отвечаем на callback
	callback := tgbotapi.NewCallback(query.ID, "")
	b.api.Send(callback)

	// Редактируем сообщение
	edit := tgbotapi.NewEditMessageText(chatID, query.Message.MessageID, response)
	b.api.Send(edit)
}

// handleAIMessage обрабатывает сообщение через AI
func (b *BotV2) handleAIMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	b.logger.Info("Processing AI message: %s", text)

	// Строим контекст
	context, err := b.buildContext()
	if err != nil {
		b.SendMessage(chatID, b.formatter.FormatError(err))
		return
	}

	// Отправляем в AI
	reply, actions, err := b.aiClient.ProcessMessage(text, context)
	if err != nil {
		b.SendMessage(chatID, b.formatter.FormatError(err))
		return
	}

	// Выполняем действия
	if len(actions) > 0 {
		for _, action := range actions {
			b.logger.Info("Executing AI action: %s", action.Type)

			// Preview mode check
			if b.previewMode {
				b.SendMessage(chatID, fmt.Sprintf("🔍 PREVIEW MODE\nAI Action: %s\nParams: %v\nWould be executed.",
					action.Type, action.Parameters))
				continue
			}

			actionResult, err := b.actionExecutor.ExecuteAction(action)
			if err != nil {
				b.logger.Error("Failed to execute AI action %s: %v", action.Type, err)
				b.SendMessage(chatID, b.formatter.FormatError(err))
			} else if actionResult != "" {
				b.SendMessage(chatID, actionResult)
			}
		}
	}

	// Отправляем ответ AI
	if reply != "" {
		b.SendMessage(chatID, reply)
	}
}

// buildContext строит контекст для AI
func (b *BotV2) buildContext() (string, error) {
	var sb strings.Builder

	// Получаем активные активы
	assets, err := b.handlers.storage.GetEnabledAssets()
	if err == nil && len(assets) > 0 {
		sb.WriteString("Active Assets:\n")
		for _, asset := range assets {
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", asset.Symbol, asset.StrategyType))
		}
		sb.WriteString("\n")
	}

	// Получаем балансы
	balances, err := b.handlers.storage.GetAllBalances()
	if err == nil && len(balances) > 0 {
		sb.WriteString("Balances:\n")
		for _, balance := range balances {
			price, _ := b.handlers.exchange.GetCurrentPrice(balance.Symbol)
			currentValue := balance.TotalQuantity * price
			pnl := currentValue - balance.TotalInvested + balance.RealizedProfit
			sb.WriteString(fmt.Sprintf("- %s: %.8f (Avg: $%.2f, Current: $%.2f, P&L: $%.2f)\n",
				balance.Symbol, balance.TotalQuantity, balance.AvgEntryPrice, price, pnl))
		}
	}

	return sb.String(), nil
}

// SendMessage отправляет сообщение пользователю
func (b *BotV2) SendMessage(chatID int64, text string) {
	if text == "" {
		return
	}

	// Если chatID = 0, отправляем всем админам
	if chatID == 0 {
		adminIDs := b.authManager.GetAdminIDs()
		for _, adminID := range adminIDs {
			b.sendToChat(adminID, text)
		}
		return
	}

	b.sendToChat(chatID, text)
}

// sendToChat отправляет сообщение в конкретный чат
func (b *BotV2) sendToChat(chatID int64, text string) {
	// Разбиваем длинные сообщения
	const maxLength = 4096
	messages := splitMessage(text, maxLength)

	for _, msg := range messages {
		message := tgbotapi.NewMessage(chatID, msg)
		message.ParseMode = "Markdown"
		if _, err := b.api.Send(message); err != nil {
			b.logger.Error("Failed to send telegram message to chat %d: %v", chatID, err)
		}
	}
}

// setUserLang определяет и устанавливает язык пользователя
func (b *BotV2) setUserLang(userID, chatID int64) {
	b.userLangsMu.Lock()
	defer b.userLangsMu.Unlock()

	// Если язык уже установлен, возвращаем
	if _, exists := b.userLangs[userID]; exists {
		return
	}

	// Пытаемся определить язык по language_code пользователя
	// В реальности нужно получить язык из user object
	// Пока ставим дефолтный
	lang := b.formatter.GetLang()
	b.userLangs[userID] = lang
	b.formatter.SetLang(lang)
}

// GetUserLang возвращает язык пользователя
func (b *BotV2) GetUserLang(userID int64) Lang {
	b.userLangsMu.RLock()
	defer b.userLangsMu.RUnlock()

	if lang, exists := b.userLangs[userID]; exists {
		return lang
	}

	return b.formatter.GetLang()
}

// SetUserLang устанавливает язык пользователя
func (b *BotV2) SetUserLang(userID int64, lang Lang) {
	b.userLangsMu.Lock()
	defer b.userLangsMu.Unlock()

	b.userLangs[userID] = lang
}

// cleanupRateLimiters периодически очищает старые rate limiters
func (b *BotV2) cleanupRateLimiters() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		b.authManager.CleanupRateLimiters()
		b.logger.Info("Cleaned up rate limiters")
	}
}

// Stop останавливает бота
func (b *BotV2) Stop() {
	b.logger.Info("Stopping Telegram bot...")
	b.api.StopReceivingUpdates()
}
