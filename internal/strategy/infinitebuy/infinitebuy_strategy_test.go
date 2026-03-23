package infinitebuy

import (
	"testing"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
	"github.com/salt-ux/stock-bot/internal/strategy"
)

func TestInfiniteBuySignals(t *testing.T) {
	c := []market.Candle{{Symbol: "005930", Interval: "1d", Time: time.Now().UTC(), Close: 110}}

	s, _ := New(10, 100, 4000000)
	sig, err := s.Evaluate(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig.Action != strategy.SignalSell {
		t.Fatalf("expected SELL for +10%% before 20 buys, got %s", sig.Action)
	}

	s, _ = New(25, 100, 4000000)
	sig, _ = s.Evaluate(c)
	if sig.Action != strategy.SignalSell {
		t.Fatalf("expected SELL for +5%% after 20 buys, got %s", sig.Action)
	}

	s, _ = New(40, 120, 4000000)
	sig, _ = s.Evaluate(c)
	if sig.Action != strategy.SignalSell {
		t.Fatalf("expected SELL for negative after 40 buys, got %s", sig.Action)
	}

	s, _ = New(5, 120, 4000000)
	sig, _ = s.Evaluate(c)
	if sig.Action != strategy.SignalBuy {
		t.Fatalf("expected BUY dca signal, got %s", sig.Action)
	}
}
