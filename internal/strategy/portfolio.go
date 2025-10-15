package strategy

import (
	"fmt"

	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
	"github.com/kirillm/dca-bot/pkg/utils"
)

// PortfolioManager управляет портфелем и распределением капитала
type PortfolioManager struct {
	storage  *storage.PostgresStorage
	exchange *exchange.BybitClient
}

func NewPortfolioManager(storage *storage.PostgresStorage, exchange *exchange.BybitClient) *PortfolioManager {
	return &PortfolioManager{
		storage:  storage,
		exchange: exchange,
	}
}

// AllocateCapital распределяет капитал между активами
func (p *PortfolioManager) AllocateCapital(totalCapital float64) error {
	utils.LogInfo(fmt.Sprintf("Распределение капитала: %.2f USDT", totalCapital))

	// Получаем все активные активы
	assets, err := p.storage.GetEnabledAssets()
	if err != nil {
		return fmt.Errorf("не удалось получить активы: %w", err)
	}

	if len(assets) == 0 {
		return fmt.Errorf("нет активных активов для распределения капитала")
	}

	// Рассчитываем сумму всех allocated_capital
	totalAllocated := 0.0
	for _, asset := range assets {
		totalAllocated += asset.AllocatedCapital
	}

	if totalAllocated == 0 {
		// Если капитал не распределен, делим поровну
		capitalPerAsset := totalCapital / float64(len(assets))
		for i := range assets {
			assets[i].AllocatedCapital = capitalPerAsset
			if err := p.storage.CreateOrUpdateAsset(&assets[i]); err != nil {
				utils.LogError(fmt.Sprintf("Не удалось обновить актив %s: %v", assets[i].Symbol, err))
			}
		}
		utils.LogInfo(fmt.Sprintf("Капитал распределен равномерно: %.2f USDT на актив", capitalPerAsset))
	} else {
		// Распределяем пропорционально заданным значениям
		for i := range assets {
			proportion := assets[i].AllocatedCapital / totalAllocated
			assets[i].AllocatedCapital = totalCapital * proportion
			if err := p.storage.CreateOrUpdateAsset(&assets[i]); err != nil {
				utils.LogError(fmt.Sprintf("Не удалось обновить актив %s: %v", assets[i].Symbol, err))
			}
		}
		utils.LogInfo("Капитал распределен пропорционально настройкам")
	}

	return nil
}

// GetPortfolioSummary возвращает сводку по портфелю
func (p *PortfolioManager) GetPortfolioSummary() (map[string]interface{}, error) {
	assets, err := p.storage.GetEnabledAssets()
	if err != nil {
		return nil, err
	}

	balances, err := p.storage.GetAllBalances()
	if err != nil {
		return nil, err
	}

	totalInvested := 0.0
	totalCurrentValue := 0.0
	totalRealizedProfit := 0.0
	totalUnrealizedPnL := 0.0

	assetDetails := []map[string]interface{}{}

	for _, balance := range balances {
		currentPrice, err := p.exchange.GetCurrentPrice(balance.Symbol)
		if err != nil {
			utils.LogError(fmt.Sprintf("Не удалось получить цену для %s: %v", balance.Symbol, err))
			continue
		}

		currentValue := balance.TotalQuantity * currentPrice
		unrealizedPnL := 0.0
		if balance.TotalQuantity > 0 {
			costBasis := balance.TotalQuantity * balance.AvgEntryPrice
			unrealizedPnL = currentValue - costBasis
		}

		totalPnL := balance.RealizedProfit + unrealizedPnL
		returnPercent := 0.0
		if balance.TotalInvested > 0 {
			returnPercent = (totalPnL / balance.TotalInvested) * 100
		}

		// Обновляем unrealized PnL в базе
		balance.UnrealizedPnL = unrealizedPnL
		if err := p.storage.UpdateBalance(&balance); err != nil {
			utils.LogError(fmt.Sprintf("Не удалось обновить баланс %s: %v", balance.Symbol, err))
		}

		totalInvested += balance.TotalInvested
		totalCurrentValue += currentValue
		totalRealizedProfit += balance.RealizedProfit
		totalUnrealizedPnL += unrealizedPnL

		// Найти соответствующий актив
		var assetInfo *storage.Asset
		for _, asset := range assets {
			if asset.Symbol == balance.Symbol {
				assetInfo = &asset
				break
			}
		}

		detail := map[string]interface{}{
			"symbol":            balance.Symbol,
			"quantity":          balance.TotalQuantity,
			"avg_entry_price":   balance.AvgEntryPrice,
			"current_price":     currentPrice,
			"invested":          balance.TotalInvested,
			"current_value":     currentValue,
			"realized_profit":   balance.RealizedProfit,
			"unrealized_pnl":    unrealizedPnL,
			"total_pnl":         totalPnL,
			"return_percent":    returnPercent,
		}

		if assetInfo != nil {
			detail["strategy_type"] = assetInfo.StrategyType
			detail["allocated_capital"] = assetInfo.AllocatedCapital
		}

		assetDetails = append(assetDetails, detail)
	}

	totalPnL := totalRealizedProfit + totalUnrealizedPnL
	totalReturnPercent := 0.0
	if totalInvested > 0 {
		totalReturnPercent = (totalPnL / totalInvested) * 100
	}

	summary := map[string]interface{}{
		"total_assets":         len(assets),
		"active_positions":     len(balances),
		"total_invested":       totalInvested,
		"total_current_value":  totalCurrentValue,
		"total_realized_profit": totalRealizedProfit,
		"total_unrealized_pnl": totalUnrealizedPnL,
		"total_pnl":            totalPnL,
		"total_return_percent": totalReturnPercent,
		"assets":               assetDetails,
	}

	return summary, nil
}

