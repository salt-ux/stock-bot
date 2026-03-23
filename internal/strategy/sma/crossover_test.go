package sma

import (
	"testing"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
	"github.com/salt-ux/stock-bot/internal/strategy"
)

func TestCrossoverBuy(t *testing.T) {
	s, err := NewCrossover(3, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	candles := buildCandles([]float64{10, 10, 10, 10, 10, 12})
	sig, err := s.Evaluate(candles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig.Action != strategy.SignalBuy {
		t.Fatalf("expected BUY, got %s", sig.Action)
	}
}

func TestCrossoverSell(t *testing.T) {
	s, err := NewCrossover(3, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	candles := buildCandles([]float64{10, 10, 10, 12, 12, 4})
	sig, err := s.Evaluate(candles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig.Action != strategy.SignalSell {
		t.Fatalf("expected SELL, got %s", sig.Action)
	}
}

func TestCrossoverHold(t *testing.T) {
	s, err := NewCrossover(3, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	candles := buildCandles([]float64{10, 10, 10, 10, 10, 10})
	sig, err := s.Evaluate(candles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig.Action != strategy.SignalHold {
		t.Fatalf("expected HOLD, got %s", sig.Action)
	}
}

func TestCrossoverNotEnoughCandles(t *testing.T) {
	s, err := NewCrossover(3, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = s.Evaluate(buildCandles([]float64{10, 11, 12}))
	if err == nil {
		t.Fatal("expected error for insufficient candles")
	}
}

func buildCandles(closes []float64) []market.Candle {
	base := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	out := make([]market.Candle, 0, len(closes))
	for i, c := range closes {
		tm := base.AddDate(0, 0, i)
		out = append(out, market.Candle{
			Symbol:   "005930",
			Interval: "1d",
			Time:     tm,
			Open:     c,
			High:     c,
			Low:      c,
			Close:    c,
			Volume:   100,
		})
	}
	return out
}
