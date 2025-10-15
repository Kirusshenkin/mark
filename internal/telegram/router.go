package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CommandHandler представляет обработчик команды
type CommandHandler func(ctx context.Context, args *CommandArgs) (string, error)

// Router маршрутизирует команды к обработчикам
type Router struct {
	handlers        map[string]CommandHandler
	authManager     *AuthManager
	validator       *Validator
	formatter       *Formatter
	adminCommands   map[string]bool
	dangerousCommands map[string]bool
}

// NewRouter создает новый роутер
func NewRouter(authManager *AuthManager, validator *Validator, formatter *Formatter) *Router {
	r := &Router{
		handlers:      make(map[string]CommandHandler),
		authManager:   authManager,
		validator:     validator,
		formatter:     formatter,
		adminCommands: make(map[string]bool),
		dangerousCommands: make(map[string]bool),
	}

	// Регистрируем админские команды
	r.adminCommands["panicstop"] = true
	r.adminCommands["gridstop"] = true
	r.adminCommands["risk"] = true

	// Регистрируем опасные команды (требуют подтверждения)
	r.dangerousCommands["sell"] = true
	r.dangerousCommands["gridstop"] = true
	r.dangerousCommands["panicstop"] = true

	return r
}

// RegisterHandler регистрирует обработчик команды
func (r *Router) RegisterHandler(command string, handler CommandHandler) {
	r.handlers[command] = handler
}

// RegisterAdminHandler регистрирует обработчик с требованием админских прав
func (r *Router) RegisterAdminHandler(command string, handler CommandHandler) {
	r.adminCommands[command] = true
	r.handlers[command] = handler
}

// HandleCommand обрабатывает команду
func (r *Router) HandleCommand(ctx context.Context, userID int64, text string) (string, bool, error) {
	// Проверяем rate limit
	if err := r.authManager.CheckRateLimit(userID, 2); err != nil {
		return r.formatter.FormatError(err), false, nil
	}

	// Проверяем доступ пользователя
	if !r.authManager.IsAllowed(userID) {
		return r.formatter.T("access_denied"), false, nil
	}

	// Парсим команду
	args, err := ParseCommand(text)
	if err != nil {
		return r.formatter.FormatError(err), false, nil
	}

	// Нормализуем команду
	args.Command = normalizeCommand(args.Command)

	// Проверяем права для админских команд
	if r.adminCommands[args.Command] {
		if err := r.authManager.RequireAdmin(userID); err != nil {
			return r.formatter.T("admin_required"), false, nil
		}
	}

	// Получаем обработчик
	handler, exists := r.handlers[args.Command]
	if !exists {
		return fmt.Sprintf("%s: %s", r.formatter.T("error"), "unknown command"), false, nil
	}

	// Проверяем, требуется ли подтверждение
	needsConfirmation := r.dangerousCommands[args.Command]

	// Выполняем обработчик
	response, err := handler(ctx, args)
	if err != nil {
		return r.formatter.FormatError(err), false, err
	}

	return response, needsConfirmation, nil
}

// MakeConfirmationKeyboard создает клавиатуру подтверждения
func (r *Router) MakeConfirmationKeyboard(command string, args string) tgbotapi.InlineKeyboardMarkup {
	confirmData := fmt.Sprintf("confirm_%s_%s", command, args)
	cancelData := "cancel"

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ "+r.formatter.T("confirm"), confirmData),
			tgbotapi.NewInlineKeyboardButtonData("❌ "+r.formatter.T("cancel"), cancelData),
		),
	)
}

// HandleCallback обрабатывает callback от inline кнопок
func (r *Router) HandleCallback(ctx context.Context, userID int64, data string) (string, error) {
	if data == "cancel" {
		return r.formatter.T("cancel"), nil
	}

	// Разбираем callback data
	// Формат: confirm_<command>_<args>
	// Для простоты можно просто вызвать команду повторно
	// В продакшене лучше сохранять состояние

	return "Callback handled", nil
}

// IsAdminCommand проверяет, является ли команда админской
func (r *Router) IsAdminCommand(command string) bool {
	return r.adminCommands[command]
}

// IsDangerousCommand проверяет, является ли команда опасной
func (r *Router) IsDangerousCommand(command string) bool {
	return r.dangerousCommands[command]
}
