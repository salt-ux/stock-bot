package main

import (
	"net/http"

	"github.com/salt-ux/stock-bot/internal/trading"
)

func paperBuyRuleExecuteHandler(
	executor *trading.BuyRuleExecutor,
	paperSvc *trading.Service,
	eventsHub *sseEventsHub,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if executor == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "규칙 실행기가 준비되지 않았습니다"})
			return
		}

		var req trading.BuyRuleExecuteRequest
		if err := decodeJSON(r.Body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "요청 형식이 올바르지 않습니다"})
			return
		}

		result, err := executor.Execute(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}

		if eventsHub != nil && paperSvc != nil {
			eventsHub.Publish("buyrule_result", result)
			eventsHub.Publish("paper_state", paperSvc.GetState())
		}
		writeJSON(w, http.StatusOK, result)
	}
}
