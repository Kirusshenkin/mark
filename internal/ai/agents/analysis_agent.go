package agents

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/kirillm/dca-bot/internal/ai"
	"github.com/kirillm/dca-bot/pkg/utils"
)

// AnalysisAgent локальный агент для рыночного анализа
type AnalysisAgent struct {
	client *ai.AIClient
}

// NewAnalysisAgent создает новый analysis agent
func NewAnalysisAgent(localAIURL, localModel string) *AnalysisAgent {
	client := ai.NewAIClient("ollama", "", localAIURL, localModel)

	return &AnalysisAgent{
		client: client,
	}
}

// Process обрабатывает запрос на анализ
func (aa *AnalysisAgent) Process(ctx context.Context, userMessage string, contextInfo string) (string, []ai.AIAction, error) {
	utils.LogInfo(fmt.Sprintf("[AnalysisAgent] Processing: %s", userMessage))

	// Парсим запрос, чтобы понять, что анализировать
	symbol, currentPrice, avgEntry := aa.parseAnalysisRequest(userMessage, contextInfo)

	// Если удалось распарсить цены, используем GetMarketAnalysis
	if symbol != "" && currentPrice > 0 {
		analysis, err := aa.client.GetMarketAnalysis(symbol, currentPrice, avgEntry)
		if err != nil {
			return "", nil, fmt.Errorf("analysis failed: %w", err)
		}

		utils.LogInfo(fmt.Sprintf("[AnalysisAgent] Analysis completed for %s", symbol))
		return analysis, nil, nil
	}

	// Иначе используем обычный ProcessMessage
	response, actions, err := aa.client.ProcessMessage(userMessage, contextInfo)
	if err != nil {
		return "", nil, fmt.Errorf("analysis agent failed: %w", err)
	}

	return response, actions, nil
}

// parseAnalysisRequest пытается извлечь symbol и цены из запроса/контекста
func (aa *AnalysisAgent) parseAnalysisRequest(userMessage, contextInfo string) (string, float64, float64) {
	// Ищем упоминания символов
	symbols := []string{"BTC", "ETH", "SOL", "BNB", "XRP"}
	var symbol string

	messageLower := strings.ToLower(userMessage + " " + contextInfo)

	for _, sym := range symbols {
		if strings.Contains(messageLower, strings.ToLower(sym)) {
			symbol = sym + "USDT"
			break
		}
	}

	if symbol == "" {
		return "", 0, 0
	}

	// Простой парсинг цен из контекста (если есть)
	// Формат: "Current price: 67234.5"
	var currentPrice, avgEntry float64

	if idx := strings.Index(contextInfo, "Current price:"); idx != -1 {
		priceStr := extractNumber(contextInfo[idx:])
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			currentPrice = price
		}
	}

	if idx := strings.Index(contextInfo, "Average entry:"); idx != -1 {
		priceStr := extractNumber(contextInfo[idx:])
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			avgEntry = price
		}
	}

	return symbol, currentPrice, avgEntry
}

// extractNumber извлекает первое число из строки
func extractNumber(s string) string {
	var result strings.Builder
	foundDigit := false

	for _, ch := range s {
		if ch >= '0' && ch <= '9' || ch == '.' {
			result.WriteRune(ch)
			foundDigit = true
		} else if foundDigit {
			break
		}
	}

	return result.String()
}

// GetSystemPrompt возвращает системный промпт для analysis agent
func (aa *AnalysisAgent) GetSystemPrompt() string {
	return `Вы — криптотрейдер-аналитик с опытом технического анализа.

Ваша задача:
- Анализировать текущую рыночную ситуацию
- Определять тренды и уровни поддержки/сопротивления
- Оценивать риски и возможности
- Давать краткие и объективные рекомендации

Формат ответа:
1. Текущая ситуация (1 предложение)
2. Технический анализ (1-2 предложения)
3. Рекомендация (1 предложение)

Будьте объективны! Не давайте гарантий. Используйте фразы:
- "Возможно...", "Вероятно...", "Рекомендуется..."
- "Следует рассмотреть...", "При условии..."

НЕ давайте финансовых советов! Только информационный анализ.
Максимум 3-4 предложения.`
}
