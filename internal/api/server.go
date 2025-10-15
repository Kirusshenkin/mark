package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/kirillm/dca-bot/internal/exchange"
	"github.com/kirillm/dca-bot/internal/storage"
	"github.com/kirillm/dca-bot/internal/strategy"
	"github.com/kirillm/dca-bot/pkg/utils"
)

type Server struct {
	logger           *utils.Logger
	exchange         *exchange.BybitClient
	storage          *storage.PostgresStorage
	dcaStrategy      *strategy.DCAStrategy
	autoSell         *strategy.AutoSellStrategy
	gridStrategy     *strategy.GridStrategy
	portfolioManager *strategy.PortfolioManager
	port             int
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type BuyRequest struct {
	Symbol      string  `json:"symbol"`
	QuoteAmount float64 `json:"quoteAmount"`
}

type GridInitRequest struct {
	Symbol         string  `json:"symbol"`
	Levels         int     `json:"levels"`
	SpacingPercent float64 `json:"spacing_percent"`
	OrderSizeQuote float64 `json:"order_size_quote"`
}

func NewServer(
	logger *utils.Logger,
	exchange *exchange.BybitClient,
	storage *storage.PostgresStorage,
	dcaStrategy *strategy.DCAStrategy,
	autoSell *strategy.AutoSellStrategy,
	gridStrategy *strategy.GridStrategy,
	portfolioManager *strategy.PortfolioManager,
	port int,
) *Server {
	return &Server{
		logger:           logger,
		exchange:         exchange,
		storage:          storage,
		dcaStrategy:      dcaStrategy,
		autoSell:         autoSell,
		gridStrategy:     gridStrategy,
		portfolioManager: portfolioManager,
		port:             port,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/balance", s.handleBalance)
	mux.HandleFunc("/buy", s.handleBuy)
	mux.HandleFunc("/grid/init", s.handleGridInit)
	mux.HandleFunc("/portfolio", s.handlePortfolio)

	addr := fmt.Sprintf(":%d", s.port)
	s.logger.Info("Starting HTTP server on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

// handleHealth - health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"uptime":    "running",
	}

	s.sendSuccess(w, health)
}

// handleStatus - get trading status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get DCA status
	dcaStatus, err := s.dcaStrategy.GetStatus()
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get DCA status: %v", err), http.StatusInternalServerError)
		return
	}

	// Get Auto-Sell status
	autoSellStatus, err := s.autoSell.GetStatus()
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get Auto-Sell status: %v", err), http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"dca_status":       dcaStatus,
		"auto_sell_status": autoSellStatus,
		"timestamp":        time.Now().Unix(),
	}

	s.sendSuccess(w, status)
}

// handleBalance - get account balance
func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get USDT balance
	usdtBalance, err := s.exchange.GetBalance("USDT")
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get USDT balance: %v", err), http.StatusInternalServerError)
		return
	}

	// Get all balances from DB
	balances, err := s.storage.GetAllBalances()
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get balances: %v", err), http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"usdt_balance": usdtBalance,
		"positions":    balances,
		"timestamp":    time.Now().Unix(),
	}

	s.sendSuccess(w, result)
}

// handleBuy - execute manual buy
func (s *Server) handleBuy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BuyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Symbol == "" {
		s.sendError(w, "Symbol is required", http.StatusBadRequest)
		return
	}

	if req.QuoteAmount <= 0 {
		s.sendError(w, "Quote amount must be positive", http.StatusBadRequest)
		return
	}

	// Execute buy
	if err := s.dcaStrategy.ExecuteManualBuy(); err != nil {
		s.sendError(w, fmt.Sprintf("Buy failed: %v", err), http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"message":   "Buy executed successfully",
		"symbol":    req.Symbol,
		"amount":    req.QuoteAmount,
		"timestamp": time.Now().Unix(),
	}

	s.sendSuccess(w, result)
}

// handleGridInit - initialize grid trading
func (s *Server) handleGridInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GridInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Symbol == "" {
		s.sendError(w, "Symbol is required", http.StatusBadRequest)
		return
	}

	if req.Levels <= 0 {
		s.sendError(w, "Levels must be positive", http.StatusBadRequest)
		return
	}

	if req.SpacingPercent <= 0 {
		s.sendError(w, "Spacing percent must be positive", http.StatusBadRequest)
		return
	}

	if req.OrderSizeQuote <= 0 {
		s.sendError(w, "Order size must be positive", http.StatusBadRequest)
		return
	}

	// Initialize grid
	if s.gridStrategy != nil {
		// Create or get asset
		asset, err := s.storage.GetAsset(req.Symbol)
		if err != nil || asset == nil {
			asset = &storage.Asset{
				Symbol:             req.Symbol,
				Enabled:            true,
				StrategyType:       "GRID",
				GridLevels:         req.Levels,
				GridSpacingPercent: req.SpacingPercent,
				GridOrderSize:      req.OrderSizeQuote,
			}
			if err := s.storage.CreateOrUpdateAsset(asset); err != nil {
				s.sendError(w, fmt.Sprintf("Failed to create asset: %v", err), http.StatusInternalServerError)
				return
			}
		}

		if err := s.gridStrategy.InitializeGrid(asset); err != nil {
			s.sendError(w, fmt.Sprintf("Grid initialization failed: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		s.sendError(w, "Grid strategy not available", http.StatusServiceUnavailable)
		return
	}

	result := map[string]interface{}{
		"message":          "Grid initialized successfully",
		"symbol":           req.Symbol,
		"levels":           req.Levels,
		"spacing_percent":  req.SpacingPercent,
		"order_size_quote": req.OrderSizeQuote,
		"timestamp":        time.Now().Unix(),
	}

	s.sendSuccess(w, result)
}

// handlePortfolio - get portfolio status
func (s *Server) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.portfolioManager == nil {
		s.sendError(w, "Portfolio manager not available", http.StatusServiceUnavailable)
		return
	}

	status, err := s.portfolioManager.GetStatus()
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get portfolio status: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, map[string]interface{}{
		"portfolio": status,
		"timestamp": time.Now().Unix(),
	})
}

// Helper methods
func (s *Server) sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
	})
}

func (s *Server) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   message,
	})
}

// Helper function to parse query parameter
func getQueryParam(r *http.Request, key string, defaultValue string) string {
	if value := r.URL.Query().Get(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper function to parse int query parameter
func getQueryParamInt(r *http.Request, key string, defaultValue int) int {
	if value := r.URL.Query().Get(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
