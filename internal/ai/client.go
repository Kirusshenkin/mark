package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AIClient struct {
	provider  string
	apiKey    string
	baseURL   string
	model     string
	client    *http.Client
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
}

// Tool представляет функцию, которую AI может вызвать
type Tool struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition описание функции для AI
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ToolCall вызов функции от AI
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type AIAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

func NewAIClient(provider, apiKey, baseURL, model string) *AIClient {
	return &AIClient{
		provider: provider,
		apiKey:   apiKey,
		baseURL:  baseURL,
		model:    model,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

// ProcessMessage обрабатывает сообщение пользователя и возвращает ответ и действия
func (a *AIClient) ProcessMessage(userMessage string, context string) (string, []AIAction, error) {
	systemPrompt := a.buildSystemPrompt(context)
	tools := a.buildTools()

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	response, toolCalls, err := a.chatWithTools(messages, tools)
	if err != nil {
		return "", nil, err
	}

	// Преобразуем tool_calls в AIAction
	actions := a.parseToolCalls(toolCalls)

	return response, actions, nil
}

// buildSystemPrompt создает системный промпт для AI (сильно сокращенный)
func (a *AIClient) buildSystemPrompt(context string) string {
	return fmt.Sprintf(`Вы — AI-ассистент для управления криптотрейдинг-ботом.

Текущий контекст:
%s

Используйте доступные функции для выполнения команд пользователя.
Если пользователь просто общается - отвечайте без вызова функций.
Всегда давайте краткие и понятные ответы.`, context)
}

// buildTools создает список доступных инструментов для AI
func (a *AIClient) buildTools() []Tool {
	return []Tool{
		// Информационные команды
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "get_status",
				Description: "Получить текущий статус позиций и стратегий",
			},
		},
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "get_history",
				Description: "Получить историю последних сделок",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Количество сделок (по умолчанию 10)",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "get_price",
				Description: "Получить текущую цену актива",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"symbol": map[string]interface{}{
							"type":        "string",
							"description": "Торговая пара (например BTCUSDT)",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "get_portfolio",
				Description: "Показать обзор портфеля со всеми активами",
			},
		},
		// Grid стратегия
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "init_grid",
				Description: "Инициализировать Grid-стратегию для актива",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"symbol": map[string]interface{}{
							"type":        "string",
							"description": "Торговая пара (например ETHUSDT)",
						},
					},
					"required": []string{"symbol"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "get_grid_status",
				Description: "Получить статус Grid-стратегии",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"symbol": map[string]interface{}{
							"type":        "string",
							"description": "Торговая пара (например ETHUSDT)",
						},
					},
				},
			},
		},
		// Управление Auto-Sell
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "enable_autosell",
				Description: "Включить Auto-Sell стратегию",
			},
		},
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "disable_autosell",
				Description: "Выключить Auto-Sell стратегию",
			},
		},
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "update_autosell_trigger",
				Description: "Обновить триггер Auto-Sell (процент прибыли)",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"percent": map[string]interface{}{
							"type":        "number",
							"description": "Процент прибыли для триггера",
						},
					},
					"required": []string{"percent"},
				},
			},
		},
		// Ручное управление
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "manual_buy",
				Description: "Выполнить ручную покупку DCA",
			},
		},
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "manual_sell",
				Description: "Выполнить ручную продажу части позиции",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"percent": map[string]interface{}{
							"type":        "number",
							"description": "Процент позиции для продажи (1-100)",
						},
					},
					"required": []string{"percent"},
				},
			},
		},
		// Настройка DCA
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "update_dca_amount",
				Description: "Обновить сумму DCA покупки",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"amount": map[string]interface{}{
							"type":        "number",
							"description": "Новая сумма в USDT",
						},
					},
					"required": []string{"amount"},
				},
			},
		},
	}
}

// chatWithTools отправляет запрос к AI API с поддержкой tools
func (a *AIClient) chatWithTools(messages []Message, tools []Tool) (string, []ToolCall, error) {
	requestBody := ChatRequest{
		Model:    a.model,
		Messages: messages,
		Tools:    tools,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", nil, err
	}

	// Build endpoint, avoiding double /v1 if baseURL already contains it
	endpoint := a.baseURL
	if !bytes.HasSuffix([]byte(endpoint), []byte("/v1")) {
		endpoint += "/v1"
	}
	endpoint += "/chat/completions"

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.apiKey))

	resp, err := a.client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("AI API error: %s", string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", nil, err
	}

	if len(chatResp.Choices) == 0 {
		return "", nil, fmt.Errorf("no response from AI")
	}

	message := chatResp.Choices[0].Message
	return message.Content, message.ToolCalls, nil
}

// parseToolCalls конвертирует tool_calls в AIAction
func (a *AIClient) parseToolCalls(toolCalls []ToolCall) []AIAction {
	actions := make([]AIAction, 0, len(toolCalls))

	for _, tc := range toolCalls {
		action := AIAction{
			Type: tc.Function.Name,
		}

		// Парсим JSON аргументы
		if tc.Function.Arguments != "" {
			var params map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err == nil {
				action.Parameters = params
			}
		}

		actions = append(actions, action)
	}

	return actions
}

// chat отправляет запрос к AI API без tools (для GetMarketAnalysis)
func (a *AIClient) chat(messages []Message) (string, error) {
	requestBody := ChatRequest{
		Model:    a.model,
		Messages: messages,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	// Build endpoint, avoiding double /v1 if baseURL already contains it
	endpoint := a.baseURL
	if !bytes.HasSuffix([]byte(endpoint), []byte("/v1")) {
		endpoint += "/v1"
	}
	endpoint += "/chat/completions"

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.apiKey))

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API error: %s", string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// GetMarketAnalysis получает анализ рынка от AI
func (a *AIClient) GetMarketAnalysis(symbol string, currentPrice float64, avgEntry float64) (string, error) {
	profitPercent := ((currentPrice - avgEntry) / avgEntry) * 100

	prompt := fmt.Sprintf(
		"Проанализируй текущую ситуацию: %s торгуется по цене %.2f USDT. "+
			"Средняя цена входа: %.2f USDT. Текущая прибыль: %.2f%%. "+
			"Дай краткую рекомендацию (2-3 предложения).",
		symbol, currentPrice, avgEntry, profitPercent,
	)

	messages := []Message{
		{
			Role:    "system",
			Content: "Вы — криптотрейдер-аналитик. Давайте краткие и объективные рекомендации.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	return a.chat(messages)
}
