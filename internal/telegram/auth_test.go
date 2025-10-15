package telegram

import (
	"testing"
	"time"
)

func TestNewAuthManager(t *testing.T) {
	tests := []struct {
		name         string
		adminIDs     string
		whitelist    string
		wantAdmins   int
		wantWhitelist int
	}{
		{"empty", "", "", 0, 0},
		{"single admin", "123", "", 1, 0},
		{"multiple admins", "123,456,789", "", 3, 0},
		{"with whitelist", "123", "456,789", 1, 2},
		{"with spaces", "123, 456, 789", "", 3, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := NewAuthManager(tt.adminIDs, tt.whitelist)
			if len(am.adminIDs) != tt.wantAdmins {
				t.Errorf("NewAuthManager() admins = %v, want %v", len(am.adminIDs), tt.wantAdmins)
			}
			if len(am.whitelist) != tt.wantWhitelist {
				t.Errorf("NewAuthManager() whitelist = %v, want %v", len(am.whitelist), tt.wantWhitelist)
			}
		})
	}
}

func TestAuthManager_IsAdmin(t *testing.T) {
	am := NewAuthManager("123,456", "")

	tests := []struct {
		name   string
		userID int64
		want   bool
	}{
		{"admin 1", 123, true},
		{"admin 2", 456, true},
		{"not admin", 789, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := am.IsAdmin(tt.userID); got != tt.want {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthManager_IsAdmin_EmptyList(t *testing.T) {
	// Если список админов пуст, все должны быть админами
	am := NewAuthManager("", "")

	if !am.IsAdmin(123) {
		t.Error("IsAdmin() should return true when admin list is empty")
	}
}

func TestAuthManager_IsAllowed(t *testing.T) {
	am := NewAuthManager("123", "456,789")

	tests := []struct {
		name   string
		userID int64
		want   bool
	}{
		{"admin (always allowed)", 123, true},
		{"whitelisted", 456, true},
		{"whitelisted 2", 789, true},
		{"not allowed", 999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := am.IsAllowed(tt.userID); got != tt.want {
				t.Errorf("IsAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthManager_IsAllowed_NoWhitelist(t *testing.T) {
	// Если whitelist не включен, все должны быть allowed
	am := NewAuthManager("123", "")

	if !am.IsAllowed(999) {
		t.Error("IsAllowed() should return true when whitelist is disabled")
	}
}

func TestAuthManager_CheckRateLimit(t *testing.T) {
	am := NewAuthManager("", "")

	userID := int64(123)
	maxRequests := 2

	// Первые 2 запроса должны пройти
	if err := am.CheckRateLimit(userID, maxRequests); err != nil {
		t.Errorf("CheckRateLimit() first request failed: %v", err)
	}

	if err := am.CheckRateLimit(userID, maxRequests); err != nil {
		t.Errorf("CheckRateLimit() second request failed: %v", err)
	}

	// Третий запрос должен быть заблокирован
	if err := am.CheckRateLimit(userID, maxRequests); err == nil {
		t.Error("CheckRateLimit() should have blocked third request")
	}

	// После секунды лимит должен сброситься
	time.Sleep(1100 * time.Millisecond)

	if err := am.CheckRateLimit(userID, maxRequests); err != nil {
		t.Errorf("CheckRateLimit() should have reset after 1 second: %v", err)
	}
}

func TestAuthManager_RequireAdmin(t *testing.T) {
	am := NewAuthManager("123", "")

	tests := []struct {
		name    string
		userID  int64
		wantErr bool
	}{
		{"admin", 123, false},
		{"not admin", 456, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := am.RequireAdmin(tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequireAdmin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthManager_AddRemoveAdmin(t *testing.T) {
	// Инициализируем с одним админом, чтобы список не был пустым
	am := NewAuthManager("999", "")

	userID := int64(123)

	// Изначально не админ
	if am.IsAdmin(userID) {
		t.Error("User should not be admin initially")
	}

	// Добавляем
	am.AddAdmin(userID)
	if !am.IsAdmin(userID) {
		t.Error("User should be admin after AddAdmin()")
	}

	// Удаляем
	am.RemoveAdmin(userID)
	if am.IsAdmin(userID) {
		t.Error("User should not be admin after RemoveAdmin()")
	}
}

func TestAuthManager_AddRemoveWhitelist(t *testing.T) {
	am := NewAuthManager("", "123")

	userID := int64(456)

	// Изначально не в whitelist (и не admin, поэтому не allowed)
	if am.IsAllowed(userID) {
		t.Error("User should not be allowed initially")
	}

	// Добавляем
	am.AddToWhitelist(userID)
	if !am.IsAllowed(userID) {
		t.Error("User should be allowed after AddToWhitelist()")
	}

	// Удаляем
	am.RemoveFromWhitelist(userID)
	if am.IsAllowed(userID) {
		t.Error("User should not be allowed after RemoveFromWhitelist()")
	}
}

func TestAuthManager_GetAdminIDs(t *testing.T) {
	am := NewAuthManager("123,456,789", "")

	ids := am.GetAdminIDs()

	if len(ids) != 3 {
		t.Errorf("GetAdminIDs() returned %d IDs, want 3", len(ids))
	}

	// Проверяем, что все ID присутствуют
	found := make(map[int64]bool)
	for _, id := range ids {
		found[id] = true
	}

	wantIDs := []int64{123, 456, 789}
	for _, wantID := range wantIDs {
		if !found[wantID] {
			t.Errorf("GetAdminIDs() missing ID %d", wantID)
		}
	}
}

func TestAuthManager_CleanupRateLimiters(t *testing.T) {
	am := NewAuthManager("", "")

	// Создаем несколько rate limiters
	am.CheckRateLimit(123, 5)
	am.CheckRateLimit(456, 5)
	am.CheckRateLimit(789, 5)

	if len(am.rateLimiters) != 3 {
		t.Errorf("Expected 3 rate limiters, got %d", len(am.rateLimiters))
	}

	// Изменяем время последнего запроса для одного из них (симулируем старый)
	am.mu.Lock()
	if limiter, exists := am.rateLimiters[123]; exists {
		limiter.lastRequest = time.Now().Add(-10 * time.Minute)
	}
	am.mu.Unlock()

	// Очищаем
	am.CleanupRateLimiters()

	// Должен остаться только 2
	if len(am.rateLimiters) != 2 {
		t.Errorf("Expected 2 rate limiters after cleanup, got %d", len(am.rateLimiters))
	}

	// Убеждаемся, что удален именно старый
	am.mu.RLock()
	if _, exists := am.rateLimiters[123]; exists {
		t.Error("Old rate limiter should have been cleaned up")
	}
	am.mu.RUnlock()
}
