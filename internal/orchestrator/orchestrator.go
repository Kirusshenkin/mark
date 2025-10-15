package orchestrator

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kirillm/dca-bot/internal/ai"
	"github.com/kirillm/dca-bot/internal/execution"
	"github.com/kirillm/dca-bot/internal/policy"
)

// Mode режим работы orchestrator
type Mode string

const (
	ModeShadow Mode = "shadow" // AI решает, но не исполняет
	ModePilot  Mode = "pilot"  // Исполнение с консервативными лимитами
	ModeFull   Mode = "full"   // Полная автономия
)

// Orchestrator координатор автономной торговли
type Orchestrator struct {
	mode          Mode
	aiClient      *ai.DecisionClient
	policyEngine  *policy.Engine
	executor      *execution.Executor
	//portfolioMgr  PortfolioManager
	//infoService   NewsService

	ticker        *time.Ticker
	stopChan      chan struct{}
	isRunning     bool
}

// Config конфигурация orchestrator
type Config struct {
	Mode     Mode
	Interval time.Duration // Интервал принятия решений (15min default)
}

// New создает новый orchestrator
func New(
	mode Mode,
	interval time.Duration,
	aiClient *ai.DecisionClient,
	policyEngine *policy.Engine,
	executor *execution.Executor,
) *Orchestrator {
	return &Orchestrator{
		mode:         mode,
		aiClient:     aiClient,
		policyEngine: policyEngine,
		executor:     executor,
		ticker:       time.NewTicker(interval),
		stopChan:     make(chan struct{}),
		isRunning:    false,
	}
}

// Start запускает orchestrator
func (o *Orchestrator) Start(ctx context.Context) error {
	if o.isRunning {
		return fmt.Errorf("orchestrator already running")
	}

	o.isRunning = true
	log.Printf("🚀 Orchestrator started in %s mode (interval: %v)", o.mode, o.ticker.C)

	go o.run(ctx)

	return nil
}

// Stop останавливает orchestrator
func (o *Orchestrator) Stop() {
	if !o.isRunning {
		return
	}

	log.Println("🛑 Stopping orchestrator...")
	close(o.stopChan)
	o.ticker.Stop()
	o.isRunning = false
	log.Println("✅ Orchestrator stopped")
}

