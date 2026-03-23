package scheduler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/salt-ux/stock-bot/internal/config"
	"github.com/salt-ux/stock-bot/internal/strategy"
	"github.com/salt-ux/stock-bot/internal/strategy/infinitebuy"
	"github.com/salt-ux/stock-bot/internal/strategy/sma"
	"github.com/salt-ux/stock-bot/internal/trading"
)

type strategyEngineRunner interface {
	Run(ctx context.Context, symbol, interval string, limit int, s strategy.Strategy) (strategy.RunResult, error)
}

type paperOrderExecutor interface {
	PlaceOrder(ctx context.Context, req trading.OrderRequest) (trading.Order, error)
}

type AutoTradeJobResult struct {
	At          time.Time `json:"at"`
	Strategy    string    `json:"strategy"`
	Signal      string    `json:"signal"`
	Reason      string    `json:"reason"`
	OrderPlaced bool      `json:"order_placed"`
	OrderID     int64     `json:"order_id,omitempty"`
	Error       string    `json:"error,omitempty"`
}

type CronSchedulerSnapshot struct {
	Enabled         bool                `json:"enabled"`
	Running         bool                `json:"running"`
	Location        string              `json:"location"`
	MarketOpenCron  string              `json:"market_open_cron"`
	AutoTradeCron   string              `json:"auto_trade_cron"`
	AutoTradeSymbol string              `json:"auto_trade_symbol"`
	AutoTradeNextAt time.Time           `json:"auto_trade_next_at,omitempty"`
	LastAutoTrade   *AutoTradeJobResult `json:"last_auto_trade,omitempty"`
}

type CronSchedulerRunner struct {
	cfg      config.SchedulerConfig
	engine   strategyEngineRunner
	trader   paperOrderExecutor
	location *time.Location
	cron     *cron.Cron

	mu             sync.Mutex
	running        bool
	autoTradeEntry cron.EntryID
	lastAutoTrade  *AutoTradeJobResult
	onAutoTrade    func(AutoTradeJobResult)
}

func NewCronSchedulerRunner(cfg config.SchedulerConfig, engine strategyEngineRunner, trader paperOrderExecutor) (*CronSchedulerRunner, error) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load scheduler timezone: %w", err)
	}

	runner := &CronSchedulerRunner{
		cfg:      cfg,
		engine:   engine,
		trader:   trader,
		location: loc,
		cron:     cron.New(cron.WithLocation(loc)),
	}
	if !cfg.Enabled {
		return runner, nil
	}
	if engine == nil {
		return nil, fmt.Errorf("strategy engine is required")
	}
	if trader == nil {
		return nil, fmt.Errorf("paper trader is required")
	}

	if _, err := runner.cron.AddFunc(cfg.MarketOpenCron, runner.runMarketOpenEventJob); err != nil {
		return nil, fmt.Errorf("register market-open cron: %w", err)
	}
	entryID, err := runner.cron.AddFunc(cfg.AutoTradeCron, runner.runAutoTradeJob)
	if err != nil {
		return nil, fmt.Errorf("register auto-trade cron: %w", err)
	}
	runner.autoTradeEntry = entryID
	return runner, nil
}

func (r *CronSchedulerRunner) Start() {
	if !r.cfg.Enabled {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return
	}
	r.cron.Start()
	r.running = true
	log.Printf("[scheduler] started (%s)", r.location.String())
}

func (r *CronSchedulerRunner) Stop(ctx context.Context) error {
	if !r.cfg.Enabled {
		return nil
	}

	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}
	stopCtx := r.cron.Stop()
	r.running = false
	r.mu.Unlock()

	select {
	case <-stopCtx.Done():
		log.Printf("[scheduler] stopped")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *CronSchedulerRunner) Snapshot() CronSchedulerSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := CronSchedulerSnapshot{
		Enabled:         r.cfg.Enabled,
		Running:         r.running,
		Location:        r.location.String(),
		MarketOpenCron:  r.cfg.MarketOpenCron,
		AutoTradeCron:   r.cfg.AutoTradeCron,
		AutoTradeSymbol: r.cfg.AutoTradeSymbol,
		LastAutoTrade:   cloneAutoTradeResult(r.lastAutoTrade),
	}
	if r.cfg.Enabled && r.autoTradeEntry != 0 {
		entry := r.cron.Entry(r.autoTradeEntry)
		snapshot.AutoTradeNextAt = entry.Next
	}
	return snapshot
}

