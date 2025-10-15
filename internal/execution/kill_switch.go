package execution

import (
	"sync"
	"time"
)

// KillSwitch аварийная остановка торговли
type KillSwitch struct {
	mu         sync.RWMutex
	active     bool
	activatedAt time.Time
	reason     string
}

// NewKillSwitch создает новый kill switch
func NewKillSwitch() *KillSwitch {
	return &KillSwitch{
		active: false,
	}
}

// Activate активирует kill switch
func (ks *KillSwitch) Activate(reason string) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.active = true
	ks.activatedAt = time.Now()
	ks.reason = reason

	// Логируем критическое событие
	println("🚨 KILL SWITCH ACTIVATED:", reason)
}

// Deactivate деактивирует kill switch (требует ручного вмешательства)
func (ks *KillSwitch) Deactivate() {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.active = false
	ks.reason = ""

	println("✅ Kill switch deactivated")
}

// IsActive проверяет активен ли kill switch
func (ks *KillSwitch) IsActive() bool {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	return ks.active
}

// GetStatus возвращает статус kill switch
func (ks *KillSwitch) GetStatus() (bool, string, time.Time) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	return ks.active, ks.reason, ks.activatedAt
}
