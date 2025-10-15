package execution

import (
	"fmt"
	"math"
)

// SlippageGuard защита от чрезмерного проскальзывания
type SlippageGuard struct {
	thresholdPercent float64
}

// NewSlippageGuard создает новый slippage guard
func NewSlippageGuard(thresholdPercent float64) *SlippageGuard {
	return &SlippageGuard{
		thresholdPercent: thresholdPercent,
	}
}

// CheckSlippage проверяет приемлемость проскальзывания
func (sg *SlippageGuard) CheckSlippage(actualPrice, expectedPrice float64) error {
	if expectedPrice <= 0 {
		return fmt.Errorf("invalid expected price: %.2f", expectedPrice)
	}

	// Рассчитываем проскальзывание в процентах
	slippage := math.Abs((actualPrice - expectedPrice) / expectedPrice * 100.0)

	if slippage > sg.thresholdPercent {
		return fmt.Errorf("%w: %.2f%% (threshold: %.2f%%)", ErrSlippageTooHigh, slippage, sg.thresholdPercent)
	}

	return nil
}

// CalculateSlippage вычисляет процент проскальзывания
func (sg *SlippageGuard) CalculateSlippage(actualPrice, expectedPrice float64) float64 {
	if expectedPrice <= 0 {
		return 0.0
	}

	return math.Abs((actualPrice - expectedPrice) / expectedPrice * 100.0)
}

// SetThreshold устанавливает новый порог
func (sg *SlippageGuard) SetThreshold(thresholdPercent float64) {
	sg.thresholdPercent = thresholdPercent
}

// GetThreshold возвращает текущий порог
func (sg *SlippageGuard) GetThreshold() float64 {
	return sg.thresholdPercent
}
