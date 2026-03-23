package trading

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
)

const (
	defaultBuyRulePrincipalKRW int64   = 4000000
	defaultBuyRuleSplitCount   int     = 40
	minBuyRuleSplitCount       int     = 10
	maxBuyRuleSplitCount       int     = 50
	maxBuyRulePrincipalKRW     int64   = 1000000000000
	earlyStageThresholdPct     float64 = 50.0
	baseTargetPct              float64 = 10.0
	maxCapFromCurrentPct       float64 = 15.0
	sellGeneralTargetPct       float64 = 10.0
)

type BuyRuleExecuteItem struct {
	Symbol       string `json:"symbol"`
	DisplayName  string `json:"display_name,omitempty"`
	PrincipalKRW int64  `json:"principal_krw"`
	SplitCount   int    `json:"split_count"`
}

type BuyRuleExecuteRequest struct {
	Items []BuyRuleExecuteItem `json:"items"`
}

type BuyRuleSymbolResult struct {
	Symbol            string  `json:"symbol"`
	DisplayName       string  `json:"display_name,omitempty"`
	Phase             string  `json:"phase"`
	QuotePrice        float64 `json:"quote_price"`
	AvgPrice          float64 `json:"avg_price"`
	PositionQty       int64   `json:"position_qty"`
	BuyCount          int     `json:"buy_count"`
	PrincipalKRW      int64   `json:"principal_krw"`
	AppliedPrincipal  int64   `json:"applied_principal_krw"`
	UsedPrincipalKRW  int64   `json:"used_principal_krw"`
	RemainingKRW      int64   `json:"remaining_krw"`
	SplitCount        int     `json:"split_count"`
	PerSplitBudgetKRW int64   `json:"per_split_budget_krw"`
	UtilizationPct    float64 `json:"utilization_pct"`
	DynamicTargetPct  float64 `json:"dynamic_target_pct"`
	BuyTargetAvg      float64 `json:"buy_target_avg"`
	BuyTargetDynamic  float64 `json:"buy_target_dynamic"`
	SellTargetGeneral float64 `json:"sell_target_general"`
	SellTargetDynamic float64 `json:"sell_target_dynamic"`
	PlannedBuyQty     int64   `json:"planned_buy_qty"`
	PlannedSellQty    int64   `json:"planned_sell_qty"`
	ExecutedOrders    []Order `json:"executed_orders,omitempty"`
	Message           string  `json:"message"`
}

type BuyRuleExecuteResult struct {
	ExecutedAt      time.Time             `json:"executed_at"`
	TotalSymbols    int                   `json:"total_symbols"`
	TotalOrders     int                   `json:"total_orders"`
	TotalBuyOrders  int                   `json:"total_buy_orders"`
	TotalSellOrders int                   `json:"total_sell_orders"`
	Results         []BuyRuleSymbolResult `json:"results"`
}

type BuyRuleExecutor struct {
	paper  *Service
	market *market.Service
}

func NewBuyRuleExecutor(paper *Service, marketSvc *market.Service) *BuyRuleExecutor {
	return &BuyRuleExecutor{
		paper:  paper,
		market: marketSvc,
	}
}

