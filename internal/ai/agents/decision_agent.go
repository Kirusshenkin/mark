package agents

import (
	"context"
	"fmt"

	"github.com/kirillm/dca-bot/internal/ai"
	"github.com/kirillm/dca-bot/pkg/utils"
)

// DecisionAgent облачный агент для стратегических решений
type DecisionAgent struct {
	decisionClient *ai.DecisionClient
	mode           string // shadow, pilot, full
}

// NewDecisionAgent создает новый decision agent
func NewDecisionAgent(cloudAIURL, cloudAPIKey, cloudModel, mode string) *DecisionAgent {
	// Создаем клиент для облачной модели (Moonshot Kimi K2)
	baseClient := ai.NewAIClient("moonshot", cloudAPIKey, cloudAIURL, cloudModel)
	decisionClient := ai.NewDecisionClient(baseClient)

	return &DecisionAgent{
		decisionClient: decisionClient,
		mode:           mode,
	}
}

// RequestDecision запрашивает стратегическое решение
func (da *DecisionAgent) RequestDecision(ctx context.Context, req ai.DecisionRequest) (*ai.DecisionResponse, error) {
	utils.LogInfo(fmt.Sprintf("[DecisionAgent] Requesting decision in %s mode", da.mode))

	// Устанавливаем режим
	req.Mode = da.mode

	// Отправляем запрос к облачной модели
	decision, err := da.decisionClient.RequestDecision(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("decision agent failed: %w", err)
	}

	utils.LogInfo(fmt.Sprintf(
		"[DecisionAgent] Decision: regime=%s, confidence=%.2f, actions=%d",
		decision.Regime,
		decision.Confidence,
		len(decision.Actions),
	))

	// В режиме shadow только логируем, не возвращаем действия
	if da.mode == "shadow" {
		utils.LogInfo(fmt.Sprintf("[DecisionAgent] SHADOW MODE - actions not executed: %v", decision.Actions))
		decision.Actions = nil // Очищаем действия
	}

	// В режиме pilot ограничиваем параметры
	if da.mode == "pilot" {
		decision.Actions = da.applyPilotLimits(decision.Actions)
	}

	return decision, nil
}

// applyPilotLimits применяет ограничения для pilot режима
func (da *DecisionAgent) applyPilotLimits(actions []ai.Action) []ai.Action {
	limitedActions := make([]ai.Action, 0, len(actions))

	for _, action := range actions {
		limitedAction := action

		// Уменьшаем суммы на 50% в pilot режиме
		switch action.Type {
		case "set_dca":
			if quoteUSDT, ok := action.Parameters["quote_usdt"].(float64); ok {
				limitedAction.Parameters["quote_usdt"] = quoteUSDT * 0.5
			}

		case "set_grid":
			if orderSize, ok := action.Parameters["order_size_quote"].(float64); ok {
				limitedAction.Parameters["order_size_quote"] = orderSize * 0.5
			}

		case "rebalance":
			// В pilot режиме не выполняем rebalance
			utils.LogInfo("[DecisionAgent] PILOT MODE - skipping rebalance action")
			continue
		}

		limitedActions = append(limitedActions, limitedAction)
	}

	if len(limitedActions) < len(actions) {
		utils.LogInfo(fmt.Sprintf(
			"[DecisionAgent] PILOT MODE - limited %d actions to %d",
			len(actions),
			len(limitedActions),
		))
	}

	return limitedActions
}

// SetMode изменяет режим работы агента
func (da *DecisionAgent) SetMode(mode string) error {
	validModes := map[string]bool{
		"shadow": true,
		"pilot":  true,
		"full":   true,
	}

	if !validModes[mode] {
		return fmt.Errorf("invalid mode: %s (valid: shadow, pilot, full)", mode)
	}

	da.mode = mode
	utils.LogInfo(fmt.Sprintf("[DecisionAgent] Mode changed to: %s", mode))
	return nil
}

// GetMode возвращает текущий режим
func (da *DecisionAgent) GetMode() string {
	return da.mode
}

// FormatDecision форматирует решение для отображения пользователю
func (da *DecisionAgent) FormatDecision(decision *ai.DecisionResponse) string {
	if decision == nil {
		return "Нет решения"
	}

	regimeEmoji := map[string]string{
		"ACCUMULATE":   "📊",
		"TREND_FOLLOW": "📈",
		"RANGE_GRID":   "🎯",
		"DEFENSE":      "🛡️",
	}

	emoji := regimeEmoji[decision.Regime]
	if emoji == "" {
		emoji = "🤖"
	}

	result := fmt.Sprintf(`%s Стратегическое решение AI

Режим: %s
Уверенность: %.0f%%

Обоснование:
%s

Действия (%d):`,
		emoji,
		decision.Regime,
		decision.Confidence*100,
		decision.Rationale,
		len(decision.Actions),
	)

	if len(decision.Actions) == 0 {
		result += "\n└─ Нет действий (низкая уверенность или режим shadow)"
	} else {
		for i, action := range decision.Actions {
			prefix := "├─"
			if i == len(decision.Actions)-1 {
				prefix = "└─"
			}

			actionDesc := da.formatAction(action)
			result += fmt.Sprintf("\n%s %s", prefix, actionDesc)
		}
	}

	// Предупреждение для pilot/shadow режимов
	if da.mode == "shadow" {
		result += "\n\n⚠️ SHADOW MODE: решение не будет выполнено автоматически"
	} else if da.mode == "pilot" {
		result += "\n\n⚠️ PILOT MODE: параметры ограничены (50% от рекомендованных)"
	}

	return result
}

// formatAction форматирует действие для отображения
func (da *DecisionAgent) formatAction(action ai.Action) string {
	switch action.Type {
	case "set_dca":
		return fmt.Sprintf(
			"DCA для %s: $%.2f каждые %d минут",
			action.Symbol,
			action.Parameters["quote_usdt"],
			action.Parameters["interval_min"],
		)

	case "set_grid":
		return fmt.Sprintf(
			"Grid для %s: %v уровней, %.1f%% spacing, $%.2f за уровень",
			action.Symbol,
			action.Parameters["levels"],
			action.Parameters["spacing_pct"],
			action.Parameters["order_size_quote"],
		)

	case "set_autosell":
		return fmt.Sprintf(
			"Auto-Sell для %s: триггер %.0f%%, продажа %.0f%%",
			action.Symbol,
			action.Parameters["trigger_pct"],
			action.Parameters["sell_pct"],
		)

	case "pause_strategy":
		return fmt.Sprintf(
			"Пауза для %s (причина: %s)",
			action.Symbol,
			action.Parameters["reason"],
		)

	case "rebalance":
		return fmt.Sprintf("Ребалансировка портфеля: %v", action.Parameters["target_allocation"])

	default:
		return fmt.Sprintf("%s для %s", action.Type, action.Symbol)
	}
}
