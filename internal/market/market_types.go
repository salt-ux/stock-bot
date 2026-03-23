package market

import "time"

type Quote struct {
	Symbol string    `json:"symbol"`
	Price  float64   `json:"price"`
	AsOf   time.Time `json:"as_of"`
}

type Candle struct {
	Symbol   string    `json:"symbol"`
	Interval string    `json:"interval"`
	Time     time.Time `json:"time"`
	Open     float64   `json:"open"`
	High     float64   `json:"high"`
	Low      float64   `json:"low"`
	Close    float64   `json:"close"`
	Volume   int64     `json:"volume"`
}