// RebalancePortfolio перебалансирует портфель
func (p *PortfolioManager) RebalancePortfolio() error {
	utils.LogInfo("Начало ребалансировки портфеля")

	summary, err := p.GetPortfolioSummary()
	if err != nil {
		return err
	}

	totalCurrentValue := summary["total_current_value"].(float64)
	assets, err := p.storage.GetEnabledAssets()
	if err != nil {
		return err
	}

	// Рассчитываем целевое распределение
	totalAllocated := 0.0
	for _, asset := range assets {
		totalAllocated += asset.AllocatedCapital
	}

	if totalAllocated == 0 {
		return fmt.Errorf("капитал не распределен между активами")
	}

	// Для каждого актива проверяем отклонение от целевого распределения
	for _, asset := range assets {
		balance, err := p.storage.GetBalance(asset.Symbol)
		if err != nil {
			utils.LogError(fmt.Sprintf("Не удалось получить баланс для %s: %v", asset.Symbol, err))
			continue
		}

		currentPrice, err := p.exchange.GetCurrentPrice(asset.Symbol)
		if err != nil {
			utils.LogError(fmt.Sprintf("Не удалось получить цену для %s: %v", asset.Symbol, err))
			continue
		}

		currentValue := balance.TotalQuantity * currentPrice
		targetProportion := asset.AllocatedCapital / totalAllocated
		targetValue := totalCurrentValue * targetProportion

		deviation := ((currentValue - targetValue) / targetValue) * 100

		utils.LogInfo(fmt.Sprintf("%s: текущее %.2f, целевое %.2f, отклонение %.2f%%",
			asset.Symbol, currentValue, targetValue, deviation))

		// Если отклонение больше 20%, ребалансируем
		if deviation > 20 || deviation < -20 {
			utils.LogInfo(fmt.Sprintf("Требуется ребалансировка для %s", asset.Symbol))
			// Здесь можно добавить логику купли/продажи для ребалансировки
			// Но это требует более сложной логики и проверок
		}
	}

	return nil
}

// GetAssetAllocation возвращает распределение портфеля
func (p *PortfolioManager) GetAssetAllocation() ([]map[string]interface{}, error) {
	assets, err := p.storage.GetEnabledAssets()
	if err != nil {
		return nil, err
	}

	balances, err := p.storage.GetAllBalances()
	if err != nil {
		return nil, err
	}

	totalValue := 0.0
	allocations := []map[string]interface{}{}

	for _, balance := range balances {
		currentPrice, err := p.exchange.GetCurrentPrice(balance.Symbol)
		if err != nil {
			continue
		}
		totalValue += balance.TotalQuantity * currentPrice
	}

	for _, asset := range assets {
		balance, err := p.storage.GetBalance(asset.Symbol)
		if err != nil {
			continue
		}

		currentPrice, err := p.exchange.GetCurrentPrice(asset.Symbol)
		if err != nil {
			continue
		}

		currentValue := balance.TotalQuantity * currentPrice
		percentage := 0.0
		if totalValue > 0 {
			percentage = (currentValue / totalValue) * 100
		}

		allocation := map[string]interface{}{
			"symbol":            asset.Symbol,
			"strategy_type":     asset.StrategyType,
			"allocated_capital": asset.AllocatedCapital,
			"current_value":     currentValue,
			"percentage":        percentage,
			"quantity":          balance.TotalQuantity,
			"avg_price":         balance.AvgEntryPrice,
			"current_price":     currentPrice,
		}

		allocations = append(allocations, allocation)
	}

	return allocations, nil
}

