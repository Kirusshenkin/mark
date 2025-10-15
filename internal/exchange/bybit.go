package exchange

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kirillm/dca-bot/internal/domain"
	"golang.org/x/time/rate"
)

type BybitClient struct {
	apiKey           string
	apiSecret        string
	baseURL          string
	client           *http.Client
	recvWindow       string
	maxRetries       int
	initialRetryDelay time.Duration
	rateLimiter      *rate.Limiter
}

type TickerResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			Symbol   string `json:"symbol"`
			LastPrice string `json:"lastPrice"`
			Bid1Price string `json:"bid1Price"`
			Ask1Price string `json:"ask1Price"`
		} `json:"list"`
	} `json:"result"`
}

type WalletBalanceResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			Coin []struct {
				Coin            string `json:"coin"`
				WalletBalance   string `json:"walletBalance"`
				AvailableToWithdraw string `json:"availableToWithdraw"`
			} `json:"coin"`
		} `json:"list"`
	} `json:"result"`
}

type OrderResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		OrderID   string `json:"orderId"`
		OrderLinkID string `json:"orderLinkId"`
	} `json:"result"`
}

type OrderInfo struct {
	OrderID       string
	ClientOrderID string
	Symbol        string
	Side          string
	Price         float64
	Quantity      float64
	Status        string
	CreatedAt     time.Time
}

func NewBybitClient(apiKey, apiSecret, baseURL string) *BybitClient {
	return &BybitClient{
		apiKey:           apiKey,
		apiSecret:        apiSecret,
		baseURL:          baseURL,
		client:           &http.Client{Timeout: 30 * time.Second},
		recvWindow:       domain.BybitRecvWindow,
		maxRetries:       3,
		initialRetryDelay: 500 * time.Millisecond,
		// Bybit limit: 10 requests per second
		// We use 100ms interval = 10 req/sec with burst of 10
		rateLimiter:      rate.NewLimiter(rate.Every(100*time.Millisecond), 10),
	}
}

// doWithRetry executes HTTP request with exponential backoff retry logic
func (b *BybitClient) doWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt < b.maxRetries; attempt++ {
		// Rate limiting: wait for permission to make request
		if err := b.rateLimiter.Wait(context.Background()); err != nil {
			return nil, fmt.Errorf("rate limiter error: %w", err)
		}

		resp, err = b.client.Do(req)

		// Success if no error and not a 5xx server error
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// Close response body if present before retry
		if resp != nil {
			resp.Body.Close()
		}

		// Don't retry on last attempt
		if attempt == b.maxRetries-1 {
			break
		}

		// Calculate exponential backoff: initialDelay * 2^attempt
		backoff := b.initialRetryDelay * time.Duration(1<<uint(attempt))

		// Log retry attempt (in production, use proper logger)
		if err != nil {
			fmt.Printf("Request failed (attempt %d/%d): %v. Retrying in %v...\n",
				attempt+1, b.maxRetries, err, backoff)
		} else {
			fmt.Printf("Request failed with status %d (attempt %d/%d). Retrying in %v...\n",
				resp.StatusCode, attempt+1, b.maxRetries, backoff)
		}

		time.Sleep(backoff)
	}

	// Return last error
	if err != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", b.maxRetries, err)
	}
	return resp, fmt.Errorf("request failed after %d retries with status %d", b.maxRetries, resp.StatusCode)
}

