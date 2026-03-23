package trading

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
)

type v22SimTestProvider struct{}

func (v v22SimTestProvider) GetQuote(_ context.Context, symbol string) (market.Quote, error) {
	return market.Quote{
		Symbol: symbol,
		Price:  70000,
		AsOf:   time.Now().UTC(),
	}, nil
}

func (v v22SimTestProvider) GetCandles(_ context.Context, symbol string, interval string, limit int) ([]market.Candle, error) {
	return []market.Candle{
		{
			Symbol:   symbol,
			Interval: interval,
			Time:     time.Now().UTC(),
			Open:     70000,
			High:     70000,
			Low:      70000,
			Close:    70000,
			Volume:   1,
		},
	}, nil
}

func TestGenerateV22SimulationPrices(t *testing.T) {
	prices, changes := generateV22SimulationPrices(42, 70000, 50, -3, 10)
	if len(prices) != 50 {
		t.Fatalf("unexpected prices length: %d", len(prices))
	}
	if len(changes) != 50 {
		t.Fatalf("unexpected changes length: %d", len(changes))
	}
	if changes[0] != 0 {
		t.Fatalf("expected first change to be 0, got %v", changes[0])
	}
	for i := 1; i < len(changes); i++ {
		if changes[i] < -3 || changes[i] > 10 {
			t.Fatalf("change out of range at %d: %v", i, changes[i])
		}
	}
}

func TestV22SimulationRunnerStartCompletesOneStep(t *testing.T) {
	marketSvc := market.NewService(v22SimTestProvider{}, time.Second)
	paperSvc, err := NewService(marketSvc, Config{
		InitialCash:      50000000,
		DuplicateWindow:  30 * time.Second,
		MaxRecentOrders:  200,
		RiskMaxNotional:  10000000,
		RiskDailyLossCap: 1000000,
	})
	if err != nil {
		t.Fatalf("new paper service: %v", err)
	}
	executor := NewBuyRuleExecutor(paperSvc, marketSvc)
	runner := NewV22SimulationRunner(marketSvc, paperSvc, executor)

	_, err = runner.Start(context.Background(), V22SimulationStartRequest{
		Symbol:          "005930",
		DisplayName:     "삼성전자",
		Days:            1,
		IntervalSeconds: 10,
		MinChangePct:    -3,
		MaxChangePct:    10,
		BasePrice:       70000,
		PrincipalKRW:    4000000,
		SplitCount:      40,
		Seed:            1,
	})
	if err != nil {
		t.Fatalf("start simulation: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		state := runner.State()
		if !state.Running {
			if state.CurrentStep != 1 {
				t.Fatalf("unexpected current step: %d", state.CurrentStep)
			}
			if len(state.Ticks) != 1 {
				t.Fatalf("unexpected tick length: %d", len(state.Ticks))
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("simulation did not finish in time")
}

func TestV22SimulationRunnerDoesNotHitDuplicateWindowAcrossSimDays(t *testing.T) {
	marketSvc := market.NewService(v22SimTestProvider{}, time.Second)
	paperSvc, err := NewService(marketSvc, Config{
		InitialCash:      50000000,
		DuplicateWindow:  30 * time.Second,
		MaxRecentOrders:  200,
		RiskMaxNotional:  10000000,
		RiskDailyLossCap: 1000000,
	})
	if err != nil {
		t.Fatalf("new paper service: %v", err)
	}
	executor := NewBuyRuleExecutor(paperSvc, marketSvc)
	runner := NewV22SimulationRunner(marketSvc, paperSvc, executor)

	_, err = runner.Start(context.Background(), V22SimulationStartRequest{
		Symbol:          "005930",
		DisplayName:     "삼성전자",
		Days:            3,
		IntervalSeconds: 1,
		MinChangePct:    -3,
		MaxChangePct:    10,
		BasePrice:       70000,
		PrincipalKRW:    4000000,
		SplitCount:      10,
		Seed:            20260302,
	})
	if err != nil {
		t.Fatalf("start simulation: %v", err)
	}

	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		state := runner.State()
		if !state.Running {
			if len(state.Ticks) != 3 {
				t.Fatalf("unexpected tick length: %d", len(state.Ticks))
			}
			for _, tick := range state.Ticks {
				if strings.Contains(tick.Message, "duplicate order blocked") {
					t.Fatalf("duplicate order blocked in tick: %+v", tick)
				}
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("simulation did not finish in time")
}
