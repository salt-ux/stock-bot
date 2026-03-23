package trading

import "time"

type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

type OrderRequest struct {
	Symbol string `json:"symbol"`
	Side   Side   `json:"side"`
	Qty    int64  `json:"qty"`
}

type Order struct {
	ID        int64     `json:"id"`
	Symbol    string    `json:"symbol"`
	Side      Side      `json:"side"`
	Qty       int64     `json:"qty"`
	FillPrice float64   `json:"fill_price"`
	FilledAt  time.Time `json:"filled_at"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
}

type Position struct {
	Symbol     string  `json:"symbol"`
	Qty        int64   `json:"qty"`
	AvgPrice   float64 `json:"avg_price"`
	LastPrice  float64 `json:"last_price"`
	Unrealized float64 `json:"unrealized_pnl"`
}

type State struct {
	Cash             float64    `json:"cash"`
	RealizedPnLToday float64    `json:"realized_pnl_today"`
	Positions        []Position `json:"positions"`
	RecentOrders     []Order    `json:"recent_orders"`
}
