package risk

import "testing"

func TestValidateBuyOverMaxPosition(t *testing.T) {
	cfg := Config{MaxPositionNotional: 1000, DailyLossLimit: 500}
	err := cfg.Validate(OrderIntent{Side: "BUY", Quantity: 11}, Snapshot{CurrentQty: 0, RealizedPnLToday: 0}, 100)
	if err == nil {
		t.Fatal("expected max position error")
	}
}

func TestValidateDailyLoss(t *testing.T) {
	cfg := Config{MaxPositionNotional: 100000, DailyLossLimit: 500}
	err := cfg.Validate(OrderIntent{Side: "BUY", Quantity: 1}, Snapshot{CurrentQty: 0, RealizedPnLToday: -500}, 100)
	if err == nil {
		t.Fatal("expected daily loss error")
	}
}

func TestValidateSellQuantity(t *testing.T) {
	cfg := Config{MaxPositionNotional: 100000, DailyLossLimit: 500}
	err := cfg.Validate(OrderIntent{Side: "SELL", Quantity: 2}, Snapshot{CurrentQty: 1, RealizedPnLToday: 0}, 100)
	if err == nil {
		t.Fatal("expected insufficient quantity error")
	}
}
