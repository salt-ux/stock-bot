package kiwoom

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/salt-ux/stock-bot/internal/config"
	"github.com/salt-ux/stock-bot/internal/market"
)

type SymbolLookupSearcher struct {
	provider *Provider

	path            string
	listAPIID       string
	searchAPIID     string
	cacheTTL        time.Duration
	defaultLimit    int
	mu              sync.RWMutex
	cachedMaster    []market.SymbolLookupResult
	cachedExpiresAt time.Time
}

func NewSymbolLookupSearcher(kiwoomCfg config.KiwoomConfig, marketCfg config.MarketConfig) *SymbolLookupSearcher {
	cfgCopy := marketCfg
	cfgCopy.UseWebSocket = false

	ttl := time.Duration(marketCfg.SymbolCacheTTL) * time.Second
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}

	return &SymbolLookupSearcher{
		provider:     NewProvider(kiwoomCfg, cfgCopy),
		path:         strings.TrimSpace(marketCfg.SymbolPath),
		listAPIID:    strings.TrimSpace(marketCfg.SymbolListAPIID),
		searchAPIID:  strings.TrimSpace(marketCfg.SymbolSearchAPIID),
		cacheTTL:     ttl,
		defaultLimit: 50,
		cachedMaster: nil,
	}
}

func (s *SymbolLookupSearcher) SearchSymbols(ctx context.Context, query string, limit int) ([]market.SymbolLookupResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = s.defaultLimit
	}
	if limit > 50 {
		limit = 50
	}
	if s.provider == nil {
		return nil, fmt.Errorf("kiwoom symbol search provider is not initialized")
	}
	if s.path == "" {
		return nil, fmt.Errorf("kiwoom symbol path is not configured")
	}

	if items, err := s.searchByQueryAPI(ctx, query, limit); err == nil && len(items) > 0 {
		return items, nil
	}

	master, err := s.loadMasterSymbols(ctx)
	if err != nil {
		return nil, err
	}
	filtered := filterAndRankSymbolResults(master, query, limit)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no symbols matched query")
	}
	return filtered, nil
}

func (s *SymbolLookupSearcher) searchByQueryAPI(ctx context.Context, query string, limit int) ([]market.SymbolLookupResult, error) {
	if s.searchAPIID == "" {
		return nil, fmt.Errorf("kiwoom symbol search api id is not configured")
	}

	var lastErr error
	for _, payload := range searchPayloadCandidates(query) {
		resp, err := s.provider.callMarketAPI(ctx, s.path, s.searchAPIID, payload)
		if err != nil {
			lastErr = err
			continue
		}
		results := extractSymbolLookupResults(resp, limit)
		if len(results) > 0 {
			return filterAndRankSymbolResults(results, query, limit), nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("query api returned no symbol records")
}

func (s *SymbolLookupSearcher) loadMasterSymbols(ctx context.Context) ([]market.SymbolLookupResult, error) {
	now := time.Now()

	s.mu.RLock()
	if len(s.cachedMaster) > 0 && now.Before(s.cachedExpiresAt) {
		out := make([]market.SymbolLookupResult, len(s.cachedMaster))
		copy(out, s.cachedMaster)
		s.mu.RUnlock()
		return out, nil
	}
	s.mu.RUnlock()

	if s.listAPIID == "" {
		return nil, fmt.Errorf("kiwoom symbol list api id is not configured")
	}

	var lastErr error
	for _, payload := range masterListPayloadCandidates() {
		resp, err := s.provider.callMarketAPI(ctx, s.path, s.listAPIID, payload)
		if err != nil {
			lastErr = err
			continue
		}
		results := extractSymbolLookupResults(resp, 5000)
		if len(results) == 0 {
			continue
		}

		s.mu.Lock()
		s.cachedMaster = results
		s.cachedExpiresAt = now.Add(s.cacheTTL)
		s.mu.Unlock()

		out := make([]market.SymbolLookupResult, len(results))
		copy(out, results)
		return out, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("master list api returned no symbol records")
}

func searchPayloadCandidates(query string) []map[string]any {
	normalized := strings.TrimSpace(query)
	if normalized == "" {
		return []map[string]any{{}}
	}
	if isNumericSymbolCode(normalized) {
		return []map[string]any{
			{"stk_cd": normalized},
			{"symbol": normalized},
			{"code": normalized},
		}
	}
	return []map[string]any{
		{"stk_nm": normalized},
		{"name": normalized},
		{"query": normalized},
	}
}

func masterListPayloadCandidates() []map[string]any {
	return []map[string]any{
		{},
		{"mrkt_tp": "0"},
		{"mrkt_tp": "ALL"},
	}
}

func extractSymbolLookupResults(resp map[string]any, limit int) []market.SymbolLookupResult {
	if resp == nil {
		return nil
	}

	records := findMapNodes(resp)
	seen := make(map[string]struct{}, len(records))
	out := make([]market.SymbolLookupResult, 0, minInt(limit, len(records)))
	for _, rec := range records {
		symbol := strings.ToUpper(strings.TrimSpace(firstString(
			rec,
			"symbol", "stk_cd", "code", "isu_cd", "item", "short_code",
		)))
		if symbol == "" {
			continue
		}

		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}

		name := strings.TrimSpace(firstString(
			rec,
			"name", "stk_nm", "stock_name", "item_name", "isu_nm", "hts_kor_isnm",
		))
		if name == "" {
			name = symbol
		}
		marketName := strings.TrimSpace(firstString(
			rec,
			"market", "mkt_nm", "market_name", "mrkt", "stex_tp",
		))

		out = append(out, market.SymbolLookupResult{
			Symbol: symbol,
			Name:   name,
			Market: marketName,
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func filterAndRankSymbolResults(items []market.SymbolLookupResult, query string, limit int) []market.SymbolLookupResult {
	if len(items) == 0 {
		return nil
	}
	queryKey := normalizeSymbolLookupKey(query)
	if queryKey == "" {
		return nil
	}

	type candidate struct {
		item  market.SymbolLookupResult
		score int
	}
	candidates := make([]candidate, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		symbol := strings.ToUpper(strings.TrimSpace(item.Symbol))
		if symbol == "" {
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}

		score, ok := symbolMatchScore(item, queryKey)
		if !ok {
			continue
		}
		item.Symbol = symbol
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			item.Name = symbol
		}
		candidates = append(candidates, candidate{item: item, score: score})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score < candidates[j].score
		}
		if candidates[i].item.Name != candidates[j].item.Name {
			return candidates[i].item.Name < candidates[j].item.Name
		}
		return candidates[i].item.Symbol < candidates[j].item.Symbol
	})

	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	out := make([]market.SymbolLookupResult, 0, minInt(limit, len(candidates)))
	for _, c := range candidates {
		out = append(out, c.item)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func symbolMatchScore(item market.SymbolLookupResult, queryKey string) (int, bool) {
	keys := []string{
		normalizeSymbolLookupKey(item.Symbol),
		normalizeSymbolLookupKey(item.Name),
	}

	best := 99
	for _, key := range keys {
		if key == "" {
			continue
		}
		switch {
		case key == queryKey:
			if best > 0 {
				best = 0
			}
		case strings.HasPrefix(key, queryKey):
			if best > 1 {
				best = 1
			}
		case strings.Contains(key, queryKey):
			if best > 2 {
				best = 2
			}
		}
	}
	if best == 99 {
		return 0, false
	}
	return best, true
}

func normalizeSymbolLookupKey(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case unicode.IsSpace(r):
			continue
		case r == '-', r == '_', r == '/', r == '.':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isNumericSymbolCode(value string) bool {
	if len(value) != 6 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