// run основной цикл orchestrator
func (o *Orchestrator) run(ctx context.Context) {
	// Первый цикл сразу после старта
	if err := o.runDecisionCycle(ctx); err != nil {
		log.Printf("❌ Initial decision cycle error: %v", err)
	}

	for {
		select {
		case <-o.ticker.C:
			if err := o.runDecisionCycle(ctx); err != nil {
				log.Printf("❌ Decision cycle error: %v", err)
				o.handleError(ctx, err)
			}

		case <-o.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// runDecisionCycle выполняет один цикл принятия решений
func (o *Orchestrator) runDecisionCycle(ctx context.Context) error {
	log.Printf("🧠 Starting decision cycle (mode: %s)", o.mode)

	// 1. Проверяем circuit breakers
	// TODO: implement circuit breaker check
	// if o.circuitBreaker.IsTriggered() {
	//     log.Println("⛔ Circuit breaker active, skipping cycle")
	//     return nil
	// }

	// 2. Собираем контекст для AI
	request := o.gatherContext(ctx)

	// 3. Запрашиваем решение у AI
	decision, err := o.aiClient.RequestDecision(ctx, request)
	if err != nil {
		return fmt.Errorf("AI decision request failed: %w", err)
	}

	log.Printf("🧠 AI Decision: regime=%s confidence=%.2f actions=%d",
		decision.Regime, decision.Confidence, len(decision.Actions))
	log.Printf("💡 Rationale: %s", decision.Rationale)

	// 4. Сохраняем решение в БД
	// TODO: save decision to database

	// 5. Валидируем и исполняем действия
	approvedActions := 0
	for i, action := range decision.Actions {
		log.Printf("📝 Action %d/%d: %s %s", i+1, len(decision.Actions), action.Type, action.Symbol)

		// Валидация через policy engine
		validation, err := o.policyEngine.ValidateAction(ctx, policy.ActionRequest{
			Type:       action.Type,
			Symbol:     action.Symbol,
			Parameters: action.Parameters,
		})

		if err != nil {
			log.Printf("❌ Policy validation error: %v", err)
			continue
		}

		if !validation.Approved {
			log.Printf("🚫 Action rejected by policy engine:")
			for _, v := range validation.Violations {
				log.Printf("   - %s: %s", v.Type, v.Message)
			}
			// TODO: save violation to DB
			continue
		}

		approvedActions++

		// Исполнение (только если не shadow mode)
		if o.mode != ModeShadow {
			if err := o.executeAction(ctx, action, decision); err != nil {
				log.Printf("❌ Execution failed: %v", err)
				o.handleExecutionError(ctx, action, err)
			} else {
				log.Printf("✅ Executed: %s %s", action.Type, action.Symbol)
			}
		} else {
			log.Printf("🔍 Shadow mode: would execute %s %s", action.Type, action.Symbol)
		}
	}

	log.Printf("📊 Cycle complete: %d/%d actions approved", approvedActions, len(decision.Actions))

	return nil
}

// gatherContext собирает контекст для AI решения
func (o *Orchestrator) gatherContext(ctx context.Context) ai.DecisionRequest {
	// TODO: Implement real data gathering
	// Для MVP возвращаем упрощенный контекст

	return ai.DecisionRequest{
		CurrentPortfolio: ai.PortfolioSnapshot{
			Assets: []ai.AssetStatus{
				{
					Symbol:        "BTCUSDT",
					Quantity:      0.01,
					AvgEntryPrice: 65000,
					CurrentPrice:  66000,
					InvestedUSDT:  650,
					CurrentValue:  660,
					PnL:           10,
					PnLPercent:    1.54,
				},
			},
			TotalValueUSDT:  660,
			TotalInvested:   650,
			TotalPnL:        10,
			TotalPnLPercent: 1.54,
		},
		MarketConditions: ai.MarketData{
			BTCPrice:       66000,
			BTCChange24h:   2.5,
			MarketSentiment: "neutral",
			Volatility:     1.5,
		},
		RecentNews: []ai.NewsSignal{},
		RiskLimits: ai.RiskLimits{
			MaxOrderUSDT:     o.getMaxOrderLimit(),
			MaxPositionUSDT:  1000,
			MaxTotalExposure: 3000,
			MaxDailyLoss:     100,
		},
		Mode: string(o.mode),
	}
}

// getMaxOrderLimit возвращает лимит в зависимости от режима
func (o *Orchestrator) getMaxOrderLimit() float64 {
	baseLimit := o.policyEngine.GetPolicy().MaxOrderUSDT

	switch o.mode {
	case ModeShadow:
		return baseLimit // Полный лимит (для логирования)
	case ModePilot:
		return baseLimit * 0.5 // 50% от лимита
	case ModeFull:
		return baseLimit // Полный лимит
	default:
		return baseLimit
	}
}

// executeAction исполняет действие
func (o *Orchestrator) executeAction(ctx context.Context, action ai.Action, decision *ai.DecisionResponse) error {
	// Преобразуем AI action в execution request
	execReq := execution.ExecutionRequest{
		Action: policy.ActionRequest{
			Type:       action.Type,
			Symbol:     action.Symbol,
			Parameters: action.Parameters,
		},
		Decision: &execution.AIDecision{
			Regime:     decision.Regime,
			Confidence: decision.Confidence,
			Rationale:  decision.Rationale,
		},
	}

	result, err := o.executor.Execute(ctx, execReq)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("execution failed: %v", result.Error)
	}

	return nil
}

// handleError обрабатывает ошибки цикла
func (o *Orchestrator) handleError(ctx context.Context, err error) {
	log.Printf("⚠️ Error in decision cycle: %v", err)
	// TODO: Implement error handling
	// - Уведомление оператора
	// - Логирование в БД
	// - Возможный fallback на консервативный режим
}

// handleExecutionError обрабатывает ошибки исполнения
func (o *Orchestrator) handleExecutionError(ctx context.Context, action ai.Action, err error) {
	log.Printf("⚠️ Execution error for %s %s: %v", action.Type, action.Symbol, err)
	// TODO: Implement execution error handling
	// - Retry logic
	// - Уведомление оператора
	// - Сохранение в БД
}

// SetMode изменяет режим работы
func (o *Orchestrator) SetMode(mode Mode) {
	log.Printf("🔄 Switching mode: %s → %s", o.mode, mode)
	o.mode = mode
}

// GetMode возвращает текущий режим
func (o *Orchestrator) GetMode() Mode {
	return o.mode
}

// IsRunning проверяет запущен ли orchestrator
func (o *Orchestrator) IsRunning() bool {
	return o.isRunning
}
