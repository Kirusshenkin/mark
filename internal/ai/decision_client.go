package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DecisionClient клиент для получения стратегических решений от AI
type DecisionClient struct {
	baseClient *AIClient
}

// NewDecisionClient создает новый decision client
func NewDecisionClient(baseClient *AIClient) *DecisionClient {
	return &DecisionClient{
		baseClient: baseClient,
	}
}

// DecisionRequest запрос на принятие решения
type DecisionRequest struct {
	CurrentPortfolio PortfolioSnapshot `json:"current_portfolio"`
	MarketConditions MarketData        `json:"market_conditions"`
	RecentNews       []NewsSignal      `json:"recent_news"`
	RiskLimits       RiskLimits        `json:"risk_limits"`
	Mode             string            `json:"mode"` // shadow, pilot, full
}

// PortfolioSnapshot снимок портфеля
type PortfolioSnapshot struct {
	Assets          []AssetStatus `json:"assets"`
	TotalValueUSDT  float64       `json:"total_value_usdt"`
	TotalInvested   float64       `json:"total_invested"`
	TotalPnL        float64       `json:"total_pnl"`
	TotalPnLPercent float64       `json:"total_pnl_percent"`
}

// AssetStatus статус актива
type AssetStatus struct {
	Symbol        string  `json:"symbol"`
	Quantity      float64 `json:"quantity"`
	AvgEntryPrice float64 `json:"avg_entry_price"`
	CurrentPrice  float64 `json:"current_price"`
	InvestedUSDT  float64 `json:"invested_usdt"`
	CurrentValue  float64 `json:"current_value"`
	PnL           float64 `json:"pnl"`
	PnLPercent    float64 `json:"pnl_percent"`
}

// MarketData рыночные данные
type MarketData struct {
	BTCPrice       float64 `json:"btc_price"`
	BTCChange24h   float64 `json:"btc_change_24h"`
	MarketSentiment string  `json:"market_sentiment"` // bullish, bearish, neutral
	Volatility     float64 `json:"volatility"`        // % per hour
}

// NewsSignal новостной сигнал
type NewsSignal struct {
	Headline       string   `json:"headline"`
	Sentiment      string   `json:"sentiment"` // positive, negative, neutral
	SentimentScore float64  `json:"sentiment_score"`
	Topics         []string `json:"topics"`
	Symbols        []string `json:"symbols"`
}

// RiskLimits лимиты рисков
type RiskLimits struct {
	MaxOrderUSDT     float64 `json:"max_order_usdt"`
	MaxPositionUSDT  float64 `json:"max_position_usdt"`
	MaxTotalExposure float64 `json:"max_total_exposure"`
	MaxDailyLoss     float64 `json:"max_daily_loss"`
}

// DecisionResponse ответ AI с решением
type DecisionResponse struct {
	Regime     string   `json:"regime"`     // ACCUMULATE, TREND_FOLLOW, RANGE_GRID, DEFENSE
	Confidence float64  `json:"confidence"` // 0.0 - 1.0
	Rationale  string   `json:"rationale"`
	Actions    []Action `json:"actions"`
}

// Action действие для выполнения
type Action struct {
	Type       string                 `json:"type"` // set_dca, set_grid, set_autosell, rebalance, pause_strategy
	Symbol     string                 `json:"symbol"`
	Parameters map[string]interface{} `json:"parameters"`
}

// RequestDecision запрашивает стратегическое решение у AI
func (dc *DecisionClient) RequestDecision(ctx context.Context, req DecisionRequest) (*DecisionResponse, error) {
	// Строим промпт для AI
	prompt := dc.buildDecisionPrompt(req)

	// Отправляем запрос к AI
	messages := []Message{
		{Role: "system", Content: GetDecisionSystemPrompt()},
		{Role: "user", Content: prompt},
	}

	response, err := dc.baseClient.chat(messages)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	// Парсим JSON ответ
	var decision DecisionResponse
	if err := json.Unmarshal([]byte(response), &decision); err != nil {
		// Попытка извлечь JSON из markdown code block
		if cleanJSON := extractJSON(response); cleanJSON != "" {
			if err := json.Unmarshal([]byte(cleanJSON), &decision); err != nil {
				return nil, fmt.Errorf("failed to parse AI response: %w\nRaw response: %s", err, response)
			}
		} else {
			return nil, fmt.Errorf("failed to parse AI response: %w\nRaw response: %s", err, response)
		}
	}

	// Валидация ответа
	if err := dc.validateDecision(&decision); err != nil {
		return nil, fmt.Errorf("invalid decision: %w", err)
	}

	return &decision, nil
}

