package agents

import (
	"context"
	"fmt"

	"github.com/kirillm/dca-bot/internal/ai"
	"github.com/kirillm/dca-bot/pkg/utils"
)

// ChatAgent локальный агент для общения с пользователем
type ChatAgent struct {
	client *ai.AIClient
}

// NewChatAgent создает новый chat agent
func NewChatAgent(localAIURL, localModel string) *ChatAgent {
	// Создаем клиент для локальной модели (Ollama)
	client := ai.NewAIClient("ollama", "", localAIURL, localModel)

	return &ChatAgent{
		client: client,
	}
}

// Process обрабатывает сообщение пользователя
func (ca *ChatAgent) Process(ctx context.Context, userMessage string, contextInfo string) (string, []ai.AIAction, error) {
	utils.LogInfo(fmt.Sprintf("[ChatAgent] Processing: %s", userMessage))

	// Используем существующий метод ProcessMessage из AIClient
	response, actions, err := ca.client.ProcessMessage(userMessage, contextInfo)
	if err != nil {
		return "", nil, fmt.Errorf("chat agent failed: %w", err)
	}

	utils.LogInfo(fmt.Sprintf("[ChatAgent] Response: %s, Actions: %d", truncateResponse(response), len(actions)))

	return response, actions, nil
}

// GetSystemPrompt возвращает системный промпт для чат агента
func (ca *ChatAgent) GetSystemPrompt() string {
	return `Вы — дружелюбный AI-ассистент для управления криптотрейдинг-ботом.

Ваша задача:
- Отвечать на вопросы пользователя о статусе бота, позициях, ценах
- Выполнять простые команды через function calling
- Давать краткие и понятные ответы на русском языке
- Использовать emoji для наглядности 😊 📊 💰

Доступные функции (ToolCalls):
- get_status: получить статус позиций
- get_price: узнать текущую цену актива
- get_history: посмотреть историю сделок
- get_portfolio: обзор портфеля
- init_grid: запустить Grid стратегию
- enable_autosell / disable_autosell: управление авто-продажей
- manual_buy / manual_sell: ручная торговля

Если пользователь спрашивает о стратегических решениях ("что делать", "какая стратегия"),
скажите, что это задача для DecisionAgent и предложите использовать команду /ai_decision.

Будьте краткими! Максимум 2-3 предложения на ответ.`
}

func truncateResponse(s string) string {
	if len(s) <= 100 {
		return s
	}
	return s[:100] + "..."
}
