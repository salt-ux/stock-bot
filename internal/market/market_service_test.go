package market

import (
	"context"
	"testing"
	"time"
)

type fakeProvider struct {
	quoteCalls int
}

func (f *fakeProvider) GetQuote(_ context.Context, symbol string) (Quote, error) {
	f.quoteCalls++
	return Quote{Symbol: symbol, Price: 100, AsOf: time.Now()}, nil
}

func (f *fakeProvider) GetCandles(_ context.Context, symbol, interval string, limit int) ([]Candle, error) {
	candles := make([]Candle, 0, limit)
	for i := 0; i < limit; i++ {
		candles = append(candles, Candle{Symbol: symbol, Interval: interval, Close: 10 + float64(i)})
	}
	return candles, nil
}

func TestQuoteUsesCache(t *testing.T) {
	provider := &fakeProvider{}
	svc := NewService(provider, 5*time.Second)

	_, err := svc.Quote(context.Background(), "005930")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = svc.Quote(context.Background(), "005930")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.quoteCalls != 1 {
		t.Fatalf("expected 1 provider call, got %d", provider.quoteCalls)
	}
}

func TestCandlesLimitValidation(t *testing.T) {
	provider := &fakeProvider{}
	svc := NewService(provider, 5*time.Second)

	_, err := svc.Candles(context.Background(), "005930", "1d", 0)
	if err == nil {
		t.Fatal("expected validation error for limit")
	}
}

func TestQuoteOverride(t *testing.T) {
	provider := &fakeProvider{}
	svc := NewService(provider, 5*time.Second)

	if err := svc.SetQuoteOverride("005930", 70100, time.Time{}); err != nil {
		t.Fatalf("set override: %v", err)
	}
	quote, err := svc.Quote(context.Background(), "005930")
	if err != nil {
		t.Fatalf("quote with override: %v", err)
	}
	if quote.Price != 70100 {
		t.Fatalf("expected override price 70100, got %v", quote.Price)
	}
	if provider.quoteCalls != 0 {
		t.Fatalf("expected provider not called with override, got %d", provider.quoteCalls)
	}

	svc.ClearQuoteOverride("005930")
	quote, err = svc.Quote(context.Background(), "005930")
	if err != nil {
		t.Fatalf("quote after clear override: %v", err)
	}
	if quote.Price != 100 {
		t.Fatalf("expected provider quote price 100, got %v", quote.Price)
	}
	if provider.quoteCalls != 1 {
		t.Fatalf("expected provider call count 1 after clear override, got %d", provider.quoteCalls)
	}
}
