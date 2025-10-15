package manager

import (
	"fmt"
	"time"

	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
	"github.com/kirillm/dca-bot/pkg/utils"
)

// PortfolioManager manages user's portfolio
type PortfolioManager struct {
	storage  *storage.PostgresStorage
	exchange *exchange.BybitClient
	logger   *utils.Logger
}

func NewPortfolioManager(
	storage *storage.PostgresStorage,
	exchange *exchange.BybitClient,
	logger *utils.Logger,
) *PortfolioManager {
	return &PortfolioManager{
		storage:  storage,
		exchange: exchange,
		logger:   logger,
	}
}

// GetPortfolioSummary returns portfolio summary
func (p *PortfolioManager) GetPortfolioSummary() (string, error) {
	assets, err := p.storage.GetEnabledAssets()
	if err != nil {
		return "", fmt.Errorf("failed to get assets: %w", err)
	}

	if len(assets) == 0 {
		return "📊 Portfolio is empty. Add assets to get started!", nil
	}

	totalInvested := 0.0
	totalCurrentValue := 0.0
	totalRealizedProfit := 0.0
	totalUnrealizedPnL := 0.0

	summary := "📊 **Portfolio Summary**\n\n"

	for _, asset := range assets {
		balance, err := p.storage.GetBalance(asset.Symbol)
		if err != nil {
			continue
		}

		currentPrice, err := p.exchange.GetPrice(asset.Symbol)
		if err != nil {
			continue
		}

		currentValue := balance.TotalQuantity * currentPrice
		unrealizedPnL := currentValue - balance.TotalInvested

		totalInvested += balance.TotalInvested
		totalCurrentValue += currentValue
		totalRealizedProfit += balance.RealizedProfit
		totalUnrealizedPnL += unrealizedPnL

		profitPercent := 0.0
		if balance.TotalInvested > 0 {
			profitPercent = ((currentValue + balance.RealizedProfit - balance.TotalInvested) / balance.TotalInvested) * 100
		}

		summary += fmt.Sprintf("**%s** (%s)\n", asset.Symbol, asset.StrategyType)
		summary += fmt.Sprintf("  Value: $%.2f | P&L: $%.2f (%.2f%%)\n\n", currentValue, unrealizedPnL+balance.RealizedProfit, profitPercent)
	}

	totalPnL := totalRealizedProfit + totalUnrealizedPnL
	totalReturn := 0.0
	if totalInvested > 0 {
		totalReturn = (totalPnL / totalInvested) * 100
	}

	summary += "---\n"
	summary += fmt.Sprintf("**Total Invested:** $%.2f\n", totalInvested)
	summary += fmt.Sprintf("**Current Value:** $%.2f\n", totalCurrentValue)
	summary += fmt.Sprintf("**Realized Profit:** $%.2f\n", totalRealizedProfit)
	summary += fmt.Sprintf("**Unrealized P&L:** $%.2f\n", totalUnrealizedPnL)
	summary += fmt.Sprintf("**Total P&L:** $%.2f (%.2f%%)\n", totalPnL, totalReturn)

	return summary, nil
}

// GetAssetAllocation returns asset allocation in portfolio
func (p *PortfolioManager) GetAssetAllocation() (string, error) {
	assets, err := p.storage.GetEnabledAssets()
	if err != nil {
		return "", err
	}

	if len(assets) == 0 {
		return "No assets in portfolio", nil
	}

	totalValue := 0.0
	assetValues := make(map[string]float64)

	for _, asset := range assets {
		balance, err := p.storage.GetBalance(asset.Symbol)
		if err != nil {
			continue
		}

		currentPrice, err := p.exchange.GetPrice(asset.Symbol)
		if err != nil {
			continue
		}

		value := balance.TotalQuantity * currentPrice
		assetValues[asset.Symbol] = value
		totalValue += value
	}

	allocation := "📊 **Asset Allocation**\n\n"
	for symbol, value := range assetValues {
		percentage := (value / totalValue) * 100
		allocation += fmt.Sprintf("%s: %.2f%% ($%.2f)\n", symbol, percentage, value)
	}
	allocation += fmt.Sprintf("\nTotal: $%.2f", totalValue)

	return allocation, nil
}

