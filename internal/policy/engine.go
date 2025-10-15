package policy

import (
	"context"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Storage интерфейс для работы с БД
type Storage interface {
	GetAllBalances(ctx context.Context) ([]Balance, error)
	GetRecentTrades(ctx context.Context, since time.Time) ([]Trade, error)
	SavePolicyViolation(ctx context.Context, violation *PolicyViolation) error
	SaveCircuitBreakerEvent(ctx context.Context, event *CircuitBreakerEvent) error
}

// Balance для расчета экспозиции
type Balance struct {
	Symbol         string
	TotalInvested  float64
	UnrealizedPnL  float64
}

// Trade для анализа частоты и убытков
type Trade struct {
	Symbol    string
	Side      string
	Amount    float64
	CreatedAt time.Time
}

// PolicyViolation событие нарушения
type PolicyViolation struct {
	ActionType     string
	ViolationType  string
	LimitName      string
	LimitValue     float64
	AttemptedValue float64
	Severity       string
}

// CircuitBreakerEvent событие триггера
type CircuitBreakerEvent struct {
	Reason      string
	Details     string
	PausedUntil time.Time
}

// Engine движок policy-based risk management
type Engine struct {
	policy      *Policy
	storage     Storage
	metrics     *RiskMetrics
	lastCheck   time.Time
}

// NewEngine создает новый policy engine
func NewEngine(policyPath string, storage Storage) (*Engine, error) {
	policy, err := loadPolicy(policyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}

	return &Engine{
		policy:    policy,
		storage:   storage,
		metrics:   &RiskMetrics{},
		lastCheck: time.Now(),
	}, nil
}

// loadPolicy загружает policy из YAML
func loadPolicy(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config struct {
		RiskProfiles map[string]Policy `yaml:"risk_profiles"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// По умолчанию используем moderate
	profileName := os.Getenv("POLICY_PROFILE")
	if profileName == "" {
		profileName = "moderate"
	}

	policy, ok := config.RiskProfiles[profileName]
	if !ok {
		return nil, fmt.Errorf("policy profile %s not found", profileName)
	}

	policy.ProfileName = profileName
	return &policy, nil
}

// ValidateAction проверяет действие на соответствие политике
func (e *Engine) ValidateAction(ctx context.Context, action ActionRequest) (*ValidationResult, error) {
	// Обновляем метрики
	if err := e.updateMetrics(ctx); err != nil {
		return nil, fmt.Errorf("failed to update metrics: %w", err)
	}

	result := &ValidationResult{
		Approved:   true,
		RiskScore:  0.0,
		Violations: []Violation{},
		CheckedAt:  time.Now(),
	}

	// Проверяем circuit breakers
	if triggered := e.checkCircuitBreakers(ctx); triggered != nil {
		result.Approved = false
		result.Violations = append(result.Violations, Violation{
			Type:     "circuit_breaker",
			Severity: "critical",
			Message:  fmt.Sprintf("Circuit breaker triggered: %s", triggered.Reason),
		})
		return result, nil
	}

	// Валидация по типу действия
	switch action.Type {
	case "set_dca", "buy":
		e.validateBuyAction(action, result)
	case "sell":
		e.validateSellAction(action, result)
	case "set_grid":
		e.validateGridAction(action, result)
	default:
		// Для остальных действий используем базовую валидацию
	}

	// Проверка частоты трейдов
	if e.metrics.DailyTradeCount >= e.policy.TradesPerHour*24 {
		result.Violations = append(result.Violations, Violation{
			Type:           "trade_frequency",
			LimitName:      "trades_per_day",
			LimitValue:     float64(e.policy.TradesPerHour * 24),
			AttemptedValue: float64(e.metrics.DailyTradeCount + 1),
			Severity:       "warning",
			Message:        "Daily trade limit exceeded",
		})
	}

	// Проверка дневных убытков
	if e.metrics.DailyLossUSDT >= e.policy.MaxDailyLossUSDT {
		result.Violations = append(result.Violations, Violation{
			Type:           "daily_loss",
			LimitName:      "max_daily_loss_usdt",
			LimitValue:     e.policy.MaxDailyLossUSDT,
			AttemptedValue: e.metrics.DailyLossUSDT,
			Severity:       "critical",
			Message:        "Daily loss limit reached",
		})
		result.Approved = false
	}

	// Если есть critical нарушения - отклоняем
	for _, v := range result.Violations {
		if v.Severity == "critical" {
			result.Approved = false

			// Сохраняем violation в БД
			if err := e.storage.SavePolicyViolation(ctx, &PolicyViolation{
				ViolationType:  v.Type,
				LimitName:      v.LimitName,
				LimitValue:     v.LimitValue,
				AttemptedValue: v.AttemptedValue,
				Severity:       v.Severity,
			}); err != nil {
				// Логируем но не фейлим
				fmt.Printf("Failed to save policy violation: %v\n", err)
			}
		}
	}

	// Расчет risk score (0.0 - 1.0)
	result.RiskScore = e.calculateRiskScore()

	return result, nil
}

// validateBuyAction проверяет действия покупки
func (e *Engine) validateBuyAction(action ActionRequest, result *ValidationResult) {
	// Получаем сумму из параметров
	var amount float64
	if val, ok := action.Parameters["quote_usdt"].(float64); ok {
		amount = val
	} else if val, ok := action.Parameters["quoteAmount"].(float64); ok {
		amount = val
	}

	// Проверка размера ордера
	if amount > e.policy.MaxOrderUSDT {
		result.Violations = append(result.Violations, Violation{
			Type:           "order_size",
			LimitName:      "max_order_usdt",
			LimitValue:     e.policy.MaxOrderUSDT,
			AttemptedValue: amount,
			Severity:       "critical",
			Message:        fmt.Sprintf("Order size %.2f exceeds limit %.2f", amount, e.policy.MaxOrderUSDT),
		})
	}

	// Проверка общей экспозиции
	newExposure := e.metrics.TotalExposureUSDT + amount
	if newExposure > e.policy.MaxTotalExposure {
		result.Violations = append(result.Violations, Violation{
			Type:           "total_exposure",
			LimitName:      "max_total_exposure",
			LimitValue:     e.policy.MaxTotalExposure,
			AttemptedValue: newExposure,
			Severity:       "critical",
			Message:        fmt.Sprintf("Total exposure %.2f would exceed limit %.2f", newExposure, e.policy.MaxTotalExposure),
		})
	}
}

// validateSellAction проверяет действия продажи
func (e *Engine) validateSellAction(action ActionRequest, result *ValidationResult) {
	// Продажи обычно разрешены, но можем добавить ограничения
	// Например, минимальный интервал между продажами
}

// validateGridAction проверяет Grid стратегию
func (e *Engine) validateGridAction(action ActionRequest, result *ValidationResult) {
	// Grid может создать несколько ордеров
	levels, _ := action.Parameters["levels"].(float64)
	orderSize, _ := action.Parameters["order_size_quote"].(float64)

	totalGridCapital := levels * orderSize

	if totalGridCapital > e.policy.MaxPositionUSDT {
		result.Violations = append(result.Violations, Violation{
			Type:           "position_size",
			LimitName:      "max_position_size_usd",
			LimitValue:     e.policy.MaxPositionUSDT,
			AttemptedValue: totalGridCapital,
			Severity:       "warning",
			Message:        fmt.Sprintf("Grid total capital %.2f exceeds position limit %.2f", totalGridCapital, e.policy.MaxPositionUSDT),
		})
	}
}

// checkCircuitBreakers проверяет все предохранители
func (e *Engine) checkCircuitBreakers(ctx context.Context) *CircuitBreakerEvent {
	for _, cb := range e.policy.CircuitBreakers {
		switch cb.Type {
		case "drawdown":
			if e.metrics.CurrentDrawdown >= cb.Threshold {
				return &CircuitBreakerEvent{
					Reason:      fmt.Sprintf("drawdown %.2f%% >= %.2f%%", e.metrics.CurrentDrawdown, cb.Threshold),
					PausedUntil: time.Now().Add(1 * time.Hour),
				}
			}
		case "daily_loss":
			if e.metrics.DailyLossUSDT >= cb.Threshold {
				return &CircuitBreakerEvent{
					Reason:      fmt.Sprintf("daily loss $%.2f >= $%.2f", e.metrics.DailyLossUSDT, cb.Threshold),
					PausedUntil: time.Now().Add(24 * time.Hour),
				}
			}
		case "volatility":
			if e.metrics.VolatilityPct >= cb.Threshold {
				return &CircuitBreakerEvent{
					Reason:      fmt.Sprintf("volatility %.2f%% >= %.2f%%", e.metrics.VolatilityPct, cb.Threshold),
					PausedUntil: time.Now().Add(30 * time.Minute),
				}
			}
		}
	}
	return nil
}

// updateMetrics обновляет текущие метрики риска
func (e *Engine) updateMetrics(ctx context.Context) error {
	// Получаем балансы для расчета экспозиции
	balances, err := e.storage.GetAllBalances(ctx)
	if err != nil {
		return err
	}

	totalExposure := 0.0
	totalPnL := 0.0
	for _, b := range balances {
		totalExposure += b.TotalInvested
		totalPnL += b.UnrealizedPnL
	}
	e.metrics.TotalExposureUSDT = totalExposure

	// Расчет drawdown
	if totalExposure > 0 {
		e.metrics.CurrentDrawdown = -(totalPnL / totalExposure) * 100
		if e.metrics.CurrentDrawdown < 0 {
			e.metrics.CurrentDrawdown = 0
		}
	}

	// Получаем сделки за последние 24 часа
	since := time.Now().Add(-24 * time.Hour)
	trades, err := e.storage.GetRecentTrades(ctx, since)
	if err != nil {
		return err
	}

	e.metrics.DailyTradeCount = len(trades)

	// Расчет дневных убытков
	dailyLoss := 0.0
	for _, t := range trades {
		if t.Side == "SELL" {
			// TODO: правильный расчет P&L требует сопоставления с покупками
			// Пока упрощенная логика
		}
	}
	e.metrics.DailyLossUSDT = dailyLoss
	e.metrics.LastUpdated = time.Now()

	return nil
}

// calculateRiskScore вычисляет общий риск-скор (0.0 = безопасно, 1.0 = максимум)
func (e *Engine) calculateRiskScore() float64 {
	score := 0.0

	// Экспозиция относительно лимита
	if e.policy.MaxTotalExposure > 0 {
		score += (e.metrics.TotalExposureUSDT / e.policy.MaxTotalExposure) * 0.4
	}

	// Drawdown
	score += (e.metrics.CurrentDrawdown / 100.0) * 0.3

	// Дневные убытки
	if e.policy.MaxDailyLossUSDT > 0 {
		score += (e.metrics.DailyLossUSDT / e.policy.MaxDailyLossUSDT) * 0.3
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// GetPolicy возвращает текущую политику
func (e *Engine) GetPolicy() *Policy {
	return e.policy
}

// GetMetrics возвращает текущие метрики
func (e *Engine) GetMetrics() *RiskMetrics {
	return e.metrics
}
