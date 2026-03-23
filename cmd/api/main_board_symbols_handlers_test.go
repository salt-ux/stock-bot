package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/salt-ux/stock-bot/internal/board"
)

type memorySymbolStore struct {
	items      []board.SymbolRecord
	listErr    error
	replaceErr error
}

func (m *memorySymbolStore) List(_ context.Context) ([]board.SymbolRecord, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]board.SymbolRecord, len(m.items))
	copy(out, m.items)
	return out, nil
}

func (m *memorySymbolStore) ReplaceAll(_ context.Context, items []board.SymbolRecord) error {
	if m.replaceErr != nil {
		return m.replaceErr
	}
	next := make([]board.SymbolRecord, len(items))
	copy(next, items)
	m.items = next
	return nil
}

func (m *memorySymbolStore) Close() error {
	return nil
}

func TestBoardSymbolsHandlerGet(t *testing.T) {
	store := &memorySymbolStore{
		items: []board.SymbolRecord{
			{
				Symbol:        "AMCR",
				DisplayName:   "AMCR",
				PrincipalKRW:  4000000,
				SplitCount:    40,
				IsSelected:    true,
				ProgressState: board.ProgressStateRun,
				SellRatioPct:  15,
				TradeMethod:   "V2.2",
				NoteText:      "테스트",
				SortOrder:     0,
			},
		},
	}
	handler := boardSymbolsHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/board/symbols", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}

	var body struct {
		Items []board.SymbolRecord `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].Symbol != "AMCR" {
		t.Fatalf("unexpected items: %+v", body.Items)
	}
	if !body.Items[0].IsSelected {
		t.Fatalf("expected selected symbol")
	}
	if body.Items[0].ProgressState != board.ProgressStateRun {
		t.Fatalf("expected progress_state run, got: %s", body.Items[0].ProgressState)
	}
	if body.Items[0].SellRatioPct != 15 {
		t.Fatalf("expected sell_ratio_pct 15, got: %d", body.Items[0].SellRatioPct)
	}
}

func TestBoardSymbolsHandlerPut(t *testing.T) {
	store := &memorySymbolStore{}
	handler := boardSymbolsHandler(store)

	reqBody := []byte(`{"items":[{"symbol":"TSLA","display_name":"테슬라","principal_krw":5000000,"split_count":20,"is_selected":true,"progress_state":"RUN","sell_ratio_pct":25,"trade_method":"V2.2","note_text":"메모","sort_order":0}]}`)
	req := httptest.NewRequest(http.MethodPut, "/board/symbols", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}
	if len(store.items) != 1 || store.items[0].Symbol != "TSLA" {
		t.Fatalf("store not updated: %+v", store.items)
	}
	if !store.items[0].IsSelected {
		t.Fatalf("store selected flag not updated: %+v", store.items)
	}
	if store.items[0].ProgressState != board.ProgressStateRun {
		t.Fatalf("store progress state not updated: %+v", store.items)
	}
	if store.items[0].SellRatioPct != 25 {
		t.Fatalf("store sell ratio not updated: %+v", store.items)
	}
}

func TestBoardSymbolsHandlerRejectsInvalidJSON(t *testing.T) {
	store := &memorySymbolStore{}
	handler := boardSymbolsHandler(store)

	req := httptest.NewRequest(http.MethodPut, "/board/symbols", bytes.NewReader([]byte(`{"items":1}`)))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestBoardSymbolsHandlerPutReturnsBadRequestOnStoreError(t *testing.T) {
	store := &memorySymbolStore{replaceErr: errors.New("boom")}
	handler := boardSymbolsHandler(store)

	req := httptest.NewRequest(http.MethodPut, "/board/symbols", bytes.NewReader([]byte(`{"items":[]}`)))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}
}
