package markets

import (
	"testing"

	"github.com/salt-ux/stock-bot/internal/config"
)

func TestNewServiceUnsupported(t *testing.T) {
	_, err := NewService(config.Config{Market: config.MarketConfig{Provider: "bad", CacheTTLSeconds: 5}})
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}