// SavePnLSnapshot saves PnL snapshot for analytics
func (p *PortfolioManager) SavePnLSnapshot(snapshotType string) error {
	assets, err := p.storage.GetEnabledAssets()
	if err != nil {
		return err
	}

	for _, asset := range assets {
		balance, err := p.storage.GetBalance(asset.Symbol)
		if err != nil {
			continue
		}

		currentPrice, err := p.exchange.GetPrice(asset.Symbol)
		if err != nil {
			continue
		}

		currentValue := balance.TotalQuantity * currentPrice
		unrealizedPnL := currentValue - balance.TotalInvested
		totalPnL := balance.RealizedProfit + unrealizedPnL

		returnPercent := 0.0
		if balance.TotalInvested > 0 {
			returnPercent = (totalPnL / balance.TotalInvested) * 100
		}

		snapshot := &storage.PnLHistory{
			Symbol:        asset.Symbol,
			RealizedPnL:   balance.RealizedProfit,
			UnrealizedPnL: unrealizedPnL,
			TotalPnL:      totalPnL,
			TotalInvested: balance.TotalInvested,
			CurrentValue:  currentValue,
			ReturnPercent: returnPercent,
			SnapshotType:  snapshotType,
			CreatedAt:     time.Now(),
		}

		if err := p.storage.SavePnLSnapshot(snapshot); err != nil {
			p.logger.Error("Failed to save PnL snapshot for %s: %v", asset.Symbol, err)
		}
	}

	return nil
}

// GetPerformanceMetrics returns performance metrics for a period
func (p *PortfolioManager) GetPerformanceMetrics(symbol string, days int) (string, error) {
	// Get historical data
	history, err := p.storage.GetPnLHistory(symbol, "DAILY", days)
	if err != nil {
		return "", err
	}

	if len(history) == 0 {
		return fmt.Sprintf("No historical data for %s", symbol), nil
	}

	// Current position
	balance, err := p.storage.GetBalance(symbol)
	if err != nil {
		return "", err
	}

	currentPrice, err := p.exchange.GetPrice(symbol)
	if err != nil {
		return "", err
	}

	currentValue := balance.TotalQuantity * currentPrice
	unrealizedPnL := currentValue - balance.TotalInvested
	totalPnL := balance.RealizedProfit + unrealizedPnL

	// Calculate metrics
	metrics := fmt.Sprintf(
		"📊 **Performance Metrics: %s** (Last %d days)\n\n"+
			"Current Price: $%.2f\n"+
			"Total Invested: $%.2f\n"+
			"Current Value: $%.2f\n"+
			"Total P&L: $%.2f (%.2f%%)\n"+
			"Realized Profit: $%.2f\n"+
			"Unrealized P&L: $%.2f\n"+
			"Data Points: %d\n",
		symbol,
		days,
		currentPrice,
		balance.TotalInvested,
		currentValue,
		totalPnL,
		func() float64 {
			if balance.TotalInvested > 0 {
				return (totalPnL / balance.TotalInvested) * 100
			}
			return 0.0
		}(),
		balance.RealizedProfit,
		unrealizedPnL,
		len(history),
	)

	return metrics, nil
}

// AllocateCapital allocates capital between assets
func (p *PortfolioManager) AllocateCapital(totalCapital float64) error {
	assets, err := p.storage.GetEnabledAssets()
	if err != nil {
		return err
	}

	if len(assets) == 0 {
		return fmt.Errorf("no assets to allocate capital")
	}

	// Simple equal distribution
	capitalPerAsset := totalCapital / float64(len(assets))

	for _, asset := range assets {
		asset.AllocatedCapital = capitalPerAsset
		if err := p.storage.CreateOrUpdateAsset(&asset); err != nil {
			p.logger.Error("Failed to update allocated capital for %s: %v", asset.Symbol, err)
		}
	}

	p.logger.Info("Allocated $%.2f across %d assets ($%.2f each)", totalCapital, len(assets), capitalPerAsset)
	return nil
}

// RebalancePortfolio rebalances portfolio (optional function)
func (p *PortfolioManager) RebalancePortfolio() error {
	p.logger.Info("Portfolio rebalancing requested")
	// TODO: implement rebalancing logic
	return fmt.Errorf("portfolio rebalancing not yet implemented")
}