// GetPrice получает текущую цену актива
func (b *BybitClient) GetPrice(symbol string) (float64, error) {
	endpoint := "/v5/market/tickers"
	params := fmt.Sprintf("category=%s&symbol=%s", domain.BybitCategorySpot, symbol)

	url := fmt.Sprintf("%s%s?%s", b.baseURL, endpoint, params)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := b.doWithRetry(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	var tickerResp TickerResponse
	if err := json.Unmarshal(body, &tickerResp); err != nil {
		return 0, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if tickerResp.RetCode != 0 {
		return 0, fmt.Errorf("%w: %s", domain.ErrExchangeAPI, tickerResp.RetMsg)
	}

	if len(tickerResp.Result.List) == 0 {
		return 0, fmt.Errorf("no price data for symbol %s", symbol)
	}

	lastPrice := tickerResp.Result.List[0].LastPrice
	if lastPrice == "" {
		return 0, fmt.Errorf("empty price data for symbol %s", symbol)
	}

	price, err := strconv.ParseFloat(lastPrice, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price for %s: %w", symbol, err)
	}

	return price, nil
}

// GetBalance получает баланс монеты
func (b *BybitClient) GetBalance(coin string) (float64, error) {
	endpoint := "/v5/account/wallet-balance"
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	params := fmt.Sprintf("accountType=%s&coin=%s", domain.BybitAccountUnified, coin)

	signature := b.generateSignature(timestamp, params)

	url := fmt.Sprintf("%s%s?%s", b.baseURL, endpoint, params)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	b.setAuthHeaders(req, timestamp, signature)

	resp, err := b.doWithRetry(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	var balanceResp WalletBalanceResponse
	if err := json.Unmarshal(body, &balanceResp); err != nil {
		return 0, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if balanceResp.RetCode != 0 {
		return 0, fmt.Errorf("%w: %s", domain.ErrExchangeAPI, balanceResp.RetMsg)
	}

	if len(balanceResp.Result.List) == 0 || len(balanceResp.Result.List[0].Coin) == 0 {
		return 0, nil
	}

	for _, coinData := range balanceResp.Result.List[0].Coin {
		if coinData.Coin == coin {
			// Проверяем, что строка не пустая перед парсингом
			if coinData.AvailableToWithdraw == "" {
				return 0, nil
			}
			balance, err := strconv.ParseFloat(coinData.AvailableToWithdraw, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse balance for %s: %w", coin, err)
			}
			return balance, nil
		}
	}

	return 0, nil
}

// PlaceOrder размещает рыночный ордер
func (b *BybitClient) PlaceOrder(symbol, side string, quantity float64) (*OrderInfo, error) {
	endpoint := "/v5/order/create"
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	// Генерируем уникальный clientOrderId для идемпотентности
	// Формат: "dca_{timestamp}_{symbol}_{side}"
	clientOrderId := fmt.Sprintf("dca_%s_%s_%s", timestamp, symbol, side)

	params := map[string]interface{}{
		"category":      domain.BybitCategorySpot,
		"symbol":        symbol,
		"side":          side,
		"orderType":     domain.OrderTypeMarket,
		"qty":           fmt.Sprintf("%.8f", quantity),
		"orderLinkId":   clientOrderId, // Для идемпотентности
	}

	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	signature := b.generateSignature(timestamp, string(jsonData))

	url := fmt.Sprintf("%s%s", b.baseURL, endpoint)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	b.setAuthHeaders(req, timestamp, signature)

	resp, err := b.doWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var orderResp OrderResponse
	if err := json.Unmarshal(body, &orderResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if orderResp.RetCode != 0 {
		return nil, fmt.Errorf("%w: %s", domain.ErrExchangeAPI, orderResp.RetMsg)
	}

	return &OrderInfo{
		OrderID:       orderResp.Result.OrderID,
		ClientOrderID: clientOrderId,
		Symbol:        symbol,
		Side:          side,
		Quantity:      quantity,
		Status:        domain.StatusFilled,
		CreatedAt:     time.Now(),
	}, nil
}

// generateSignature генерирует подпись для запросов (GET и POST)
func (b *BybitClient) generateSignature(timestamp, payload string) string {
	message := timestamp + b.apiKey + b.recvWindow + payload
	h := hmac.New(sha256.New, []byte(b.apiSecret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// setAuthHeaders устанавливает заголовки авторизации для запроса
func (b *BybitClient) setAuthHeaders(req *http.Request, timestamp, signature string) {
	req.Header.Set("X-BAPI-API-KEY", b.apiKey)
	req.Header.Set("X-BAPI-SIGN", signature)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", b.recvWindow)
}

// CalculateOrderAmount рассчитывает количество актива для покупки
func (b *BybitClient) CalculateOrderAmount(symbol string, usdtAmount float64) (float64, error) {
	price, err := b.GetPrice(symbol)
	if err != nil {
		return 0, err
	}
	return usdtAmount / price, nil
}

// GetCurrentPrice - alias для GetPrice для совместимости
func (b *BybitClient) GetCurrentPrice(symbol string) (float64, error) {
	return b.GetPrice(symbol)
}
