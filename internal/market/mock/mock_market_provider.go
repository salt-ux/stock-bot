package mock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
)

type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) GetQuote(_ context.Context, symbol string) (market.Quote, error) {
	symbol = strings.TrimSpace(strings.ToUpper(symbol))
	if symbol == "" {
		return market.Quote{}, fmt.Errorf("symbol is required")
	}

	base := float64(symbolSeed(symbol)%90000+10000) / 100
	jitter := float64(time.Now().Unix()%1000) / 100
	price := base + jitter

	return market.Quote{
		Symbol: symbol,
		Price:  price,
		AsOf:   time.Now().UTC(),
	}, nil
}

func (p *Provider) GetCandles(_ context.Context, symbol string, interval string, limit int) ([]market.Candle, error) {
	symbol = strings.TrimSpace(strings.ToUpper(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if interval == "" {
		interval = "1d"
	}
	if limit < 1 || limit > 500 {
		return nil, fmt.Errorf("limit must be between 1 and 500")
	}

	step := intervalStep(interval)
	seed := float64(symbolSeed(symbol)%5000+1000) / 10
	now := time.Now().UTC().Truncate(step)

	candles := make([]market.Candle, 0, limit)
	for i := limit - 1; i >= 0; i-- {
		tm := now.Add(-time.Duration(i) * step)
		base := seed + float64((limit-i)%17)
		open := base
		close := base + float64((i%5)-2)*0.6
		high := max(open, close) + 0.8
		low := min(open, close) - 0.8
		volume := int64(1000 + (i+1)*37)

		candles = append(candles, market.Candle{
			Symbol:   symbol,
			Interval: interval,
			Time:     tm,
			Open:     open,
			High:     high,
			Low:      low,
			Close:    close,
			Volume:   volume,
		})
	}

	return candles, nil
}

func symbolSeed(symbol string) int {
	total := 0
	for _, r := range symbol {
		total += int(r)
	}
	return total
}

func intervalStep(interval string) time.Duration {
	switch strings.ToLower(interval) {
	case "1m":
		return time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "1h":
		return time.Hour
	default:
		return 24 * time.Hour
	}
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
