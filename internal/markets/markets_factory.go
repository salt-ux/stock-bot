package markets

import (
	"fmt"
	"strings"
	"time"

	"github.com/salt-ux/stock-bot/internal/config"
	"github.com/salt-ux/stock-bot/internal/market"
	"github.com/salt-ux/stock-bot/internal/market/kiwoom"
	"github.com/salt-ux/stock-bot/internal/market/mock"
)

func NewService(cfg config.Config) (*market.Service, error) {
	ttl := time.Duration(cfg.Market.CacheTTLSeconds) * time.Second
	switch strings.ToLower(strings.TrimSpace(cfg.Market.Provider)) {
	case "", "mock":
		return market.NewService(mock.NewProvider(), ttl), nil
	case "kiwoom":
		return market.NewService(kiwoom.NewProvider(cfg.Kiwoom, cfg.Market), ttl), nil
	default:
		return nil, fmt.Errorf("unsupported market provider: %s", cfg.Market.Provider)
	}
}