// buildDecisionPrompt строит промпт для принятия решения
func (dc *DecisionClient) buildDecisionPrompt(req DecisionRequest) string {
	portfolioJSON, _ := json.MarshalIndent(req.CurrentPortfolio, "", "  ")
	marketJSON, _ := json.MarshalIndent(req.MarketConditions, "", "  ")
	newsJSON, _ := json.MarshalIndent(req.RecentNews, "", "  ")
	limitsJSON, _ := json.MarshalIndent(req.RiskLimits, "", "  ")

	return fmt.Sprintf(`Analyze the current situation and provide a strategic trading decision.

Current Context:
- Mode: %s
- Time: %s

Portfolio:
%s

Market Conditions:
%s

Recent News (last 1 hour):
%s

Risk Limits:
%s

Provide your decision in JSON format (no markdown, pure JSON):
{
  "regime": "ACCUMULATE|TREND_FOLLOW|RANGE_GRID|DEFENSE",
  "confidence": 0.0-1.0,
  "rationale": "Brief explanation of your decision",
  "actions": [
    {
      "type": "set_dca|set_grid|set_autosell|rebalance|pause_strategy",
      "symbol": "BTCUSDT",
      "parameters": {"quote_usdt": 100, "interval_min": 720}
    }
  ]
}

Rules:
1. NEVER exceed risk limits
2. If confidence < 0.6, return empty actions array
3. Max 3 actions per decision
4. Consider news sentiment
5. Adjust strategy based on market conditions`,
		req.Mode,
		time.Now().Format(time.RFC3339),
		string(portfolioJSON),
		string(marketJSON),
		string(newsJSON),
		string(limitsJSON),
	)
}

// validateDecision проверяет корректность решения
func (dc *DecisionClient) validateDecision(decision *DecisionResponse) error {
	// Проверка режима
	validRegimes := map[string]bool{
		"ACCUMULATE":   true,
		"TREND_FOLLOW": true,
		"RANGE_GRID":   true,
		"DEFENSE":      true,
	}

	if !validRegimes[decision.Regime] {
		return fmt.Errorf("invalid regime: %s", decision.Regime)
	}

	// Проверка confidence
	if decision.Confidence < 0.0 || decision.Confidence > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0, got: %.2f", decision.Confidence)
	}

	// Проверка количества действий
	if len(decision.Actions) > 3 {
		return fmt.Errorf("too many actions: %d (max 3)", len(decision.Actions))
	}

	// Проверка типов действий
	validActionTypes := map[string]bool{
		"set_dca":        true,
		"set_grid":       true,
		"set_autosell":   true,
		"rebalance":      true,
		"pause_strategy": true,
	}

	for i, action := range decision.Actions {
		if !validActionTypes[action.Type] {
			return fmt.Errorf("invalid action type at index %d: %s", i, action.Type)
		}
	}

	return nil
}

// extractJSON извлекает JSON из markdown code block
func extractJSON(text string) string {
	// Простой парсер для ```json...```
	start := -1
	end := -1

	for i := 0; i < len(text)-2; i++ {
		if text[i:i+3] == "```" {
			if start == -1 {
				start = i + 3
				// Пропускаем "json" если есть
				if i+7 < len(text) && text[i+3:i+7] == "json" {
					start = i + 7
				}
				// Пропускаем перенос строки
				if start < len(text) && text[start] == '\n' {
					start++
				}
			} else {
				end = i
				break
			}
		}
	}

	if start > 0 && end > start {
		return text[start:end]
	}

	return text
}
