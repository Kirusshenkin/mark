package telegram

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AuthManager управляет правами доступа и rate limiting
type AuthManager struct {
	adminIDs      map[int64]bool
	whitelist     map[int64]bool
	rateLimiters  map[int64]*RateLimiter
	mu            sync.RWMutex
	enableWhitelist bool
}

// RateLimiter ограничивает частоту запросов от пользователя
type RateLimiter struct {
	lastRequest time.Time
	requestCount int
	mu sync.Mutex
}

// NewAuthManager создает новый менеджер авторизации
func NewAuthManager(adminIDsStr, whitelistStr string) *AuthManager {
	am := &AuthManager{
		adminIDs:     make(map[int64]bool),
		whitelist:    make(map[int64]bool),
		rateLimiters: make(map[int64]*RateLimiter),
	}

	// Парсим админов
	if adminIDsStr != "" {
		for _, idStr := range strings.Split(adminIDsStr, ",") {
			idStr = strings.TrimSpace(idStr)
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				am.adminIDs[id] = true
			}
		}
	}

	// Парсим whitelist
	if whitelistStr != "" {
		am.enableWhitelist = true
		for _, idStr := range strings.Split(whitelistStr, ",") {
			idStr = strings.TrimSpace(idStr)
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				am.whitelist[id] = true
			}
		}
	}

	return am
}

// IsAdmin проверяет, является ли пользователь администратором
func (am *AuthManager) IsAdmin(userID int64) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Если список админов пуст, разрешаем всем
	if len(am.adminIDs) == 0 {
		return true
	}

	return am.adminIDs[userID]
}

// IsAllowed проверяет, разрешен ли доступ пользователю
func (am *AuthManager) IsAllowed(userID int64) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Если whitelist не включен, разрешаем всем
	if !am.enableWhitelist {
		return true
	}

	// Админы всегда разрешены
	if am.adminIDs[userID] {
		return true
	}

	return am.whitelist[userID]
}

// CheckRateLimit проверяет rate limit для пользователя
func (am *AuthManager) CheckRateLimit(userID int64, maxRequestsPerSecond int) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Получаем или создаем rate limiter для пользователя
	limiter, exists := am.rateLimiters[userID]
	if !exists {
		limiter = &RateLimiter{
			lastRequest: time.Time{},
			requestCount: 0,
		}
		am.rateLimiters[userID] = limiter
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()

	// Если прошла секунда, сбрасываем счетчик
	if now.Sub(limiter.lastRequest) >= time.Second {
		limiter.requestCount = 0
		limiter.lastRequest = now
	}

	// Проверяем лимит
	limiter.requestCount++
	if limiter.requestCount > maxRequestsPerSecond {
		waitTime := time.Second - now.Sub(limiter.lastRequest)
		return fmt.Errorf("rate limit exceeded, please wait %v", waitTime.Round(time.Millisecond))
	}

	return nil
}

// RequireAdmin возвращает ошибку, если пользователь не администратор
func (am *AuthManager) RequireAdmin(userID int64) error {
	if !am.IsAdmin(userID) {
		return fmt.Errorf("access denied: admin permission required")
	}
	return nil
}

// AddAdmin добавляет администратора
func (am *AuthManager) AddAdmin(userID int64) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.adminIDs[userID] = true
}

// RemoveAdmin удаляет администратора
func (am *AuthManager) RemoveAdmin(userID int64) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.adminIDs, userID)
}

// AddToWhitelist добавляет пользователя в whitelist
func (am *AuthManager) AddToWhitelist(userID int64) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.whitelist[userID] = true
}

// RemoveFromWhitelist удаляет пользователя из whitelist
func (am *AuthManager) RemoveFromWhitelist(userID int64) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.whitelist, userID)
}

// GetAdminIDs возвращает список ID администраторов
func (am *AuthManager) GetAdminIDs() []int64 {
	am.mu.RLock()
	defer am.mu.RUnlock()

	ids := make([]int64, 0, len(am.adminIDs))
	for id := range am.adminIDs {
		ids = append(ids, id)
	}
	return ids
}

// CleanupRateLimiters очищает старые rate limiters (вызывать периодически)
func (am *AuthManager) CleanupRateLimiters() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	for userID, limiter := range am.rateLimiters {
		limiter.mu.Lock()
		// Удаляем неактивные более 5 минут
		if now.Sub(limiter.lastRequest) > 5*time.Minute {
			delete(am.rateLimiters, userID)
		}
		limiter.mu.Unlock()
	}
}