func (r *CronSchedulerRunner) runMarketOpenEventJob() {
	log.Printf("[scheduler] market-open event triggered at %s", time.Now().In(r.location).Format(time.RFC3339))
}

func (r *CronSchedulerRunner) runAutoTradeJob() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r.runAutoTradeOnce(ctx)
}

func (r *CronSchedulerRunner) runAutoTradeOnce(ctx context.Context) {
	strategyImpl, defaultLimit, err := r.buildAutoTradeStrategy()
	if err != nil {
		r.recordAutoTrade(AutoTradeJobResult{
			At:       time.Now().UTC(),
			Strategy: r.cfg.AutoTradeStrategy,
			Signal:   string(strategy.SignalHold),
			Reason:   "invalid strategy config",
			Error:    err.Error(),
		})
		log.Printf("[scheduler] auto-trade skipped: %v", err)
		return
	}

	limit := r.cfg.AutoTradeLimit
	if limit <= 0 {
		limit = defaultLimit
	}

	result, err := r.engine.Run(ctx, r.cfg.AutoTradeSymbol, r.cfg.AutoTradeInterval, limit, strategyImpl)
	if err != nil {
		r.recordAutoTrade(AutoTradeJobResult{
			At:       time.Now().UTC(),
			Strategy: strategyImpl.Name(),
			Signal:   string(strategy.SignalHold),
			Reason:   "engine run failed",
			Error:    err.Error(),
		})
		log.Printf("[scheduler] auto-trade engine error: %v", err)
		return
	}

	jobResult := AutoTradeJobResult{
		At:       time.Now().UTC(),
		Strategy: result.Strategy,
		Signal:   string(result.Signal.Action),
		Reason:   result.Signal.Reason,
	}

	switch result.Signal.Action {
	case strategy.SignalBuy:
		order, placeErr := r.trader.PlaceOrder(ctx, trading.OrderRequest{
			Symbol: r.cfg.AutoTradeSymbol,
			Side:   trading.SideBuy,
			Qty:    r.cfg.AutoTradeQty,
		})
		if placeErr != nil {
			jobResult.Error = placeErr.Error()
		} else {
			jobResult.OrderPlaced = true
			jobResult.OrderID = order.ID
		}
	case strategy.SignalSell:
		order, placeErr := r.trader.PlaceOrder(ctx, trading.OrderRequest{
			Symbol: r.cfg.AutoTradeSymbol,
			Side:   trading.SideSell,
			Qty:    r.cfg.AutoTradeQty,
		})
		if placeErr != nil {
			jobResult.Error = placeErr.Error()
		} else {
			jobResult.OrderPlaced = true
			jobResult.OrderID = order.ID
		}
	}

	r.recordAutoTrade(jobResult)
	log.Printf("[scheduler] auto-trade signal=%s order_placed=%t reason=%s err=%s", jobResult.Signal, jobResult.OrderPlaced, jobResult.Reason, jobResult.Error)
}

func (r *CronSchedulerRunner) recordAutoTrade(result AutoTradeJobResult) {
	r.mu.Lock()
	copyResult := result
	r.lastAutoTrade = &copyResult
	callback := r.onAutoTrade
	r.mu.Unlock()

	if callback != nil {
		callback(copyResult)
	}
}

func (r *CronSchedulerRunner) SetAutoTradeListener(listener func(AutoTradeJobResult)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onAutoTrade = listener
}

func (r *CronSchedulerRunner) buildAutoTradeStrategy() (strategy.Strategy, int, error) {
	name := strings.ToLower(strings.TrimSpace(r.cfg.AutoTradeStrategy))
	switch name {
	case "", "sma", "sma_crossover":
		s, err := sma.NewCrossover(r.cfg.SMAShortWindow, r.cfg.SMALongWindow)
		if err != nil {
			return nil, 0, err
		}
		return s, 60, nil
	case "infinite_buy":
		s, err := infinitebuy.New(r.cfg.InfiniteBuyCount, r.cfg.InfiniteAvgPrice, r.cfg.InfiniteAllocation)
		if err != nil {
			return nil, 0, err
		}
		return s, 2, nil
	default:
		return nil, 0, fmt.Errorf("unsupported strategy: %s", r.cfg.AutoTradeStrategy)
	}
}

func cloneAutoTradeResult(v *AutoTradeJobResult) *AutoTradeJobResult {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
