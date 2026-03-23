package strategy

import (
	"context"
	"testing"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
)

type fakeProvider struct{}

func (f fakeProvider) GetQuote(_ context.Context, symbol string) (market.Quote, error) {
	return market.Quote{Symbol: symbol, Price: 100, AsOf: time.Now().UTC()}, nil
}

func (f fakeProvider) GetCandles(_ context.Context, symbol string, interval string, limit int) ([]market.Candle, error) {
	out := make([]market.Candle, 0, limit)
	now := time.Now().UTC()
	for i := 0; i < limit; i++ {
		out = append(out, market.Candle{Symbol: symbol, Interval: interval, Time: now.Add(-time.Duration(limit-i) * time.Minute), Close: 100 + float64(i)})
	}
	return out, nil
}

type holdStrategy struct{}

func (h holdStrategy) Name() string { return "hold" }
func (h holdStrategy) Evaluate(candles []market.Candle) (Signal, error) {
	last := candles[len(candles)-1]
	return Signal{Action: SignalHold, Price: last.Close, At: last.Time, Reason: "test"}, nil
}

func TestEngineRun(t *testing.T) {
	svc := market.NewService(fakeProvider{}, 5*time.Second)
	engine := NewEngine(svc)

	result, err := engine.Run(context.Background(), "005930", "1d", 10, holdStrategy{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Strategy != "hold" {
		t.Fatalf("unexpected strategy name: %s", result.Strategy)
	}
	if result.Candles != 10 {
		t.Fatalf("unexpected candle count: %d", result.Candles)
	}
}
