package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
	"github.com/salt-ux/stock-bot/internal/trading"
)

type v22SimHandlerProvider struct{}

func (v v22SimHandlerProvider) GetQuote(_ context.Context, symbol string) (market.Quote, error) {
	return market.Quote{Symbol: symbol, Price: 70000, AsOf: time.Now().UTC()}, nil
}

func (v v22SimHandlerProvider) GetCandles(_ context.Context, symbol, interval string, limit int) ([]market.Candle, error) {
	return []market.Candle{{Symbol: symbol, Interval: interval, Time: time.Now().UTC(), Close: 70000}}, nil
}

func TestPaperV22SimulationStartAndStateHandlers(t *testing.T) {
	marketSvc := market.NewService(v22SimHandlerProvider{}, time.Second)
	paperSvc, err := trading.NewService(marketSvc, trading.Config{
		InitialCash:      50000000,
		DuplicateWindow:  30 * time.Second,
		MaxRecentOrders:  200,
		RiskMaxNotional:  10000000,
		RiskDailyLossCap: 1000000,
	})
	if err != nil {
		t.Fatalf("new trading service: %v", err)
	}
	executor := trading.NewBuyRuleExecutor(paperSvc, marketSvc)
	runner := trading.NewV22SimulationRunner(marketSvc, paperSvc, executor)
	defer runner.Stop()

	startHandler := paperV22SimulationStartHandler(runner)
	stateHandler := paperV22SimulationStateHandler(runner)

	startBody := []byte(`{"symbol":"005930","display_name":"삼성전자","days":1,"interval_seconds":10,"min_change_pct":-3,"max_change_pct":10,"base_price":70000}`)
	startReq := httptest.NewRequest(http.MethodPost, "/paper/v22-sim/start", bytes.NewReader(startBody))
	startRR := httptest.NewRecorder()
	startHandler.ServeHTTP(startRR, startReq)
	if startRR.Code != http.StatusOK {
		t.Fatalf("unexpected start status: %d body=%s", startRR.Code, startRR.Body.String())
	}

	stateReq := httptest.NewRequest(http.MethodGet, "/paper/v22-sim/state", nil)
	stateRR := httptest.NewRecorder()
	stateHandler.ServeHTTP(stateRR, stateReq)
	if stateRR.Code != http.StatusOK {
		t.Fatalf("unexpected state status: %d body=%s", stateRR.Code, stateRR.Body.String())
	}

	var state trading.V22SimulationState
	if err := json.Unmarshal(stateRR.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if state.Symbol != "005930" {
		t.Fatalf("unexpected symbol: %s", state.Symbol)
	}
}
