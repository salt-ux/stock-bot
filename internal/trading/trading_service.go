package trading

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/salt-ux/stock-bot/internal/market"
	"github.com/salt-ux/stock-bot/internal/risk"
)

type Config struct {
	InitialCash      float64
	DuplicateWindow  time.Duration
	MaxRecentOrders  int
	RiskMaxNotional  float64
	RiskDailyLossCap float64
}

type Service struct {
	market *market.Service
	risk   risk.Config
	cfg    Config

	mu sync.Mutex

	cash          float64
	positions     map[string]*Position
	recentOrders  []Order
	nextOrderID   int64
	realizedPnL   map[string]float64
	duplicateMemo map[string]time.Time
}

func NewService(marketSvc *market.Service, cfg Config) (*Service, error) {
	if marketSvc == nil {
		return nil, fmt.Errorf("market service is required")
	}
	if cfg.InitialCash <= 0 {
		return nil, fmt.Errorf("initial cash must be > 0")
	}
	if cfg.DuplicateWindow <= 0 {
		cfg.DuplicateWindow = 30 * time.Second
	}
	if cfg.MaxRecentOrders <= 0 {
		cfg.MaxRecentOrders = 200
	}
	if cfg.RiskMaxNotional <= 0 || cfg.RiskDailyLossCap <= 0 {
		return nil, fmt.Errorf("risk config must be > 0")
	}

	return &Service{
		market:        marketSvc,
		risk:          risk.Config{MaxPositionNotional: cfg.RiskMaxNotional, DailyLossLimit: cfg.RiskDailyLossCap},
		cfg:           cfg,
		cash:          cfg.InitialCash,
		positions:     map[string]*Position{},
		realizedPnL:   map[string]float64{},
		duplicateMemo: map[string]time.Time{},
		nextOrderID:   1,
		recentOrders:  make([]Order, 0, cfg.MaxRecentOrders),
	}, nil
}

func (s *Service) PlaceOrder(ctx context.Context, req OrderRequest) (Order, error) {
	symbol := strings.ToUpper(strings.TrimSpace(req.Symbol))
	if symbol == "" {
		return Order{}, fmt.Errorf("symbol is required")
	}
	if req.Qty <= 0 {
		return Order{}, fmt.Errorf("qty must be > 0")
	}
	if req.Side != SideBuy && req.Side != SideSell {
		return Order{}, fmt.Errorf("side must be BUY or SELL")
	}

	quote, err := s.market.Quote(ctx, symbol)
	if err != nil {
		return Order{}, fmt.Errorf("load quote: %w", err)
	}

	executionTime := quote.AsOf.UTC()
	if executionTime.IsZero() {
		executionTime = time.Now().UTC()
	}
	dayKey := executionTime.Format("2006-01-02")
	fingerprint := fmt.Sprintf("%s|%s|%d", symbol, req.Side, req.Qty)

	s.mu.Lock()
	defer s.mu.Unlock()

	if ts, ok := s.duplicateMemo[fingerprint]; ok {
		if !executionTime.After(ts) || executionTime.Sub(ts) <= s.cfg.DuplicateWindow {
			return Order{}, fmt.Errorf("duplicate order blocked")
		}
	}

	pos := s.positions[symbol]
	if pos == nil {
		pos = &Position{Symbol: symbol}
		s.positions[symbol] = pos
	}

	snap := risk.Snapshot{CurrentQty: pos.Qty, RealizedPnLToday: s.realizedPnL[dayKey]}
	intent := risk.OrderIntent{Side: string(req.Side), Quantity: req.Qty}
	if err := s.risk.Validate(intent, snap, quote.Price); err != nil {
		return Order{}, err
	}

	if req.Side == SideBuy {
		cost := quote.Price * float64(req.Qty)
		if cost > s.cash {
			return Order{}, fmt.Errorf("insufficient cash")
		}
		s.cash -= cost
		totalCost := pos.AvgPrice*float64(pos.Qty) + cost
		pos.Qty += req.Qty
		pos.AvgPrice = totalCost / float64(pos.Qty)
		pos.LastPrice = quote.Price
		pos.Unrealized = (pos.LastPrice - pos.AvgPrice) * float64(pos.Qty)
	} else {
		realized := (quote.Price - pos.AvgPrice) * float64(req.Qty)
		s.realizedPnL[dayKey] += realized
		s.cash += quote.Price * float64(req.Qty)
		pos.Qty -= req.Qty
		pos.LastPrice = quote.Price
		if pos.Qty == 0 {
			pos.AvgPrice = 0
			pos.Unrealized = 0
		} else {
			pos.Unrealized = (pos.LastPrice - pos.AvgPrice) * float64(pos.Qty)
		}
	}

	order := Order{
		ID:        s.nextOrderID,
		Symbol:    symbol,
		Side:      req.Side,
		Qty:       req.Qty,
		FillPrice: quote.Price,
		FilledAt:  executionTime,
		Status:    "FILLED",
	}
	s.nextOrderID++

	s.duplicateMemo[fingerprint] = executionTime
	s.recentOrders = append([]Order{order}, s.recentOrders...)
	if len(s.recentOrders) > s.cfg.MaxRecentOrders {
		s.recentOrders = s.recentOrders[:s.cfg.MaxRecentOrders]
	}

	return order, nil
}

func (s *Service) GetState() State {
	now := time.Now().UTC()
	dayKey := now.Format("2006-01-02")

	s.mu.Lock()
	defer s.mu.Unlock()

	positions := make([]Position, 0, len(s.positions))
	for _, p := range s.positions {
		if p.Qty == 0 {
			continue
		}
		cp := *p
		positions = append(positions, cp)
	}
	sort.Slice(positions, func(i, j int) bool { return positions[i].Symbol < positions[j].Symbol })

	orders := make([]Order, len(s.recentOrders))
	copy(orders, s.recentOrders)

	return State{
		Cash:             s.cash,
		RealizedPnLToday: s.realizedPnL[dayKey],
		Positions:        positions,
		RecentOrders:     orders,
	}
}

func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cash = s.cfg.InitialCash
	s.positions = map[string]*Position{}
	s.recentOrders = make([]Order, 0, s.cfg.MaxRecentOrders)
	s.realizedPnL = map[string]float64{}
	s.duplicateMemo = map[string]time.Time{}
	s.nextOrderID = 1
}
