package main

import (
	"bytes"
	"io"
	"net/http"

	"github.com/salt-ux/stock-bot/internal/trading"
)

func paperV22SimulationStartHandler(runner *trading.V22SimulationRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if runner == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "시뮬레이션 실행기가 준비되지 않았습니다"})
			return
		}

		var req trading.V22SimulationStartRequest
		if err := decodeOptionalJSON(r.Body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "요청 형식이 올바르지 않습니다"})
			return
		}
		state, err := runner.Start(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, state)
	}
}

func paperV22SimulationStateHandler(runner *trading.V22SimulationRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if runner == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "시뮬레이션 실행기가 준비되지 않았습니다"})
			return
		}
		writeJSON(w, http.StatusOK, runner.State())
	}
}

func paperV22SimulationStopHandler(runner *trading.V22SimulationRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if runner == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "시뮬레이션 실행기가 준비되지 않았습니다"})
			return
		}
		state := runner.Stop()
		writeJSON(w, http.StatusOK, state)
	}
}

func decodeOptionalJSON(body io.Reader, v any) error {
	raw, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	return decodeJSON(bytes.NewReader(raw), v)
}
