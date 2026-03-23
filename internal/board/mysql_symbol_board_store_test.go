package board

import "testing"

func TestNormalizeSymbolRecord(t *testing.T) {
	got, err := normalizeSymbolRecord(SymbolRecord{
		Symbol:        " aapl ",
		DisplayName:   "",
		PrincipalKRW:  -1,
		SplitCount:    200,
		IsSelected:    true,
		ProgressState: "run",
		SellRatioPct:  999,
		TradeMethod:   "",
		NoteText:      "  테스트 비고  ",
	}, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Symbol != "AAPL" {
		t.Fatalf("unexpected symbol: %s", got.Symbol)
	}
	if got.DisplayName != "AAPL" {
		t.Fatalf("unexpected display_name: %s", got.DisplayName)
	}
	if got.PrincipalKRW != DefaultPrincipalKRW {
		t.Fatalf("unexpected principal: %d", got.PrincipalKRW)
	}
	if got.SplitCount != MaxSplitCount {
		t.Fatalf("unexpected split_count: %d", got.SplitCount)
	}
	if got.SortOrder != 3 {
		t.Fatalf("unexpected sort_order: %d", got.SortOrder)
	}
	if !got.IsSelected {
		t.Fatalf("unexpected is_selected: %v", got.IsSelected)
	}
	if got.ProgressState != ProgressStateRun {
		t.Fatalf("unexpected progress_state: %s", got.ProgressState)
	}
	if got.SellRatioPct != MaxSellRatioPct {
		t.Fatalf("unexpected sell_ratio_pct: %d", got.SellRatioPct)
	}
	if got.TradeMethod != DefaultTradeMethod {
		t.Fatalf("unexpected trade_method: %s", got.TradeMethod)
	}
	if got.NoteText != "테스트 비고" {
		t.Fatalf("unexpected note_text: %q", got.NoteText)
	}
}

func TestNormalizeSymbolRecordRejectsInvalidSymbol(t *testing.T) {
	_, err := normalizeSymbolRecord(SymbolRecord{Symbol: "AAPL@"}, 0)
	if err == nil {
		t.Fatalf("expected error for invalid symbol")
	}
}