func (e *BuyRuleExecutor) Execute(ctx context.Context, req BuyRuleExecuteRequest) (BuyRuleExecuteResult, error) {
	if e.paper == nil || e.market == nil {
		return BuyRuleExecuteResult{}, fmt.Errorf("executor dependencies are not initialized")
	}
	if len(req.Items) == 0 {
		return BuyRuleExecuteResult{}, fmt.Errorf("items are required")
	}

	items := normalizeBuyRuleItems(req.Items)
	if len(items) == 0 {
		return BuyRuleExecuteResult{}, fmt.Errorf("valid items are required")
	}

	result := BuyRuleExecuteResult{
		ExecutedAt:   time.Now().UTC(),
		TotalSymbols: len(items),
		Results:      make([]BuyRuleSymbolResult, 0, len(items)),
	}

	for _, item := range items {
		state := e.paper.GetState()
		pos := findPositionBySymbol(state.Positions, item.Symbol)
		orderStats := summarizeSymbolOrderStats(state.RecentOrders, item.Symbol)
		buyCount := orderStats.BuyCountSinceLastSell

		quote, err := e.market.Quote(ctx, item.Symbol)
		if err != nil {
			result.Results = append(result.Results, BuyRuleSymbolResult{
				Symbol:      item.Symbol,
				DisplayName: item.DisplayName,
				Message:     fmt.Sprintf("시세 조회 실패: %v", err),
			})
			continue
		}
		if quote.Price <= 0 {
			result.Results = append(result.Results, BuyRuleSymbolResult{
				Symbol:      item.Symbol,
				DisplayName: item.DisplayName,
				Message:     "유효한 현재가를 가져오지 못했습니다.",
			})
			continue
		}

		symbolResult := evaluateBuyRule(item, pos, buyCount, quote.Price, orderStats.RealizedPnLKRW)

		// 실무적으로는 지정가/예약주문이 필요하지만 현재 엔진은 즉시 체결만 지원하므로
		// 조건 충족 시 시장가 체결로 주문을 실행합니다.
		if symbolResult.PlannedSellQty > 0 {
			order, err := e.paper.PlaceOrder(ctx, OrderRequest{
				Symbol: item.Symbol,
				Side:   SideSell,
				Qty:    symbolResult.PlannedSellQty,
			})
			if err == nil {
				symbolResult.ExecutedOrders = append(symbolResult.ExecutedOrders, order)
				result.TotalOrders++
				result.TotalSellOrders++
			} else {
				symbolResult.Message = fmt.Sprintf("매도 실행 실패: %v", err)
			}
		} else if symbolResult.PlannedBuyQty > 0 {
			order, err := e.paper.PlaceOrder(ctx, OrderRequest{
				Symbol: item.Symbol,
				Side:   SideBuy,
				Qty:    symbolResult.PlannedBuyQty,
			})
			if err == nil {
				symbolResult.ExecutedOrders = append(symbolResult.ExecutedOrders, order)
				result.TotalOrders++
				result.TotalBuyOrders++
			} else {
				symbolResult.Message = fmt.Sprintf("매수 실행 실패: %v", err)
			}
		}

		if symbolResult.Message == "" {
			if len(symbolResult.ExecutedOrders) == 0 {
				symbolResult.Message = "조건 미충족으로 주문 없음"
			} else {
				symbolResult.Message = "규칙 기반 주문 실행 완료"
			}
		}
		result.Results = append(result.Results, symbolResult)
	}

	return result, nil
}

func normalizeBuyRuleItems(items []BuyRuleExecuteItem) []BuyRuleExecuteItem {
	seen := make(map[string]struct{}, len(items))
	out := make([]BuyRuleExecuteItem, 0, len(items))
	for _, item := range items {
		symbol := normalizeRuleSymbol(item.Symbol)
		if symbol == "" {
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		out = append(out, BuyRuleExecuteItem{
			Symbol:       symbol,
			DisplayName:  strings.TrimSpace(item.DisplayName),
			PrincipalKRW: normalizeRulePrincipal(item.PrincipalKRW),
			SplitCount:   normalizeRuleSplitCount(item.SplitCount),
		})
	}
	return out
}

func normalizeRuleSymbol(value string) string {
	s := strings.ToUpper(strings.TrimSpace(value))
	if s == "" {
		return ""
	}
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-':
		default:
			return ""
		}
	}
	return s
}

func normalizeRulePrincipal(value int64) int64 {
	if value < 1 {
		return defaultBuyRulePrincipalKRW
	}
	if value > maxBuyRulePrincipalKRW {
		return maxBuyRulePrincipalKRW
	}
	return value
}

func normalizeRuleSplitCount(value int) int {
	if value < minBuyRuleSplitCount {
		return minBuyRuleSplitCount
	}
	if value > maxBuyRuleSplitCount {
		return maxBuyRuleSplitCount
	}
	return value
}

