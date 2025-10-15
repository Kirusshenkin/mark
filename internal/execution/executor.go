package execution

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kirillm/dca-bot/internal/policy"
)

var (
	ErrKillSwitchActive  = errors.New("kill switch is active")
	ErrPolicyViolation   = errors.New("action rejected by policy engine")
	ErrSlippageTooHigh   = errors.New("slippage exceeds threshold")
	ErrPriceUnavailable  = errors.New("unable to get price from any source")
	ErrInsufficientFunds = errors.New("insufficient balance")
)

// Exchange интерфейс биржи
type Exchange interface {
	GetPrice(ctx context.Context, symbol string) (float64, error)
	GetBalance(ctx context.Context, asset string) (float64, error)
	PlaceMarketOrder(ctx context.Context, symbol, side string, quantity float64) (string, error)
}

// PolicyEngine интерфейс policy engine
type PolicyEngine interface {
	ValidateAction(ctx context.Context, action policy.ActionRequest) (*policy.ValidationResult, error)
}

// ExecutionRequest запрос на исполнение
type ExecutionRequest struct {
	Action   policy.ActionRequest
	Decision *AIDecision // Опционально: решение AI
}

// AIDecision представление решения AI
type AIDecision struct {
	ID         int64
	Regime     string
	Confidence float64
	Rationale  string
}

// ExecutionResult результат исполнения
type ExecutionResult struct {
	Success      bool
	OrderID      string
	ExecutedAt   time.Time
	ActualPrice  float64
	ActualAmount float64
	Slippage     float64
	Error        error
}

// Executor исполнитель торговых операций
type Executor struct {
	exchange      Exchange
	policyEngine  PolicyEngine
	priceFailover *PriceFailover
	killSwitch    *KillSwitch
	slippageGuard *SlippageGuard
}

// NewExecutor создает новый executor
func NewExecutor(
	exchange Exchange,
	policyEngine PolicyEngine,
	killSwitch *KillSwitch,
) *Executor {
	return &Executor{
		exchange:      exchange,
		policyEngine:  policyEngine,
		priceFailover: NewPriceFailover(exchange),
		killSwitch:    killSwitch,
		slippageGuard: NewSlippageGuard(1.0), // 1% default threshold
	}
}

// Execute выполняет торговую операцию
func (e *Executor) Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResult, error) {
	// 1. Проверка kill switch
	if e.killSwitch.IsActive() {
		return &ExecutionResult{
			Success:    false,
			ExecutedAt: time.Now(),
			Error:      ErrKillSwitchActive,
		}, ErrKillSwitchActive
	}

	// 2. Валидация через policy engine
	validation, err := e.policyEngine.ValidateAction(ctx, req.Action)
	if err != nil {
		return nil, fmt.Errorf("policy validation error: %w", err)
	}

	if !validation.Approved {
		return &ExecutionResult{
			Success:    false,
			ExecutedAt: time.Now(),
			Error:      ErrPolicyViolation,
		}, ErrPolicyViolation
	}

	// 3. Получение цены с failover
	symbol := req.Action.Symbol
	price, err := e.priceFailover.GetPrice(ctx, symbol)
	if err != nil {
		return &ExecutionResult{
			Success:    false,
			ExecutedAt: time.Now(),
			Error:      err,
		}, err
	}

	// 4. Проверка slippage (для buy orders)
	if req.Action.Type == "buy" || req.Action.Type == "set_dca" {
		expectedPrice, ok := req.Action.Parameters["expected_price"].(float64)
		if ok {
			if err := e.slippageGuard.CheckSlippage(price, expectedPrice); err != nil {
				return &ExecutionResult{
					Success:      false,
					ExecutedAt:   time.Now(),
					ActualPrice:  price,
					Error:        err,
				}, err
			}
		}
	}

	// 5. Исполнение ордера
	result, err := e.executeOrder(ctx, req.Action, price)
	if err != nil {
		return result, err
	}

	// 6. Логирование результата
	fmt.Printf("✅ Execution successful: %s %s @ $%.2f (OrderID: %s)\n",
		req.Action.Type, symbol, price, result.OrderID)

	return result, nil
}

