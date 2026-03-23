package brokers

import (
	"fmt"

	"github.com/salt-ux/stock-bot/internal/broker"
	"github.com/salt-ux/stock-bot/internal/broker/kiwoom"
	"github.com/salt-ux/stock-bot/internal/config"
)

func NewCredentialValidator(cfg config.Config) (broker.CredentialValidator, error) {
	switch cfg.Broker.Provider {
	case "kiwoom":
		return kiwoom.NewClient(cfg.Kiwoom), nil
	default:
		return nil, fmt.Errorf("unsupported broker provider: %s", cfg.Broker.Provider)
	}
}
