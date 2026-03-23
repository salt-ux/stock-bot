package strategy

import (
	"context"
	"fmt"

	"github.com/salt-ux/stock-bot/internal/market"
)

type Engine struct {
	marketService *market.Service
}

type RunResult struct {
	Strategy string         `json:"strategy"`
	Symbol   string         `json:"symbol"`
	Signal   Signal         `json:"signal"`
	Candles  int            `json:"candles"`
	Interval string         `json:"interval"`
	Meta     map[string]any `json:"meta,omitempty"`
}

func NewEngine(marketService *market.Service) *Engine {
	return &Engine{marketService: marketService}
}

func (e *Engine) Run(ctx context.Context, symbol, interval string, limit int, s Strategy) (RunResult, error) {
	candles, err := e.marketService.Candles(ctx, symbol, interval, limit)
	if err != nil {
		return RunResult{}, fmt.Errorf("load candles: %w", err)
	}

	signal, err := s.Evaluate(candles)
	if err != nil {
		return RunResult{}, fmt.Errorf("evaluate strategy: %w", err)
	}

	return RunResult{
		Strategy: s.Name(),
		Symbol:   symbol,
		Signal:   signal,
		Candles:  len(candles),
		Interval: interval,
	}, nil
}
