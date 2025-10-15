package manager

import (
	"fmt"
	"sync"
	"time"

	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
	"github.com/kirillm/dca-bot/internal/strategy"
	"github.com/kirillm/dca-bot/pkg/utils"
)

// MultiAssetManager C?@02;O5B =5A:>;L:8<8 0:B820<8 8 8E AB@0B538O<8
type MultiAssetManager struct {
	storage       *storage.PostgresStorage
	exchange      *exchange.BybitClient
	logger        *utils.Logger
	dcaStrategies map[string]*strategy.DCAStrategy
	gridStrategies map[string]*strategy.GridStrategy
	autoSellStrategies map[string]*strategy.AutoSellStrategy
	mu            sync.RWMutex
	stopChan      chan bool
	notifyFunc    func(string)
}

func NewMultiAssetManager(
	storage *storage.PostgresStorage,
	exchange *exchange.BybitClient,
	logger *utils.Logger,
	notifyFunc func(string),
) *MultiAssetManager {
	return &MultiAssetManager{
		storage:            storage,
		exchange:           exchange,
		logger:             logger,
		dcaStrategies:      make(map[string]*strategy.DCAStrategy),
		gridStrategies:     make(map[string]*strategy.GridStrategy),
		autoSellStrategies: make(map[string]*strategy.AutoSellStrategy),
		stopChan:           make(chan bool),
		notifyFunc:         notifyFunc,
	}
}

// Start 70?CA:05B <5=5465@
func (m *MultiAssetManager) Start() {
	m.logger.Info("Multi-Asset Manager starting...")

	// 03@C605< 2A5 0:B82=K5 0:B82K
	assets, err := m.storage.GetEnabledAssets()
	if err != nil {
		m.logger.Error("Failed to load enabled assets: %v", err)
		return
	}

	m.logger.Info("Found %d enabled assets", len(assets))

	// =8F80;878@C5< AB@0B5388 4;O :064>3> 0:B820
	for _, asset := range assets {
		if err := m.InitializeAsset(&asset); err != nil {
			m.logger.Error("Failed to initialize asset %s: %v", asset.Symbol, err)
		}
	}

	// 0?CA:05< <>=8B>@8=3
	go m.monitor()

	m.logger.Info("Multi-Asset Manager started")
}

// Stop >AB0=02;8205B <5=5465@
func (m *MultiAssetManager) Stop() {
	m.logger.Info("Stopping Multi-Asset Manager...")

	m.stopChan <- true

	// AB0=02;8205< 2A5 AB@0B5388
	m.mu.Lock()
	for symbol, dca := range m.dcaStrategies {
		m.logger.Info("Stopping DCA strategy for %s", symbol)
		dca.Stop()
	}
	for symbol, autoSell := range m.autoSellStrategies {
		m.logger.Info("Stopping Auto-Sell strategy for %s", symbol)
		autoSell.Stop()
	}
	m.mu.Unlock()

	m.logger.Info("Multi-Asset Manager stopped")
}

// InitializeAsset 8=8F80;878@C5B AB@0B5388 4;O 0:B820
func (m *MultiAssetManager) InitializeAsset(asset *storage.Asset) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Initializing asset %s with strategy %s", asset.Symbol, asset.StrategyType)

	switch asset.StrategyType {
	case "DCA":
		return m.initializeDCAStrategy(asset)
	case "GRID":
		return m.initializeGridStrategy(asset)
	case "HYBRID":
		// DCA + Auto-Sell
		if err := m.initializeDCAStrategy(asset); err != nil {
			return err
		}
		return m.initializeAutoSellStrategy(asset)
	default:
		return fmt.Errorf("unknown strategy type: %s", asset.StrategyType)
	}
}

// initializeDCAStrategy 8=8F80;878@C5B DCA AB@0B538N
func (m *MultiAssetManager) initializeDCAStrategy(asset *storage.Asset) error {
	// @>25@O5<, =5 ACI5AB2C5B ;8 C65
	if _, exists := m.dcaStrategies[asset.Symbol]; exists {
		return nil
	}

	interval := time.Duration(asset.DCAInterval) * time.Minute

	dcaStrategy := strategy.NewDCAStrategy(
		m.exchange,
		m.storage,
		m.logger,
		asset.Symbol,
		asset.DCAAmount,
		interval,
		m.notifyFunc,
	)

	m.dcaStrategies[asset.Symbol] = dcaStrategy
	go dcaStrategy.Start()

	m.logger.Info("DCA strategy initialized for %s", asset.Symbol)
	return nil
}

// initializeGridStrategy 8=8F80;878@C5B Grid AB@0B538N
func (m *MultiAssetManager) initializeGridStrategy(asset *storage.Asset) error {
	gridStrategy := strategy.NewGridStrategy(m.storage, m.exchange)

	if err := gridStrategy.InitializeGrid(asset); err != nil {
		return fmt.Errorf("failed to initialize grid: %w", err)
	}

	m.gridStrategies[asset.Symbol] = gridStrategy
	m.logger.Info("Grid strategy initialized for %s", asset.Symbol)
	return nil
}

