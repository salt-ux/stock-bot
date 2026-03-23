package infinitebuy

import (
	"fmt"
	"sort"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
	"github.com/salt-ux/stock-bot/internal/strategy"
)

const TotalSplits = 40

type Strategy struct {
	BuyCount   int
	AvgPrice   float64
	Allocation float64
}

func New(buyCount int, avgPrice float64, allocation float64) (*Strategy, error) {
	if buyCount < 0 || buyCount > TotalSplits {
		return nil, fmt.Errorf("buy_count must be between 0 and %d", TotalSplits)
	}
	if avgPrice < 0 {
		return nil, fmt.Errorf("avg_price must be >= 0")
	}
	if buyCount > 0 && avgPrice <= 0 {
		return nil, fmt.Errorf("avg_price must be > 0 when buy_count > 0")
	}
	if allocation < 0 {
		return nil, fmt.Errorf("allocation must be >= 0")
	}
	return &Strategy{BuyCount: buyCount, AvgPrice: avgPrice, Allocation: allocation}, nil
}

func (s *Strategy) Name() string {
	return "infinite_buy"
}

func (s *Strategy) Evaluate(candles []market.Candle) (strategy.Signal, error) {
	if len(candles) == 0 {
		return strategy.Signal{}, fmt.Errorf("need at least 1 candle")
	}
	ordered := append([]market.Candle(nil), candles...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Time.Before(ordered[j].Time)
	})
	last := ordered[len(ordered)-1]
	at := normalizeTime(last.Time)
	price := last.Close

	gain := 0.0
	if s.AvgPrice > 0 {
		gain = (price - s.AvgPrice) / s.AvgPrice
	}

	meta := map[string]float64{
		"buy_count":  float64(s.BuyCount),
		"avg_price":  s.AvgPrice,
		"gain_ratio": gain,
	}
	if s.Allocation > 0 {
		meta["allocation"] = s.Allocation
		meta["per_order_budget"] = s.Allocation / TotalSplits
	}

	sig := strategy.Signal{
		Action:   strategy.SignalHold,
		Price:    price,
		At:       at,
		Reason:   "conditions not met",
		Metadata: meta,
	}

	if s.BuyCount > 0 && s.BuyCount < 20 && gain >= 0.10 {
		sig.Action = strategy.SignalSell
		sig.Reason = "take profit 10% before 20 buys"
		return sig, nil
	}
	if s.BuyCount >= 20 && s.BuyCount < TotalSplits && gain >= 0.05 {
		sig.Action = strategy.SignalSell
		sig.Reason = "take profit 5% after 20 buys"
		return sig, nil
	}
	if s.BuyCount >= TotalSplits && gain < 0 {
		sig.Action = strategy.SignalSell
		sig.Reason = "stop after 40 buys and still negative"
		return sig, nil
	}

	if s.BuyCount < TotalSplits {
		sig.Action = strategy.SignalBuy
		sig.Reason = "daily dca buy step"
		return sig, nil
	}

	return sig, nil
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}
