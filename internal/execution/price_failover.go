package execution

import (
	"context"
	"fmt"
	"time"
)

// PriceSource источник цен
type PriceSource interface {
	GetPrice(ctx context.Context, symbol string) (float64, error)
}

// PriceFailover failover механизм для получения цен
type PriceFailover struct {
	primarySource   PriceSource
	fallbackSources []PriceSource
	cache           map[string]cachedPrice
}

type cachedPrice struct {
	price     float64
	timestamp time.Time
}

// NewPriceFailover создает новый price failover
func NewPriceFailover(primarySource PriceSource) *PriceFailover {
	return &PriceFailover{
		primarySource:   primarySource,
		fallbackSources: []PriceSource{},
		cache:           make(map[string]cachedPrice),
	}
}

// AddFallbackSource добавляет запасной источник цен
func (pf *PriceFailover) AddFallbackSource(source PriceSource) {
	pf.fallbackSources = append(pf.fallbackSources, source)
}

// GetPrice получает цену с failover
func (pf *PriceFailover) GetPrice(ctx context.Context, symbol string) (float64, error) {
	// Пробуем основной источник
	price, err := pf.primarySource.GetPrice(ctx, symbol)
	if err == nil {
		// Кешируем успешный результат
		pf.cache[symbol] = cachedPrice{
			price:     price,
			timestamp: time.Now(),
		}
		return price, nil
	}

	// Основной источник недоступен, пробуем fallback
	for i, source := range pf.fallbackSources {
		price, err := source.GetPrice(ctx, symbol)
		if err == nil {
			fmt.Printf("⚠️ Using fallback source #%d for %s price\n", i+1, symbol)
			pf.cache[symbol] = cachedPrice{
				price:     price,
				timestamp: time.Now(),
			}
			return price, nil
		}
	}

	// Все источники недоступны, используем кеш если есть
	if cached, ok := pf.cache[symbol]; ok {
		age := time.Since(cached.timestamp)
		if age < 5*time.Minute { // Кеш валиден 5 минут
			fmt.Printf("⚠️ Using cached price for %s (age: %v)\n", symbol, age)
			return cached.price, nil
		}
	}

	return 0, ErrPriceUnavailable
}
