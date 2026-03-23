package strategy

import "time"

type SignalAction string

const (
	SignalBuy  SignalAction = "BUY"
	SignalSell SignalAction = "SELL"
	SignalHold SignalAction = "HOLD"
)

type Signal struct {
	Action   SignalAction       `json:"action"`
	Price    float64            `json:"price"`
	At       time.Time          `json:"at"`
	Reason   string             `json:"reason"`
	Metadata map[string]float64 `json:"metadata,omitempty"`
}
