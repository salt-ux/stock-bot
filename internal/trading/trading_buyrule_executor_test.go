package trading

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
)

type fixedQuoteProvider struct {
	mu     sync.RWMutex
	prices map[string]float64
}

func (p *fixedQuoteProvider) SetPrice(symbol string, price float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.prices == nil {
		p.prices = make(map[string]float64)
	}
	p.prices[symbol] = price
}

func (p *fixedQuoteProvider) GetQuote(_ context.Context, symbol string) (market.Quote, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	price, ok := p.prices[symbol]
	if !ok {
		return market.Quote{}, fmt.Errorf("quote not found: %s", symbol)
	}
	return market.Quote{
		Symbol: symbol,
		Price:  price,
		AsOf:   time.Now().UTC(),
	}, nil
}

func (p *fixedQuoteProvider) GetCandles(_ context.Context, symbol string, interval string, limit int) ([]market.Candle, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestBuyRuleExecutorExecutesEarlyStageBuy(t *testing.T) {
	provider := &fixedQuoteProvider{}
	provider.SetPrice("009830", 10000)
	marketSvc := market.NewService(provider, time.Nanosecond)

	paperSvc, err := NewService(marketSvc, Config{
		InitialCash:      50000000,
		DuplicateWindow:  30 * time.Second,
		MaxRecentOrders:  200,
		RiskMaxNotional:  10000000,
		RiskDailyLossCap: 1000000,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	executor := NewBuyRuleExecutor(paperSvc, marketSvc)
	result, err := executor.Execute(context.Background(), BuyRuleExecuteRequest{
		Items: []BuyRuleExecuteItem{
			{
				Symbol:       "009830",
				DisplayName:  "한화솔루션",
				PrincipalKRW: 4000000,
				SplitCount:   40,
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.TotalOrders != 1 || result.TotalBuyOrders != 1 || result.TotalSellOrders != 0 {
		t.Fatalf("unexpected order totals: %+v", result)
	}
	if len(result.Results) != 1 {
		t.Fatalf("unexpected results length: %d", len(result.Results))
	}
	if result.Results[0].PlannedBuyQty != 10 {
		t.Fatalf("unexpected planned buy qty: %d", result.Results[0].PlannedBuyQty)
	}
	state := paperSvc.GetState()
	if len(state.Positions) != 1 || state.Positions[0].Qty != 10 {
		t.Fatalf("unexpected positions: %+v", state.Positions)
	}
}

func TestBuyRuleExecutorExecutesSellWhenTargetsReached(t *testing.T) {
	provider := &fixedQuoteProvider{}
	provider.SetPrice("009830", 10000)
	marketSvc := market.NewService(provider, time.Nanosecond)

	paperSvc, err := NewService(marketSvc, Config{
		InitialCash:      50000000,
		DuplicateWindow:  30 * time.Second,
		MaxRecentOrders:  200,
		RiskMaxNotional:  10000000,
		RiskDailyLossCap: 1000000,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if _, err := paperSvc.PlaceOrder(context.Background(), OrderRequest{
		Symbol: "009830",
		Side:   SideBuy,
		Qty:    20,
	}); err != nil {
		t.Fatalf("seed buy order: %v", err)
	}

	provider.SetPrice("009830", 12000)
	executor := NewBuyRuleExecutor(paperSvc, marketSvc)
	result, err := executor.Execute(context.Background(), BuyRuleExecuteRequest{
		Items: []BuyRuleExecuteItem{
			{
				Symbol:       "009830",
				DisplayName:  "한화솔루션",
				PrincipalKRW: 4000000,
				SplitCount:   40,
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.TotalOrders != 1 || result.TotalSellOrders != 1 {
		t.Fatalf("unexpected sell totals: %+v", result)
	}
	if len(result.Results) != 1 {
		t.Fatalf("unexpected results length: %d", len(result.Results))
	}
	if result.Results[0].PlannedSellQty != 20 {
		t.Fatalf("unexpected planned sell qty: %d", result.Results[0].PlannedSellQty)
	}

	state := paperSvc.GetState()
	if len(state.Positions) != 0 {
		t.Fatalf("expected position to be closed, got: %+v", state.Positions)
	}
}

func TestSummarizeSymbolOrderStatsResetsBuyCountAfterSell(t *testing.T) {
	orders := []Order{
		{Symbol: "009830", Side: SideSell, Qty: 4, FillPrice: 12000, Status: "FILLED"},
		{Symbol: "009830", Side: SideBuy, Qty: 3, FillPrice: 10000, Status: "FILLED"},
		{Symbol: "009830", Side: SideBuy, Qty: 2, FillPrice: 10000, Status: "FILLED"},
	}

	stats := summarizeSymbolOrderStats(orders, "009830")
	if stats.BuyCountSinceLastSell != 0 {
		t.Fatalf("expected buy count reset to 0 after sell, got %d", stats.BuyCountSinceLastSell)
	}
	if stats.RealizedPnLKRW != 8000 {
		t.Fatalf("unexpected realized pnl: %v", stats.RealizedPnLKRW)
	}
}

func TestEvaluateBuyRuleCapsBuyByRemainingPrincipal(t *testing.T) {
	item := BuyRuleExecuteItem{
		Symbol:       "009830",
		DisplayName:  "한화솔루션",
		PrincipalKRW: 100000,
		SplitCount:   10,
	}

	result := evaluateBuyRule(item, Position{Symbol: item.Symbol, Qty: 9, AvgPrice: 10000}, 2, 10000, 0)
	if result.PlannedBuyQty != 1 {
		t.Fatalf("expected buy qty capped to 1 by remaining principal, got %d", result.PlannedBuyQty)
	}
	if result.RemainingKRW != 10000 {
		t.Fatalf("expected remaining principal 10000, got %d", result.RemainingKRW)
	}
	if result.BuyCount != 2 {
		t.Fatalf("expected buy count to be preserved, got %d", result.BuyCount)
	}
}

func TestEvaluateBuyRuleAppliesRealizedPnLToPrincipal(t *testing.T) {
	item := BuyRuleExecuteItem{
		Symbol:       "009830",
		DisplayName:  "한화솔루션",
		PrincipalKRW: 100000,
		SplitCount:   10,
	}

	result := evaluateBuyRule(item, Position{Symbol: item.Symbol}, 0, 6000, 20000)
	if result.AppliedPrincipal != 120000 {
		t.Fatalf("expected applied principal 120000, got %d", result.AppliedPrincipal)
	}
	if result.PlannedBuyQty != 2 {
		t.Fatalf("expected planned buy qty 2, got %d", result.PlannedBuyQty)
	}
}