// executeOrder исполняет конкретный ордер
func (e *Executor) executeOrder(ctx context.Context, action policy.ActionRequest, price float64) (*ExecutionResult, error) {
	symbol := action.Symbol

	switch action.Type {
	case "buy", "set_dca":
		return e.executeBuy(ctx, symbol, action.Parameters, price)

	case "sell":
		return e.executeSell(ctx, symbol, action.Parameters, price)

	case "set_grid":
		// Grid требует создания нескольких ордеров
		return e.executeGrid(ctx, symbol, action.Parameters)

	default:
		return &ExecutionResult{
			Success:    false,
			ExecutedAt: time.Now(),
			Error:      fmt.Errorf("unknown action type: %s", action.Type),
		}, fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// executeBuy выполняет покупку
func (e *Executor) executeBuy(ctx context.Context, symbol string, params map[string]interface{}, price float64) (*ExecutionResult, error) {
	// Получаем сумму в USDT
	var quoteAmount float64
	if val, ok := params["quote_usdt"].(float64); ok {
		quoteAmount = val
	} else if val, ok := params["quoteAmount"].(float64); ok {
		quoteAmount = val
	} else {
		return nil, fmt.Errorf("missing quote_usdt parameter")
	}

	// Проверяем баланс USDT
	usdtBalance, err := e.exchange.GetBalance(ctx, "USDT")
	if err != nil {
		return nil, fmt.Errorf("failed to get USDT balance: %w", err)
	}

	if usdtBalance < quoteAmount {
		return &ExecutionResult{
			Success:    false,
			ExecutedAt: time.Now(),
			Error:      ErrInsufficientFunds,
		}, ErrInsufficientFunds
	}

	// Рассчитываем количество базового актива
	quantity := quoteAmount / price

	// Размещаем market order
	orderID, err := e.exchange.PlaceMarketOrder(ctx, symbol, "BUY", quantity)
	if err != nil {
		return &ExecutionResult{
			Success:    false,
			ExecutedAt: time.Now(),
			Error:      err,
		}, err
	}

	return &ExecutionResult{
		Success:      true,
		OrderID:      orderID,
		ExecutedAt:   time.Now(),
		ActualPrice:  price,
		ActualAmount: quoteAmount,
		Slippage:     0.0, // TODO: calculate actual slippage
	}, nil
}

// executeSell выполняет продажу
func (e *Executor) executeSell(ctx context.Context, symbol string, params map[string]interface{}, price float64) (*ExecutionResult, error) {
	// Получаем процент для продажи
	sellPercent, ok := params["percent"].(float64)
	if !ok {
		sellPercent = 100.0 // Полная продажа по умолчанию
	}

	// Получаем баланс актива
	asset := symbol[:len(symbol)-4] // Удаляем "USDT" из конца
	balance, err := e.exchange.GetBalance(ctx, asset)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if balance <= 0 {
		return &ExecutionResult{
			Success:    false,
			ExecutedAt: time.Now(),
			Error:      ErrInsufficientFunds,
		}, ErrInsufficientFunds
	}

	// Рассчитываем количество для продажи
	quantity := balance * (sellPercent / 100.0)

	// Размещаем market order
	orderID, err := e.exchange.PlaceMarketOrder(ctx, symbol, "SELL", quantity)
	if err != nil {
		return &ExecutionResult{
			Success:    false,
			ExecutedAt: time.Now(),
			Error:      err,
		}, err
	}

	return &ExecutionResult{
		Success:      true,
		OrderID:      orderID,
		ExecutedAt:   time.Now(),
		ActualPrice:  price,
		ActualAmount: quantity * price,
		Slippage:     0.0,
	}, nil
}

// executeGrid создает Grid ордера
func (e *Executor) executeGrid(ctx context.Context, symbol string, params map[string]interface{}) (*ExecutionResult, error) {
	// Grid логика требует создания множества ордеров
	// Это более сложная операция, которую лучше делегировать GridStrategy

	return &ExecutionResult{
		Success:    false,
		ExecutedAt: time.Now(),
		Error:      fmt.Errorf("grid execution not yet implemented in executor"),
	}, fmt.Errorf("grid execution requires GridStrategy")
}

// SetSlippageThreshold устанавливает порог slippage
func (e *Executor) SetSlippageThreshold(thresholdPercent float64) {
	e.slippageGuard.SetThreshold(thresholdPercent)
}