func evaluateBuyRule(item BuyRuleExecuteItem, pos Position, buyCount int, quotePrice float64, realizedPnLKRW float64) BuyRuleSymbolResult {
	appliedPrincipal := float64(item.PrincipalKRW) + realizedPnLKRW
	if appliedPrincipal < 1 {
		appliedPrincipal = 1
	}
	avgPrice := pos.AvgPrice
	if avgPrice <= 0 {
		avgPrice = quotePrice
	}

	usedPrincipal := 0.0
	if pos.Qty > 0 && avgPrice > 0 {
		usedPrincipal = avgPrice * float64(pos.Qty)
	}
	remainingPrincipal := appliedPrincipal - usedPrincipal
	utilizationPct := 0.0
	if appliedPrincipal > 0 {
		utilizationPct = (usedPrincipal / appliedPrincipal) * 100
	}

	perSplitBudget := int64(math.Floor(appliedPrincipal / float64(item.SplitCount)))
	if perSplitBudget < 1 {
		perSplitBudget = 1
	}

	dynamicPct := baseTargetPct - (float64(buyCount) / 2.0)
	if dynamicPct < 0 {
		dynamicPct = 0
	}

	capPrice := quotePrice * (1 + maxCapFromCurrentPct/100.0)
	buyTargetAvg := math.Min(avgPrice, capPrice)
	buyTargetDynamic := math.Min(avgPrice*(1+dynamicPct/100.0), capPrice)
	sellTargetGeneral := avgPrice * (1 + sellGeneralTargetPct/100.0)
	sellTargetDynamic := avgPrice * (1 + dynamicPct/100.0)

	phase := "전반전"
	earlyStage := utilizationPct < earlyStageThresholdPct
	if !earlyStage {
		phase = "후반전"
	}

	desiredBuyBudget := 0.0
	if earlyStage {
		if quotePrice <= buyTargetAvg {
			desiredBuyBudget += float64(perSplitBudget) * 0.5
		}
		if quotePrice <= buyTargetDynamic {
			desiredBuyBudget += float64(perSplitBudget) * 0.5
		}
	} else {
		if quotePrice <= buyTargetDynamic {
			desiredBuyBudget += float64(perSplitBudget)
		}
	}
	if remainingPrincipal < 0 {
		remainingPrincipal = 0
	}
	buyBudget := math.Min(desiredBuyBudget, remainingPrincipal)
	buyQty := budgetToQty(buyBudget, quotePrice)

	sellQty := int64(0)
	if pos.Qty > 0 {
		quota75 := int64(math.Floor(float64(pos.Qty) * 0.75))
		if quota75 < 1 {
			quota75 = 1
		}
		quota25 := pos.Qty - quota75
		if quota25 < 0 {
			quota25 = 0
		}
		if quotePrice >= sellTargetGeneral {
			sellQty += quota75
		}
		if quotePrice >= sellTargetDynamic {
			sellQty += quota25
		}
		if sellQty > pos.Qty {
			sellQty = pos.Qty
		}
	}

	// 동일 시점에서 매도 조건이 맞으면 매수를 중단합니다.
	if sellQty > 0 {
		buyQty = 0
	}

	return BuyRuleSymbolResult{
		Symbol:            item.Symbol,
		DisplayName:       item.DisplayName,
		Phase:             phase,
		QuotePrice:        quotePrice,
		AvgPrice:          avgPrice,
		PositionQty:       pos.Qty,
		BuyCount:          buyCount,
		PrincipalKRW:      item.PrincipalKRW,
		AppliedPrincipal:  int64(math.Round(appliedPrincipal)),
		UsedPrincipalKRW:  int64(math.Round(usedPrincipal)),
		RemainingKRW:      int64(math.Round(appliedPrincipal - usedPrincipal)),
		SplitCount:        item.SplitCount,
		PerSplitBudgetKRW: perSplitBudget,
		UtilizationPct:    utilizationPct,
		DynamicTargetPct:  dynamicPct,
		BuyTargetAvg:      buyTargetAvg,
		BuyTargetDynamic:  buyTargetDynamic,
		SellTargetGeneral: sellTargetGeneral,
		SellTargetDynamic: sellTargetDynamic,
		PlannedBuyQty:     buyQty,
		PlannedSellQty:    sellQty,
	}
}

func budgetToQty(budget float64, price float64) int64 {
	if budget <= 0 || price <= 0 {
		return 0
	}
	qty := int64(math.Floor(budget / price))
	if qty < 1 {
		return 0
	}
	return qty
}

func findPositionBySymbol(positions []Position, symbol string) Position {
	for _, p := range positions {
		if strings.EqualFold(strings.TrimSpace(p.Symbol), symbol) {
			return p
		}
	}
	return Position{Symbol: symbol}
}

type symbolOrderStats struct {
	BuyCountSinceLastSell int
	RealizedPnLKRW        float64
}

func summarizeSymbolOrderStats(orders []Order, symbol string) symbolOrderStats {
	stats := symbolOrderStats{}
	sawSell := false
	symbolOrders := make([]Order, 0, len(orders))
	for _, o := range orders {
		if !strings.EqualFold(strings.TrimSpace(o.Symbol), symbol) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(o.Status), "FILLED") {
			continue
		}
		symbolOrders = append(symbolOrders, o)
		switch o.Side {
		case SideBuy:
			if !sawSell {
				stats.BuyCountSinceLastSell++
			}
		case SideSell:
			sawSell = true
		}
	}
	// Realized PnL is evaluated in fill sequence order (oldest -> newest).
	openQty := int64(0)
	openAvg := 0.0
	for idx := len(symbolOrders) - 1; idx >= 0; idx-- {
		o := symbolOrders[idx]
		switch o.Side {
		case SideBuy:
			if o.Qty <= 0 || o.FillPrice <= 0 {
				continue
			}
			totalCost := openAvg*float64(openQty) + o.FillPrice*float64(o.Qty)
			openQty += o.Qty
			if openQty > 0 {
				openAvg = totalCost / float64(openQty)
			}
		case SideSell:
			if o.Qty <= 0 || o.FillPrice <= 0 || openQty <= 0 {
				continue
			}
			sellQty := o.Qty
			if sellQty > openQty {
				sellQty = openQty
			}
			stats.RealizedPnLKRW += (o.FillPrice - openAvg) * float64(sellQty)
			openQty -= sellQty
			if openQty <= 0 {
				openQty = 0
				openAvg = 0
			}
		}
	}
	return stats
}
