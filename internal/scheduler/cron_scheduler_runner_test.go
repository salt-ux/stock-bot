package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/salt-ux/stock-bot/internal/config"
	"github.com/salt-ux/stock-bot/internal/strategy"
	"github.com/salt-ux/stock-bot/internal/trading"
)

type fakeStrategyEngine struct {
	result strategy.RunResult
	err    error
}

func (f fakeStrategyEngine) Run(_ context.Context, symbol, interval string, limit int, _ strategy.Strategy) (strategy.RunResult, error) {
	if symbol == "" || interval == "" || limit <= 0 {
		return strategy.RunResult{}, context.Canceled
	}
	return f.result, f.err
}

type fakePaperTrader struct {
	lastReq trading.OrderRequest
	called  bool
}

func (f *fakePaperTrader) PlaceOrder(_ context.Context, req trading.OrderRequest) (trading.Order, error) {
	f.called = true
	f.lastReq = req
	return trading.Order{ID: 99, Symbol: req.Symbol, Side: req.Side, Qty: req.Qty, Status: "FILLED", FilledAt: time.Now().UTC()}, nil
}

func TestRunAutoTradeOncePlacesBuyOrder(t *testing.T) {
	engine := fakeStrategyEngine{
		result: strategy.RunResult{
			Strategy: "test",
			Signal: strategy.Signal{
				Action: strategy.SignalBuy,
				Reason: "buy",
			},
		},
	}
	trader := &fakePaperTrader{}
	runner, err := NewCronSchedulerRunner(config.SchedulerConfig{
		Enabled:           true,
		Timezone:          "Asia/Seoul",
		MarketOpenCron:    "0 9 * * 1-5",
		AutoTradeCron:     "30 10 * * 1-5",
		AutoTradeSymbol:   "005930",
		AutoTradeInterval: "1d",
		AutoTradeLimit:    60,
		AutoTradeStrategy: "sma",
		AutoTradeQty:      2,
		SMAShortWindow:    5,
		SMALongWindow:     20,
	}, engine, trader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runner.runAutoTradeOnce(context.Background())
	if !trader.called {
		t.Fatal("expected order to be placed")
	}
	if trader.lastReq.Side != trading.SideBuy {
		t.Fatalf("unexpected side: %s", trader.lastReq.Side)
	}
	if trader.lastReq.Qty != 2 {
		t.Fatalf("unexpected qty: %d", trader.lastReq.Qty)
	}
}

func TestRunAutoTradeOnceHoldSkipsOrder(t *testing.T) {
	engine := fakeStrategyEngine{
		result: strategy.RunResult{
			Strategy: "test",
			Signal: strategy.Signal{
				Action: strategy.SignalHold,
				Reason: "hold",
			},
		},
	}
	trader := &fakePaperTrader{}
	runner, err := NewCronSchedulerRunner(config.SchedulerConfig{
		Enabled:           true,
		Timezone:          "Asia/Seoul",
		MarketOpenCron:    "0 9 * * 1-5",
		AutoTradeCron:     "30 10 * * 1-5",
		AutoTradeSymbol:   "005930",
		AutoTradeInterval: "1d",
		AutoTradeLimit:    60,
		AutoTradeStrategy: "sma",
		AutoTradeQty:      1,
		SMAShortWindow:    5,
		SMALongWindow:     20,
	}, engine, trader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runner.runAutoTradeOnce(context.Background())
	if trader.called {
		t.Fatal("expected no order to be placed for HOLD")
	}
	snapshot := runner.Snapshot()
	if snapshot.LastAutoTrade == nil {
		t.Fatal("expected last auto-trade result to be stored")
	}
	if snapshot.LastAutoTrade.Signal != string(strategy.SignalHold) {
		t.Fatalf("unexpected signal: %s", snapshot.LastAutoTrade.Signal)
	}
}
