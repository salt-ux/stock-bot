package market

import (
	"context"
	"strings"
	"time"
)

type RemoteSymbolSearcher interface {
	SearchSymbols(ctx context.Context, query string, limit int) ([]SymbolLookupResult, error)
}

type remoteSymbolLookupResolver struct {
	remote   RemoteSymbolSearcher
	fallback SymbolLookupResolver
	timeout  time.Duration
}

func NewRemoteFallbackSymbolLookupResolver(
	remote RemoteSymbolSearcher,
	fallback SymbolLookupResolver,
	timeout time.Duration,
) SymbolLookupResolver {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &remoteSymbolLookupResolver{
		remote:   remote,
		fallback: fallback,
		timeout:  timeout,
	}
}

func (r *remoteSymbolLookupResolver) Resolve(query string) (SymbolLookupResult, bool) {
	items := r.Search(query, 1)
	if len(items) == 0 {
		return SymbolLookupResult{}, false
	}
	return items[0], true
}

func (r *remoteSymbolLookupResolver) Search(query string, limit int) []SymbolLookupResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	if r.remote != nil {
		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()

		items, err := r.remote.SearchSymbols(ctx, query, limit)
		if err == nil && len(items) > 0 {
			return dedupAndTrimLookup(items, limit)
		}
	}

	if r.fallback == nil {
		return nil
	}
	return dedupAndTrimLookup(r.fallback.Search(query, limit), limit)
}

func dedupAndTrimLookup(items []SymbolLookupResult, limit int) []SymbolLookupResult {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]SymbolLookupResult, 0, minInt(limit, len(items)))
	for _, item := range items {
		symbol := strings.TrimSpace(strings.ToUpper(item.Symbol))
		if symbol == "" {
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		item.Symbol = symbol
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			item.Name = symbol
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out
}
