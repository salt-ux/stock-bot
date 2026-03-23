package kiwoom

import (
	"testing"

	"github.com/salt-ux/stock-bot/internal/market"
)

func TestExtractSymbolLookupResults(t *testing.T) {
	resp := map[string]any{
		"data": []any{
			map[string]any{
				"stk_cd": "009830",
				"stk_nm": "한화솔루션",
				"mkt_nm": "KOSPI",
			},
			map[string]any{
				"stk_cd": "009835",
				"stk_nm": "한화솔루션우",
				"mkt_nm": "KOSPI",
			},
		},
	}

	items := extractSymbolLookupResults(resp, 10)
	if len(items) != 2 {
		t.Fatalf("unexpected length: %d", len(items))
	}
	if items[0].Symbol != "009830" || items[0].Name != "한화솔루션" {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
}

func TestFilterAndRankSymbolResults(t *testing.T) {
	items := []market.SymbolLookupResult{
		{Symbol: "009835", Name: "한화솔루션우", Market: "KOSPI"},
		{Symbol: "009830", Name: "한화솔루션", Market: "KOSPI"},
		{Symbol: "042660", Name: "한화오션", Market: "KOSPI"},
	}

	got := filterAndRankSymbolResults(items, "한화솔", 10)
	if len(got) < 2 {
		t.Fatalf("expected at least 2 items, got %d", len(got))
	}
	if got[0].Symbol != "009830" {
		t.Fatalf("unexpected first symbol: %s", got[0].Symbol)
	}
}

func TestSearchPayloadCandidates(t *testing.T) {
	byName := searchPayloadCandidates("한화솔루션")
	if len(byName) == 0 {
		t.Fatalf("expected name payload candidates")
	}
	if _, ok := byName[0]["stk_nm"]; !ok {
		t.Fatalf("expected first payload to include stk_nm: %+v", byName[0])
	}

	byCode := searchPayloadCandidates("009830")
	if len(byCode) == 0 {
		t.Fatalf("expected code payload candidates")
	}
	if _, ok := byCode[0]["stk_cd"]; !ok {
		t.Fatalf("expected first payload to include stk_cd: %+v", byCode[0])
	}
}
