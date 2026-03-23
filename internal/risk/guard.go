package risk

import "fmt"

type Config struct {
	MaxPositionNotional float64
	DailyLossLimit      float64
}

type Snapshot struct {
	CurrentQty       int64
	RealizedPnLToday float64
}

type OrderIntent struct {
	Side     string
	Quantity int64
}

func (c Config) Validate(intent OrderIntent, snap Snapshot, fillPrice float64) error {
	if intent.Quantity <= 0 {
		return fmt.Errorf("quantity must be > 0")
	}
	if fillPrice <= 0 {
		return fmt.Errorf("fill price must be > 0")
	}
	if c.DailyLossLimit > 0 && snap.RealizedPnLToday <= -c.DailyLossLimit {
		return fmt.Errorf("daily loss limit exceeded")
	}

	switch intent.Side {
	case "BUY":
		projectedQty := snap.CurrentQty + intent.Quantity
		projectedNotional := float64(projectedQty) * fillPrice
		if projectedNotional > c.MaxPositionNotional {
			return fmt.Errorf("max position notional exceeded")
		}
	case "SELL":
		if intent.Quantity > snap.CurrentQty {
			return fmt.Errorf("insufficient quantity for sell")
		}
	default:
		return fmt.Errorf("unsupported side: %s", intent.Side)
	}

	return nil
}
