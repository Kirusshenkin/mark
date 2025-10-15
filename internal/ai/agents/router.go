package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kirillm/dca-bot/internal/ai"
	"github.com/kirillm/dca-bot/pkg/utils"
)

// RequestType тип запроса для роутинга
type RequestType string

const (
	RequestTypeChat     RequestType = "chat"     // Общение, простые команды
	RequestTypeDecision RequestType = "decision" // Стратегические решения
	RequestTypeAnalysis RequestType = "analysis" // Анализ рынка
)

// AgentRouter маршрутизатор запросов к разным AI агентам
type AgentRouter struct {
	chatAgent     *ChatAgent
	decisionAgent *DecisionAgent
	analysisAgent *AnalysisAgent

	// Метрики
	metrics *RouterMetrics
}

// RouterMetrics метрики роутера
type RouterMetrics struct {
	ChatCount      int
	DecisionCount  int
	AnalysisCount  int
	TotalRequests  int
	AvgLatency     time.Duration
	CloudRequests  int
	LocalRequests  int
}

// NewAgentRouter создает новый роутер
func NewAgentRouter(chatAgent *ChatAgent, decisionAgent *DecisionAgent, analysisAgent *AnalysisAgent) *AgentRouter {
	return &AgentRouter{
		chatAgent:     chatAgent,
		decisionAgent: decisionAgent,
		analysisAgent: analysisAgent,
		metrics:       &RouterMetrics{},
	}
}

// Process обрабатывает запрос, автоматически выбирая нужный агент
func (r *AgentRouter) Process(ctx context.Context, userMessage string, context string) (string, []ai.AIAction, error) {
	startTime := time.Now()
	defer func() {
		r.metrics.AvgLatency = (r.metrics.AvgLatency*time.Duration(r.metrics.TotalRequests) + time.Since(startTime)) / time.Duration(r.metrics.TotalRequests+1)
		r.metrics.TotalRequests++
	}()

	// Определяем тип запроса
	requestType := r.Route(userMessage)

	utils.LogInfo(fmt.Sprintf("[AI Router] Request type: %s | Message: %s", requestType, truncate(userMessage, 50)))

	// Направляем к нужному агенту
	switch requestType {
	case RequestTypeChat:
		r.metrics.ChatCount++
		r.metrics.LocalRequests++
		return r.chatAgent.Process(ctx, userMessage, context)

	case RequestTypeDecision:
		r.metrics.DecisionCount++
		r.metrics.CloudRequests++
		// DecisionAgent требует специальной структуры, поэтому вызывается напрямую
		// через DecisionClient.RequestDecision() с полным контекстом
		return "", nil, fmt.Errorf("decision requests должны вызываться через DecisionAgent.RequestDecision() с полным контекстом")

	case RequestTypeAnalysis:
		r.metrics.AnalysisCount++
		r.metrics.LocalRequests++
		return r.analysisAgent.Process(ctx, userMessage, context)

	default:
		return "", nil, fmt.Errorf("unknown request type: %s", requestType)
	}
}

// Route определяет тип запроса на основе содержимого
func (r *AgentRouter) Route(message string) RequestType {
	messageLower := strings.ToLower(message)

	// Ключевые слова для стратегических решений
	decisionKeywords := []string{
		"стратегия", "strategy",
		"решение", "decision",
		"что делать", "what should i do",
		"стоит ли", "should i",
		"рекомендуешь", "recommend",
		"посоветуй", "advise",
		"автономн", "autonomous",
		"режим", "regime",
	}

	for _, keyword := range decisionKeywords {
		if strings.Contains(messageLower, keyword) {
			return RequestTypeDecision
		}
	}

	// Ключевые слова для анализа
	analysisKeywords := []string{
		"анализ", "analysis", "analyze",
		"прогноз", "forecast", "predict",
		"оцен", "assess", "evaluate",
		"тренд", "trend",
		"техническ", "technical",
		"индикатор", "indicator",
		"настроение", "sentiment",
	}

	for _, keyword := range analysisKeywords {
		if strings.Contains(messageLower, keyword) {
			return RequestTypeAnalysis
		}
	}

	// По умолчанию - чат
	return RequestTypeChat
}

// GetMetrics возвращает метрики роутера
func (r *AgentRouter) GetMetrics() *RouterMetrics {
	return r.metrics
}

// FormatMetrics форматирует метрики для отображения
func (r *AgentRouter) FormatMetrics() string {
	m := r.metrics
	if m.TotalRequests == 0 {
		return "Нет запросов к AI Router"
	}

	cloudPercent := float64(m.CloudRequests) / float64(m.TotalRequests) * 100
	localPercent := float64(m.LocalRequests) / float64(m.TotalRequests) * 100

	return fmt.Sprintf(`🤖 AI Router Метрики:

Всего запросов: %d
├─ Chat: %d (%.1f%%)
├─ Decision: %d (%.1f%%)
└─ Analysis: %d (%.1f%%)

Модели:
├─ Локальная: %d (%.1f%%)
└─ Облачная: %d (%.1f%%)

Средняя латентность: %dms`,
		m.TotalRequests,
		m.ChatCount, float64(m.ChatCount)/float64(m.TotalRequests)*100,
		m.DecisionCount, float64(m.DecisionCount)/float64(m.TotalRequests)*100,
		m.AnalysisCount, float64(m.AnalysisCount)/float64(m.TotalRequests)*100,
		m.LocalRequests, localPercent,
		m.CloudRequests, cloudPercent,
		m.AvgLatency.Milliseconds(),
	)
}

// truncate обрезает строку до заданной длины
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
