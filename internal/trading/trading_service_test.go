package trading

import (
	"context"
	"testing"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
)

type fakeProvider struct {
	price float64
}

func (f fakeProvider) GetQuote(_ context.Context, symbol string) (market.Quote, error) {
	return market.Quote{Symbol: symbol, Price: f.price, AsOf: time.Now().UTC()}, nil
}

func (f fakeProvider) GetCandles(_ context.Context, symbol, interval string, limit int) ([]market.Candle, error) {
	return []market.Candle{{Symbol: symbol, Interval: interval, Time: time.Now().UTC(), Close: f.price}}, nil
}

func newSvc(t *testing.T, price float64) *Service {
	t.Helper()
	m := market.NewService(fakeProvider{price: price}, time.Second)
	s, err := NewService(m, Config{
		InitialCash:      1000000,
		DuplicateWindow:  2 * time.Second,
		MaxRecentOrders:  10,
		RiskMaxNotional:  300000,
		RiskDailyLossCap: 50000,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return s
}

func TestPlaceOrderBuySell(t *testing.T) {
	svc := newSvc(t, 100)

	_, err := svc.PlaceOrder(context.Background(), OrderRequest{Symbol: "005930", Side: SideBuy, Qty: 10})
	if err != nil {
		t.Fatalf("buy failed: %v", err)
	}

	svc.market = market.NewService(fakeProvider{price: 120}, time.Second)
	_, err = svc.PlaceOrder(context.Background(), OrderRequest{Symbol: "005930", Side: SideSell, Qty: 5})
	if err != nil {
		t.Fatalf("sell failed: %v", err)
	}

	state := svc.GetState()
	if state.RealizedPnLToday <= 0 {
		t.Fatalf("expected positive realized pnl, got %f", state.RealizedPnLToday)
	}
}

func TestDuplicateOrderBlocked(t *testing.T) {
	svc := newSvc(t, 100)
	_, err := svc.PlaceOrder(context.Background(), OrderRequest{Symbol: "005930", Side: SideBuy, Qty: 1})
	if err != nil {
		t.Fatalf("first order failed: %v", err)
	}
	_, err = svc.PlaceOrder(context.Background(), OrderRequest{Symbol: "005930", Side: SideBuy, Qty: 1})
	if err == nil {
		t.Fatal("expected duplicate block")
	}
}

func TestDuplicateOrderAllowedWhenQuoteAsOfProgresses(t *testing.T) {
	svc := newSvc(t, 100)
	baseAt := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)

	if err := svc.market.SetQuoteOverride("005930", 100, baseAt); err != nil {
		t.Fatalf("set first quote override: %v", err)
	}
	_, err := svc.PlaceOrder(context.Background(), OrderRequest{Symbol: "005930", Side: SideBuy, Qty: 1})
	if err != nil {
		t.Fatalf("first order failed: %v", err)
	}

	if err := svc.market.SetQuoteOverride("005930", 101, baseAt.Add(24*time.Hour)); err != nil {
		t.Fatalf("set second quote override: %v", err)
	}
	_, err = svc.PlaceOrder(context.Background(), OrderRequest{Symbol: "005930", Side: SideBuy, Qty: 1})
	if err != nil {
		t.Fatalf("second order should not be blocked on next day quote: %v", err)
	}
}

func TestMaxPositionBlocked(t *testing.T) {
	svc := newSvc(t, 100)
	_, err := svc.PlaceOrder(context.Background(), OrderRequest{Symbol: "005930", Side: SideBuy, Qty: 4000})
	if err == nil {
		t.Fatal("expected max position blocked")
	}
}

func TestResetClearsState(t *testing.T) {
	svc := newSvc(t, 100)
	_, err := svc.PlaceOrder(context.Background(), OrderRequest{Symbol: "005930", Side: SideBuy, Qty: 10})
	if err != nil {
		t.Fatalf("buy failed: %v", err)
	}
	before := svc.GetState()
	if len(before.Positions) == 0 {
		t.Fatalf("expected position before reset")
	}

	svc.Reset()
	after := svc.GetState()
	if len(after.Positions) != 0 {
		t.Fatalf("expected no positions after reset, got: %+v", after.Positions)
	}
	if len(after.RecentOrders) != 0 {
		t.Fatalf("expected no recent orders after reset, got: %+v", after.RecentOrders)
	}
	if after.Cash != svc.cfg.InitialCash {
		t.Fatalf("expected cash restored to initial cash %.2f, got %.2f", svc.cfg.InitialCash, after.Cash)
	}
}
