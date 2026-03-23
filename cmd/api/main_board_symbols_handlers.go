package main

import (
	"net/http"

	"github.com/salt-ux/stock-bot/internal/board"
)

type boardSymbolsReplaceRequest struct {
	Items []board.SymbolRecord `json:"items"`
}

func boardSymbolsHandler(store board.SymbolStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "종목 저장소가 설정되지 않았습니다"})
			return
		}

		switch r.Method {
		case http.MethodGet:
			items, err := store.List(r.Context())
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "종목 목록을 불러오지 못했습니다"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"items": items})
		case http.MethodPut:
			var req boardSymbolsReplaceRequest
			if err := decodeJSON(r.Body, &req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"message": "요청 형식이 올바르지 않습니다"})
				return
			}
			if req.Items == nil {
				req.Items = []board.SymbolRecord{}
			}
			if err := store.ReplaceAll(r.Context(), req.Items); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"message": "종목 목록 저장에 실패했습니다: " + err.Error()})
				return
			}

			items, err := store.List(r.Context())
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "저장 후 종목 목록을 불러오지 못했습니다"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"items": items})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
