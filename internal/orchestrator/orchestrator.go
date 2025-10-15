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

// Storage интерфейс для сохранения данных
type Storage interface {
	SaveAIDecision(decision *ai.DecisionResponse, mode string, approved bool) error
	SavePolicyViolation(violation *policy.Violation) error
}

// DataProvider интерфейс для получения данных портфеля
type DataProvider interface {
	GetAllBalances() ([]Balance, error)
	GetEnabledAssets() ([]Asset, error)
	GetPrice(symbol string) (float64, error)
}

// Balance упрощенная структура баланса
type Balance struct {
	Symbol        string
	TotalQuantity float64
	AvgEntryPrice float64
	TotalInvested float64
	RealizedProfit float64
	UnrealizedPnL float64
}

// Asset упрощенная структура актива
type Asset struct {
	Symbol string
	Enabled bool
}

// Orchestrator координатор автономной торговли
type Orchestrator struct {
	mode          Mode
	aiClient      *ai.DecisionClient
	policyEngine  *policy.Engine
	executor      *execution.Executor
	storage       Storage
	dataProvider  DataProvider
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
	storage Storage,
	dataProvider DataProvider,
) *Orchestrator {
	return &Orchestrator{
		mode:         mode,
		aiClient:     aiClient,
		policyEngine: policyEngine,
		executor:     executor,
		storage:      storage,
		dataProvider: dataProvider,
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
	if triggered := o.policyEngine.CheckCircuitBreakers(ctx); triggered != nil {
		log.Printf("⛔ Circuit breaker triggered: %s", triggered.Reason)
		log.Printf("   Paused until: %s", triggered.PausedUntil.Format("2006-01-02 15:04:05"))

		// Проверяем, не истекло ли время паузы
		if time.Now().Before(triggered.PausedUntil) {
			log.Println("   Skipping decision cycle due to active circuit breaker")
			return nil
		}
		log.Println("   Circuit breaker pause expired, resuming operations")
	}

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
	if o.storage != nil {
		// Decision will be marked as approved after validation
		if err := o.storage.SaveAIDecision(decision, string(o.mode), true); err != nil {
			log.Printf("⚠️  Failed to save AI decision to database: %v", err)
			// Continue execution даже если сохранение не удалось
		} else {
			log.Printf("💾 AI decision saved to database")
		}
	}

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

				// Сохраняем нарушение в БД
				if o.storage != nil {
					if err := o.storage.SavePolicyViolation(&v); err != nil {
						log.Printf("⚠️  Failed to save policy violation to database: %v", err)
					}
				}
			}
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
	var assets []ai.AssetStatus
	var totalValueUSDT, totalInvested, totalPnL float64

	// Если dataProvider не установлен, возвращаем минимальный контекст
	if o.dataProvider == nil {
		log.Printf("⚠️ DataProvider not set, using minimal context")
		return o.buildMinimalContext()
	}

	// Получаем балансы из БД
	balances, err := o.dataProvider.GetAllBalances()
	if err != nil {
		log.Printf("⚠️ Failed to get balances: %v, using minimal context", err)
		return o.buildMinimalContext()
	}

	// Обрабатываем каждый баланс
	for _, bal := range balances {
		// Пропускаем активы с нулевым количеством
		if bal.TotalQuantity <= 0 {
			continue
		}

		// Получаем текущую цену актива
		currentPrice, err := o.dataProvider.GetPrice(bal.Symbol)
		if err != nil {
			log.Printf("⚠️ Failed to get price for %s: %v, skipping", bal.Symbol, err)
			continue
		}

		// Рассчитываем текущую стоимость и P&L
		currentValue := bal.TotalQuantity * currentPrice
		unrealizedPnL := currentValue - bal.TotalInvested
		pnlPercent := 0.0
		if bal.TotalInvested > 0 {
			pnlPercent = (unrealizedPnL / bal.TotalInvested) * 100
		}

		// Добавляем актив в снапшот
		assets = append(assets, ai.AssetStatus{
			Symbol:        bal.Symbol,
			Quantity:      bal.TotalQuantity,
			AvgEntryPrice: bal.AvgEntryPrice,
			CurrentPrice:  currentPrice,
			InvestedUSDT:  bal.TotalInvested,
			CurrentValue:  currentValue,
			PnL:           unrealizedPnL + bal.RealizedProfit, // Учитываем и реализованную прибыль
			PnLPercent:    pnlPercent,
		})

		// Накапливаем общие значения
		totalValueUSDT += currentValue
		totalInvested += bal.TotalInvested
		totalPnL += unrealizedPnL + bal.RealizedProfit
	}

	// Получаем BTC цену для market conditions
	btcPrice, err := o.dataProvider.GetPrice("BTCUSDT")
	if err != nil {
		log.Printf("⚠️ Failed to get BTC price: %v", err)
		btcPrice = 0
	}

	// Рассчитываем общий процент P&L
	totalPnLPercent := 0.0
	if totalInvested > 0 {
		totalPnLPercent = (totalPnL / totalInvested) * 100
	}

	// Логируем собранные данные
	log.Printf("📊 Portfolio context: %d assets, total value: $%.2f, P&L: $%.2f (%.2f%%)",
		len(assets), totalValueUSDT, totalPnL, totalPnLPercent)

	return ai.DecisionRequest{
		CurrentPortfolio: ai.PortfolioSnapshot{
			Assets:          assets,
			TotalValueUSDT:  totalValueUSDT,
			TotalInvested:   totalInvested,
			TotalPnL:        totalPnL,
			TotalPnLPercent: totalPnLPercent,
		},
		MarketConditions: ai.MarketData{
			BTCPrice:        btcPrice,
			BTCChange24h:    0, // TODO: рассчитать изменение за 24ч
			MarketSentiment: "neutral",
			Volatility:      0, // TODO: рассчитать волатильность
		},
		RecentNews: []ai.NewsSignal{}, // TODO: подключить NewsSignalRepository
		RiskLimits: ai.RiskLimits{
			MaxOrderUSDT:     o.getMaxOrderLimit(),
			MaxPositionUSDT:  o.policyEngine.GetPolicy().MaxPositionUSDT,
			MaxTotalExposure: o.policyEngine.GetPolicy().MaxTotalExposure,
			MaxDailyLoss:     o.policyEngine.GetPolicy().MaxDailyLossUSDT,
		},
		Mode: string(o.mode),
	}
}

// buildMinimalContext создает минимальный контекст для fallback
func (o *Orchestrator) buildMinimalContext() ai.DecisionRequest {
	return ai.DecisionRequest{
		CurrentPortfolio: ai.PortfolioSnapshot{
			Assets:          []ai.AssetStatus{},
			TotalValueUSDT:  0,
			TotalInvested:   0,
			TotalPnL:        0,
			TotalPnLPercent: 0,
		},
		MarketConditions: ai.MarketData{
			BTCPrice:        0,
			BTCChange24h:    0,
			MarketSentiment: "unknown",
			Volatility:      0,
		},
		RecentNews: []ai.NewsSignal{},
		RiskLimits: ai.RiskLimits{
			MaxOrderUSDT:     o.getMaxOrderLimit(),
			MaxPositionUSDT:  o.policyEngine.GetPolicy().MaxPositionUSDT,
			MaxTotalExposure: o.policyEngine.GetPolicy().MaxTotalExposure,
			MaxDailyLoss:     o.policyEngine.GetPolicy().MaxDailyLossUSDT,
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
