package trading

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
)

const (
	defaultV22SimSymbol          = "005930"
	defaultV22SimDisplayName     = "삼성전자"
	defaultV22SimDays            = 50
	defaultV22SimIntervalSeconds = 10
	defaultV22SimMinChangePct    = -3.0
	defaultV22SimMaxChangePct    = 10.0
	defaultV22SimBasePrice       = 70000.0
	defaultV22SimPrincipalKRW    = 4000000
	defaultV22SimSplitCount      = 40
)

type V22SimulationStartRequest struct {
	Symbol          string  `json:"symbol"`
	DisplayName     string  `json:"display_name"`
	Days            int     `json:"days"`
	IntervalSeconds int     `json:"interval_seconds"`
	MinChangePct    float64 `json:"min_change_pct"`
	MaxChangePct    float64 `json:"max_change_pct"`
	BasePrice       float64 `json:"base_price"`
	PrincipalKRW    int64   `json:"principal_krw"`
	SplitCount      int     `json:"split_count"`
	Seed            int64   `json:"seed"`
}

type V22SimulationTick struct {
	Step            int       `json:"step"`
	Day             int       `json:"day"`
	At              time.Time `json:"at"`
	Price           float64   `json:"price"`
	ChangePct       float64   `json:"change_pct"`
	TotalOrders     int       `json:"total_orders"`
	TotalBuyOrders  int       `json:"total_buy_orders"`
	TotalSellOrders int       `json:"total_sell_orders"`
	Message         string    `json:"message"`
}

type V22SimulationState struct {
	Running         bool                `json:"running"`
	Strategy        string              `json:"strategy"`
	Symbol          string              `json:"symbol"`
	DisplayName     string              `json:"display_name"`
	Days            int                 `json:"days"`
	IntervalSeconds int                 `json:"interval_seconds"`
	CurrentStep     int                 `json:"current_step"`
	CurrentDay      int                 `json:"current_day"`
	Seed            int64               `json:"seed"`
	StartedAt       time.Time           `json:"started_at,omitempty"`
	FinishedAt      time.Time           `json:"finished_at,omitempty"`
	PrincipalKRW    int64               `json:"principal_krw"`
	SplitCount      int                 `json:"split_count"`
	PlannedPrices   []float64           `json:"planned_prices,omitempty"`
	Ticks           []V22SimulationTick `json:"ticks,omitempty"`
	Error           string              `json:"error,omitempty"`
}

type V22SimulationRunner struct {
	market   *market.Service
	paper    *Service
	executor *BuyRuleExecutor

	mu      sync.RWMutex
	running bool
	cancel  context.CancelFunc
	state   V22SimulationState
	onTick  func(V22SimulationTick)
	onState func(V22SimulationState)
}

func NewV22SimulationRunner(marketSvc *market.Service, paperSvc *Service, executor *BuyRuleExecutor) *V22SimulationRunner {
	return &V22SimulationRunner{
		market:   marketSvc,
		paper:    paperSvc,
		executor: executor,
		state: V22SimulationState{
			Strategy: "v2.2",
		},
	}
}

func (r *V22SimulationRunner) SetHooks(onTick func(V22SimulationTick), onState func(V22SimulationState)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onTick = onTick
	r.onState = onState
}

func (r *V22SimulationRunner) Start(_ context.Context, req V22SimulationStartRequest) (V22SimulationState, error) {
	if r.market == nil || r.paper == nil || r.executor == nil {
		return V22SimulationState{}, fmt.Errorf("simulation dependencies are not initialized")
	}
	cfg, err := normalizeV22SimulationRequest(req)
	if err != nil {
		return V22SimulationState{}, err
	}
	prices, changes := generateV22SimulationPrices(cfg.Seed, cfg.BasePrice, cfg.Days, cfg.MinChangePct, cfg.MaxChangePct)

	r.mu.Lock()
	if r.running {
		state := cloneV22SimulationState(r.state)
		r.mu.Unlock()
		return state, fmt.Errorf("simulation is already running")
	}
	r.paper.Reset()
	runCtx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.running = true
	r.state = V22SimulationState{
		Running:         true,
		Strategy:        "v2.2",
		Symbol:          cfg.Symbol,
		DisplayName:     cfg.DisplayName,
		Days:            cfg.Days,
		IntervalSeconds: cfg.IntervalSeconds,
		CurrentStep:     0,
		CurrentDay:      0,
		Seed:            cfg.Seed,
		StartedAt:       time.Now().UTC(),
		PrincipalKRW:    cfg.PrincipalKRW,
		SplitCount:      cfg.SplitCount,
		PlannedPrices:   append([]float64(nil), prices...),
		Ticks:           make([]V22SimulationTick, 0, cfg.Days),
	}
	onState := r.onState
	state := cloneV22SimulationState(r.state)
	r.mu.Unlock()

	if onState != nil {
		onState(state)
	}
	go r.run(runCtx, cfg, prices, changes)
	return state, nil
}

