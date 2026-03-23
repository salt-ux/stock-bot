package strategy

import "github.com/salt-ux/stock-bot/internal/market"

type Strategy interface {
	Name() string
	Evaluate(candles []market.Candle) (Signal, error)
}