// CalculateDailyPnL рассчитывает и сохраняет дневной PnL snapshot
func (p *PortfolioManager) CalculateDailyPnL() error {
	utils.LogInfo("Создание дневного PnL snapshot")

	balances, err := p.storage.GetAllBalances()
	if err != nil {
		return err
	}

	for _, balance := range balances {
		currentPrice, err := p.exchange.GetCurrentPrice(balance.Symbol)
		if err != nil {
			utils.LogError(fmt.Sprintf("Не удалось получить цену для %s: %v", balance.Symbol, err))
			continue
		}

		currentValue := balance.TotalQuantity * currentPrice
		unrealizedPnL := 0.0
		if balance.TotalQuantity > 0 {
			costBasis := balance.TotalQuantity * balance.AvgEntryPrice
			unrealizedPnL = currentValue - costBasis
		}

		totalPnL := balance.RealizedProfit + unrealizedPnL
		returnPercent := 0.0
		if balance.TotalInvested > 0 {
			returnPercent = (totalPnL / balance.TotalInvested) * 100
		}

		pnlSnapshot := &storage.PnLHistory{
			Symbol:        balance.Symbol,
			RealizedPnL:   balance.RealizedProfit,
			UnrealizedPnL: unrealizedPnL,
			TotalPnL:      totalPnL,
			TotalInvested: balance.TotalInvested,
			CurrentValue:  currentValue,
			ReturnPercent: returnPercent,
			SnapshotType:  "DAILY",
		}

		if err := p.storage.SavePnLSnapshot(pnlSnapshot); err != nil {
			utils.LogError(fmt.Sprintf("Не удалось сохранить PnL snapshot для %s: %v", balance.Symbol, err))
		}
	}

	utils.LogInfo("Дневной PnL snapshot создан")
	return nil
}

// GetPerformanceMetrics возвращает метрики производительности
func (p *PortfolioManager) GetPerformanceMetrics(symbol string, days int) (map[string]interface{}, error) {
	// Получаем историю PnL
	history, err := p.storage.GetPnLHistory(symbol, "DAILY", days)
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return map[string]interface{}{
			"symbol":        symbol,
			"data_points":   0,
			"message":       "Недостаточно данных для анализа",
		}, nil
	}

	// Рассчитываем метрики
	totalReturn := 0.0
	maxDrawdown := 0.0
	peak := 0.0

	for _, snapshot := range history {
		if snapshot.ReturnPercent > peak {
			peak = snapshot.ReturnPercent
		}
		drawdown := peak - snapshot.ReturnPercent
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	lastSnapshot := history[0] // Последний (самый новый)
	totalReturn = lastSnapshot.ReturnPercent

	metrics := map[string]interface{}{
		"symbol":              symbol,
		"period_days":         days,
		"data_points":         len(history),
		"total_return":        totalReturn,
		"max_drawdown":        maxDrawdown,
		"current_pnl":         lastSnapshot.TotalPnL,
		"realized_profit":     lastSnapshot.RealizedPnL,
		"unrealized_pnl":      lastSnapshot.UnrealizedPnL,
		"total_invested":      lastSnapshot.TotalInvested,
		"current_value":       lastSnapshot.CurrentValue,
	}

	return metrics, nil
}

// GetStatus возвращает текстовый статус портфеля для Telegram
func (p *PortfolioManager) GetStatus() (string, error) {
	summary, err := p.GetPortfolioSummary()
	if err != nil {
		return "", err
	}

	totalInvested := summary["total_invested"].(float64)
	totalCurrentValue := summary["total_current_value"].(float64)
	totalPnL := summary["total_pnl"].(float64)
	totalReturnPercent := summary["total_return_percent"].(float64)

	status := fmt.Sprintf(
		"📊 Portfolio Status\n\n"+
			"Total Invested: %.2f USDT\n"+
			"Current Value: %.2f USDT\n"+
			"Total P&L: %.2f USDT (%.2f%%)\n\n"+
			"Active Positions: %d\n\n",
		totalInvested, totalCurrentValue, totalPnL, totalReturnPercent,
		summary["active_positions"].(int),
	)

	// Добавляем информацию по каждому активу
	assets := summary["assets"].([]map[string]interface{})
	for _, asset := range assets {
		symbol := asset["symbol"].(string)
		quantity := asset["quantity"].(float64)
		currentValue := asset["current_value"].(float64)
		returnPercent := asset["return_percent"].(float64)

		status += fmt.Sprintf(
			"%s: %.8f (%.2f USDT, %.2f%%)\n",
			symbol, quantity, currentValue, returnPercent,
		)
	}

	return status, nil
}