func (r *V22SimulationRunner) Stop() V22SimulationState {
	r.mu.Lock()
	cancel := r.cancel
	r.cancel = nil
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return r.State()
}

func (r *V22SimulationRunner) State() V22SimulationState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneV22SimulationState(r.state)
}

func (r *V22SimulationRunner) run(ctx context.Context, cfg V22SimulationStartRequest, prices []float64, changes []float64) {
	simBaseAt := time.Now().UTC().Truncate(time.Second)

	for idx, price := range prices {
		select {
		case <-ctx.Done():
			r.finalizeStopped()
			return
		default:
		}

		step := idx + 1
		day := step
		at := simBaseAt.AddDate(0, 0, idx)

		err := r.market.SetQuoteOverride(cfg.Symbol, price, at)
		if err != nil {
			r.finalizeWithError(fmt.Errorf("set quote override: %w", err))
			return
		}

		result, execErr := r.executor.Execute(ctx, BuyRuleExecuteRequest{
			Items: []BuyRuleExecuteItem{
				{
					Symbol:       cfg.Symbol,
					DisplayName:  cfg.DisplayName,
					PrincipalKRW: cfg.PrincipalKRW,
					SplitCount:   cfg.SplitCount,
				},
			},
		})
		tick := V22SimulationTick{
			Step:      step,
			Day:       day,
			At:        at,
			Price:     price,
			ChangePct: changes[idx],
		}
		if execErr != nil {
			tick.Message = execErr.Error()
		} else {
			tick.TotalOrders = result.TotalOrders
			tick.TotalBuyOrders = result.TotalBuyOrders
			tick.TotalSellOrders = result.TotalSellOrders
			if len(result.Results) > 0 {
				tick.Message = result.Results[0].Message
			}
		}
		r.appendTick(tick)

		if step >= len(prices) {
			break
		}
		wait := time.Duration(cfg.IntervalSeconds) * time.Second
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			r.finalizeStopped()
			return
		case <-timer.C:
		}
	}

	r.finalizeCompleted()
}

func (r *V22SimulationRunner) appendTick(tick V22SimulationTick) {
	r.mu.Lock()
	r.state.CurrentStep = tick.Step
	r.state.CurrentDay = tick.Day
	r.state.Ticks = append(r.state.Ticks, tick)
	onTick := r.onTick
	onState := r.onState
	state := cloneV22SimulationState(r.state)
	r.mu.Unlock()

	if onTick != nil {
		onTick(tick)
	}
	if onState != nil {
		onState(state)
	}
}

func (r *V22SimulationRunner) finalizeCompleted() {
	r.mu.Lock()
	r.running = false
	r.cancel = nil
	r.state.Running = false
	r.state.FinishedAt = time.Now().UTC()
	onState := r.onState
	state := cloneV22SimulationState(r.state)
	r.mu.Unlock()

	if onState != nil {
		onState(state)
	}
}

func (r *V22SimulationRunner) finalizeStopped() {
	r.mu.Lock()
	r.running = false
	r.cancel = nil
	r.state.Running = false
	if r.state.Error == "" {
		r.state.Error = "stopped"
	}
	r.state.FinishedAt = time.Now().UTC()
	onState := r.onState
	state := cloneV22SimulationState(r.state)
	r.mu.Unlock()

	if onState != nil {
		onState(state)
	}
}

