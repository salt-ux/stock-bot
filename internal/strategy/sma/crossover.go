package sma

import (
	"fmt"
	"sort"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
	"github.com/salt-ux/stock-bot/internal/strategy"
)

type Crossover struct {
	shortWindow int
	longWindow  int
}

func NewCrossover(shortWindow, longWindow int) (*Crossover, error) {
	if shortWindow < 2 || longWindow < 2 {
		return nil, fmt.Errorf("window must be >= 2")
	}
	if shortWindow >= longWindow {
		return nil, fmt.Errorf("short window must be less than long window")
	}
	return &Crossover{shortWindow: shortWindow, longWindow: longWindow}, nil
}

func (c *Crossover) Name() string {
	return fmt.Sprintf("sma_crossover_%d_%d", c.shortWindow, c.longWindow)
}

func (c *Crossover) Evaluate(candles []market.Candle) (strategy.Signal, error) {
	sorted := append([]market.Candle(nil), candles...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Time.Before(sorted[j].Time)
	})

	if len(sorted) < c.longWindow+1 {
		return strategy.Signal{}, fmt.Errorf("need at least %d candles", c.longWindow+1)
	}

	prev := sorted[:len(sorted)-1]
	curr := sorted

	prevShort := sma(prev, c.shortWindow)
	prevLong := sma(prev, c.longWindow)
	currShort := sma(curr, c.shortWindow)
	currLong := sma(curr, c.longWindow)

	last := sorted[len(sorted)-1]
	sig := strategy.Signal{
		Action: strategy.SignalHold,
		Price:  last.Close,
		At:     normalizeTime(last.Time),
		Reason: "no crossover",
		Metadata: map[string]float64{
			"short_sma": currShort,
			"long_sma":  currLong,
		},
	}

	if prevShort <= prevLong && currShort > currLong {
		sig.Action = strategy.SignalBuy
		sig.Reason = "golden cross"
		return sig, nil
	}
	if prevShort >= prevLong && currShort < currLong {
		sig.Action = strategy.SignalSell
		sig.Reason = "dead cross"
		return sig, nil
	}

	return sig, nil
}

func sma(candles []market.Candle, window int) float64 {
	start := len(candles) - window
	total := 0.0
	for _, c := range candles[start:] {
		total += c.Close
	}
	return total / float64(window)
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}
