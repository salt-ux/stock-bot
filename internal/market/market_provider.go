package market

import "context"

type Provider interface {
	GetQuote(ctx context.Context, symbol string) (Quote, error)
	GetCandles(ctx context.Context, symbol string, interval string, limit int) ([]Candle, error)
}
