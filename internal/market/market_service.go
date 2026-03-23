package market

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Service struct {
	provider Provider
	ttl      time.Duration

	mu          sync.RWMutex
	quoteCache  map[string]quoteEntry
	candleCache map[string]candleEntry
	quoteManual map[string]Quote
}

type closerProvider interface {
	Close() error
}

type quoteEntry struct {
	data      Quote
	expiresAt time.Time
}

type candleEntry struct {
	data      []Candle
	expiresAt time.Time
}

func NewService(provider Provider, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	return &Service{
		provider:    provider,
		ttl:         ttl,
		quoteCache:  make(map[string]quoteEntry),
		candleCache: make(map[string]candleEntry),
		quoteManual: make(map[string]Quote),
	}
}

func (s *Service) Quote(ctx context.Context, symbol string) (Quote, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return Quote{}, fmt.Errorf("symbol is required")
	}
	now := time.Now()

	s.mu.RLock()
	manual, hasManual := s.quoteManual[symbol]
	if hasManual {
		s.mu.RUnlock()
		return manual, nil
	}
	item, ok := s.quoteCache[symbol]
	s.mu.RUnlock()
	if ok && now.Before(item.expiresAt) {
		return item.data, nil
	}

	quote, err := s.provider.GetQuote(ctx, symbol)
	if err != nil {
		return Quote{}, err
	}

	s.mu.Lock()
	s.quoteCache[symbol] = quoteEntry{data: quote, expiresAt: now.Add(s.ttl)}
	s.mu.Unlock()
	return quote, nil
}

func (s *Service) Candles(ctx context.Context, symbol, interval string, limit int) ([]Candle, error) {
	if limit < 1 || limit > 500 {
		return nil, fmt.Errorf("limit must be between 1 and 500")
	}

	now := time.Now()
	key := fmt.Sprintf("%s|%s|%d", symbol, interval, limit)

	s.mu.RLock()
	item, ok := s.candleCache[key]
	s.mu.RUnlock()
	if ok && now.Before(item.expiresAt) {
		return item.data, nil
	}

	candles, err := s.provider.GetCandles(ctx, symbol, interval, limit)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.candleCache[key] = candleEntry{data: candles, expiresAt: now.Add(s.ttl)}
	s.mu.Unlock()
	return candles, nil
}

func (s *Service) Close() error {
	cp, ok := s.provider.(closerProvider)
	if !ok {
		return nil
	}
	return cp.Close()
}

func (s *Service) SetQuoteOverride(symbol string, price float64, asOf time.Time) error {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if price <= 0 {
		return fmt.Errorf("price must be > 0")
	}
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	quote := Quote{
		Symbol: symbol,
		Price:  price,
		AsOf:   asOf.UTC(),
	}

	s.mu.Lock()
	s.quoteManual[symbol] = quote
	s.quoteCache[symbol] = quoteEntry{
		data:      quote,
		expiresAt: time.Now().Add(s.ttl),
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) ClearQuoteOverride(symbol string) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return
	}
	s.mu.Lock()
	delete(s.quoteManual, symbol)
	delete(s.quoteCache, symbol)
	s.mu.Unlock()
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}
