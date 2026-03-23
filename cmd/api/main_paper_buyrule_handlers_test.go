package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
	"github.com/salt-ux/stock-bot/internal/trading"
)

type fixedQuoteMarketProvider struct {
	prices map[string]float64
}

func (p fixedQuoteMarketProvider) GetQuote(_ context.Context, symbol string) (market.Quote, error) {
	price, ok := p.prices[symbol]
	if !ok {
		return market.Quote{}, fmt.Errorf("quote not found: %s", symbol)
	}
	return market.Quote{Symbol: symbol, Price: price, AsOf: time.Now().UTC()}, nil
}

func (p fixedQuoteMarketProvider) GetCandles(_ context.Context, symbol, interval string, limit int) ([]market.Candle, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestPaperBuyRuleExecuteHandler(t *testing.T) {
	marketSvc := market.NewService(fixedQuoteMarketProvider{
		prices: map[string]float64{"009830": 10000},
	}, time.Second)
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
	handler := paperBuyRuleExecuteHandler(executor, paperSvc, nil)

	reqBody := []byte(`{"items":[{"symbol":"009830","display_name":"한화솔루션","principal_krw":4000000,"split_count":40}]}`)
	req := httptest.NewRequest(http.MethodPost, "/paper/buyrule/execute", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}

	var body trading.BuyRuleExecuteResult
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.TotalOrders != 1 || body.TotalBuyOrders != 1 {
		t.Fatalf("unexpected totals: %+v", body)
	}
}