// initializeAutoSellStrategy 8=8F80;878@C5B Auto-Sell AB@0B538N
func (m *MultiAssetManager) initializeAutoSellStrategy(asset *storage.Asset) error {
	if _, exists := m.autoSellStrategies[asset.Symbol]; exists {
		return nil
	}

	autoSellStrategy := strategy.NewAutoSellStrategy(
		m.exchange,
		m.storage,
		m.logger,
		asset.Symbol,
		asset.AutoSellTriggerPercent,
		asset.AutoSellAmountPercent,
		5*time.Minute, // check interval
		m.notifyFunc,
	)

	if !asset.AutoSellEnabled {
		autoSellStrategy.Disable()
	}

	m.autoSellStrategies[asset.Symbol] = autoSellStrategy
	go autoSellStrategy.Start()

	m.logger.Info("Auto-Sell strategy initialized for %s", asset.Symbol)
	return nil
}

// AddAsset 4>102;O5B =>2K9 0:B82
func (m *MultiAssetManager) AddAsset(asset *storage.Asset) error {
	m.logger.Info("Adding new asset: %s", asset.Symbol)

	// !>E@0=O5< 2 
	if err := m.storage.CreateOrUpdateAsset(asset); err != nil {
		return fmt.Errorf("failed to save asset: %w", err)
	}

	// A;8 enabled, 8=8F80;878@C5<
	if asset.Enabled {
		if err := m.InitializeAsset(asset); err != nil {
			return fmt.Errorf("failed to initialize asset: %w", err)
		}
	}

	if m.notifyFunc != nil {
		m.notifyFunc(fmt.Sprintf(" Asset %s added with %s strategy", asset.Symbol, asset.StrategyType))
	}

	return nil
}

// RemoveAsset C40;O5B 0:B82
func (m *MultiAssetManager) RemoveAsset(symbol string) error {
	m.logger.Info("Removing asset: %s", symbol)

	// AB0=02;8205< AB@0B5388
	m.mu.Lock()
	if dca, exists := m.dcaStrategies[symbol]; exists {
		dca.Stop()
		delete(m.dcaStrategies, symbol)
	}
	if autoSell, exists := m.autoSellStrategies[symbol]; exists {
		autoSell.Stop()
		delete(m.autoSellStrategies, symbol)
	}
	delete(m.gridStrategies, symbol)
	m.mu.Unlock()

	// 50:B828@C5< 2 
	if err := m.storage.DisableAsset(symbol); err != nil {
		return fmt.Errorf("failed to disable asset: %w", err)
	}

	if m.notifyFunc != nil {
		m.notifyFunc(fmt.Sprintf("=Ñ Asset %s removed", symbol))
	}

	return nil
}

// monitor >BA;568205B Grid AB@0B5388
func (m *MultiAssetManager) monitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.monitorGridStrategies()
		case <-m.stopChan:
			return
		}
	}
}

// monitorGridStrategies >BA;568205B Grid AB@0B5388
func (m *MultiAssetManager) monitorGridStrategies() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for symbol, gridStrategy := range m.gridStrategies {
		asset, err := m.storage.GetAsset(symbol)
		if err != nil || asset == nil || !asset.Enabled {
			continue
		}

		if err := gridStrategy.MonitorGrid(asset); err != nil {
			m.logger.Error("Grid monitor error for %s: %v", symbol, err)
		}
	}
}

// GetAssetStatus 2>72@0I05B AB0BCA 0:B820
func (m *MultiAssetManager) GetAssetStatus(symbol string) (string, error) {
	asset, err := m.storage.GetAsset(symbol)
	if err != nil {
		return "", err
	}
	if asset == nil {
		return "", fmt.Errorf("asset not found: %s", symbol)
	}

	balance, err := m.storage.GetBalance(symbol)
	if err != nil {
		return "", err
	}

	currentPrice, err := m.exchange.GetPrice(symbol)
	if err != nil {
		return "", err
	}

	currentValue := balance.TotalQuantity * currentPrice
	unrealizedPnL := currentValue - balance.TotalInvested
	totalPnL := balance.RealizedProfit + unrealizedPnL

	status := fmt.Sprintf(
		"=Ê Asset Status: %s\n\n"+
			"Strategy: %s\n"+
			"Status: %s\n"+
			"Current Price: %.2f USDT\n"+
			"Total Quantity: %.8f\n"+
			"Avg Entry: %.2f USDT\n"+
			"Invested: %.2f USDT\n"+
			"Current Value: %.2f USDT\n"+
			"Realized P&L: %.2f USDT\n"+
			"Unrealized P&L: %.2f USDT\n"+
			"Total P&L: %.2f USDT",
		asset.Symbol,
		asset.StrategyType,
		func() string { if asset.Enabled { return "Active " } else { return "Inactive ø" } }(),
		currentPrice,
		balance.TotalQuantity,
		balance.AvgEntryPrice,
		balance.TotalInvested,
		currentValue,
		balance.RealizedProfit,
		unrealizedPnL,
		totalPnL,
	)

	return status, nil
}

// GetAllAssets 2>72@0I05B A?8A>: 2A5E 0:B82>2
func (m *MultiAssetManager) GetAllAssets() ([]storage.Asset, error) {
	return m.storage.GetAllAssets()
}

// UpdateAsset >1=>2;O5B ?0@0<5B@K 0:B820
func (m *MultiAssetManager) UpdateAsset(asset *storage.Asset) error {
	return m.storage.CreateOrUpdateAsset(asset)
}
