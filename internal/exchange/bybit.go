package exchange

import (
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
)

type BybitClient struct {
	apiKey     string
	apiSecret  string
	baseURL    string
	client     *http.Client
	recvWindow string
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
	OrderID   string
	Symbol    string
	Side      string
	Price     float64
	Quantity  float64
	Status    string
	CreatedAt time.Time
}

func NewBybitClient(apiKey, apiSecret, baseURL string) *BybitClient {
	return &BybitClient{
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		baseURL:    baseURL,
		client:     &http.Client{Timeout: 30 * time.Second},
		recvWindow: domain.BybitRecvWindow,
	}
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

	resp, err := b.client.Do(req)
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

	resp, err := b.client.Do(req)
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

	params := map[string]interface{}{
		"category":  domain.BybitCategorySpot,
		"symbol":    symbol,
		"side":      side,
		"orderType": domain.OrderTypeMarket,
		"qty":       fmt.Sprintf("%.8f", quantity),
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

	resp, err := b.client.Do(req)
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
		OrderID:   orderResp.Result.OrderID,
		Symbol:    symbol,
		Side:      side,
		Quantity:  quantity,
		Status:    domain.StatusFilled,
		CreatedAt: time.Now(),
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