func (r *V22SimulationRunner) finalizeWithError(err error) {
	r.mu.Lock()
	r.running = false
	r.cancel = nil
	r.state.Running = false
	r.state.Error = err.Error()
	r.state.FinishedAt = time.Now().UTC()
	onState := r.onState
	state := cloneV22SimulationState(r.state)
	r.mu.Unlock()

	if onState != nil {
		onState(state)
	}
}

func normalizeV22SimulationRequest(req V22SimulationStartRequest) (V22SimulationStartRequest, error) {
	symbol := strings.ToUpper(strings.TrimSpace(req.Symbol))
	if symbol == "" {
		symbol = defaultV22SimSymbol
	}
	if !isValidV22SimSymbol(symbol) {
		return V22SimulationStartRequest{}, fmt.Errorf("invalid symbol: %s", req.Symbol)
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = defaultV22SimDisplayName
	}
	runes := []rune(displayName)
	if len(runes) > 120 {
		displayName = strings.TrimSpace(string(runes[:120]))
		if displayName == "" {
			displayName = symbol
		}
	}

	days := req.Days
	if days <= 0 {
		days = defaultV22SimDays
	}
	if days > 365 {
		days = 365
	}
	intervalSeconds := req.IntervalSeconds
	if intervalSeconds <= 0 {
		intervalSeconds = defaultV22SimIntervalSeconds
	}
	if intervalSeconds > 3600 {
		intervalSeconds = 3600
	}

	minChangePct := req.MinChangePct
	maxChangePct := req.MaxChangePct
	if minChangePct == 0 && maxChangePct == 0 {
		minChangePct = defaultV22SimMinChangePct
		maxChangePct = defaultV22SimMaxChangePct
	}
	if minChangePct > maxChangePct {
		minChangePct, maxChangePct = maxChangePct, minChangePct
	}
	if maxChangePct > 50 {
		maxChangePct = 50
	}
	if minChangePct < -50 {
		minChangePct = -50
	}

	basePrice := req.BasePrice
	if basePrice <= 0 {
		basePrice = defaultV22SimBasePrice
	}
	if basePrice < 100 {
		basePrice = 100
	}

	principal := normalizeRulePrincipal(req.PrincipalKRW)
	if principal < 1 {
		principal = defaultV22SimPrincipalKRW
	}
	splitCount := normalizeRuleSplitCount(req.SplitCount)
	if splitCount <= 0 {
		splitCount = defaultV22SimSplitCount
	}

	seed := req.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	return V22SimulationStartRequest{
		Symbol:          symbol,
		DisplayName:     displayName,
		Days:            days,
		IntervalSeconds: intervalSeconds,
		MinChangePct:    minChangePct,
		MaxChangePct:    maxChangePct,
		BasePrice:       basePrice,
		PrincipalKRW:    principal,
		SplitCount:      splitCount,
		Seed:            seed,
	}, nil
}

func generateV22SimulationPrices(seed int64, basePrice float64, days int, minChangePct float64, maxChangePct float64) ([]float64, []float64) {
	rng := rand.New(rand.NewSource(seed))
	prices := make([]float64, 0, days)
	changes := make([]float64, 0, days)

	current := math.Max(basePrice, 100)
	prices = append(prices, roundPrice(current))
	changes = append(changes, 0)

	for i := 1; i < days; i++ {
		change := minChangePct + rng.Float64()*(maxChangePct-minChangePct)
		current = current * (1 + (change / 100.0))
		if current < 100 {
			current = 100
		}
		prices = append(prices, roundPrice(current))
		changes = append(changes, roundPercent(change))
	}
	return prices, changes
}

func isValidV22SimSymbol(symbol string) bool {
	if symbol == "" || len(symbol) > 12 {
		return false
	}
	for _, r := range symbol {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

func roundPrice(v float64) float64 {
	return math.Round(v)
}

func roundPercent(v float64) float64 {
	return math.Round(v*100) / 100
}

func cloneV22SimulationState(src V22SimulationState) V22SimulationState {
	dst := src
	if src.PlannedPrices != nil {
		dst.PlannedPrices = append([]float64(nil), src.PlannedPrices...)
	}
	if src.Ticks != nil {
		dst.Ticks = append([]V22SimulationTick(nil), src.Ticks...)
	}
	return dst
}
