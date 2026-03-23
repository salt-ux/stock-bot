package board

import "context"

const (
	DefaultPrincipalKRW int64 = 4000000
	DefaultSplitCount   int   = 40
	MinSplitCount       int   = 10
	MaxSplitCount       int   = 50

	ProgressStateWait = "WAIT"
	ProgressStateRun  = "RUN"

	DefaultProgressState = ProgressStateWait
	DefaultSellRatioPct  = 10
	MinSellRatioPct      = 0
	MaxSellRatioPct      = 100
	DefaultTradeMethod   = "V2.2"
)

type SymbolRecord struct {
	Symbol        string `json:"symbol"`
	DisplayName   string `json:"display_name"`
	PrincipalKRW  int64  `json:"principal_krw"`
	SplitCount    int    `json:"split_count"`
	IsSelected    bool   `json:"is_selected"`
	ProgressState string `json:"progress_state"`
	SellRatioPct  int    `json:"sell_ratio_pct"`
	TradeMethod   string `json:"trade_method"`
	NoteText      string `json:"note_text"`
	SortOrder     int    `json:"sort_order"`
}

type SymbolStore interface {
	List(ctx context.Context) ([]SymbolRecord, error)
	ReplaceAll(ctx context.Context, items []SymbolRecord) error
	Close() error
}
