package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
	"github.com/salt-ux/stock-bot/internal/market/mock"
)

func TestResolveSymbolQueryByCode(t *testing.T) {
	resolver := market.NewDefaultSymbolLookupResolver()
	got, err := resolveSymbolQuery("005930", resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "005930" {
		t.Fatalf("unexpected symbol: %s", got)
	}
}

func TestResolveSymbolQueryByName(t *testing.T) {
	resolver := market.NewDefaultSymbolLookupResolver()
	got, err := resolveSymbolQuery("삼성전자", resolver)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "005930" {
		t.Fatalf("unexpected symbol: %s", got)
	}
}

func TestMarketQuoteHandlerAcceptsNameInput(t *testing.T) {
	resolver := market.NewDefaultSymbolLookupResolver()
	svc := market.NewService(mock.NewProvider(), time.Second)
	handler := marketQuoteHandler(svc, resolver)

	req := httptest.NewRequest(http.MethodGet, "/market/quote?symbol=%EC%82%BC%EC%84%B1%EC%A0%84%EC%9E%90", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}

	var quote market.Quote
	if err := json.Unmarshal(rr.Body.Bytes(), &quote); err != nil {
		t.Fatalf("decode quote: %v", err)
	}
	if quote.Symbol != "005930" {
		t.Fatalf("unexpected resolved symbol: %s", quote.Symbol)
	}
}

func TestMarketSymbolSearchHandler(t *testing.T) {
	handler := marketSymbolSearchHandler(market.NewDefaultSymbolLookupResolver())
	req := httptest.NewRequest(http.MethodGet, "/market/symbols/search?query=%EC%82%BC%EC%84%B1&limit=5", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}

	var body struct {
		Items []market.SymbolLookupResult `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Items) == 0 {
		t.Fatalf("expected non-empty search items")
	}
}
