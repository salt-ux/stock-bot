package market

import (
	"context"
	"errors"
	"testing"
)

type stubRemoteSymbolSearcher struct {
	items []SymbolLookupResult
	err   error
}

func (s stubRemoteSymbolSearcher) SearchSymbols(_ context.Context, _ string, _ int) ([]SymbolLookupResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make([]SymbolLookupResult, len(s.items))
	copy(out, s.items)
	return out, nil
}

type stubFallbackResolver struct {
	items []SymbolLookupResult
}

func (s stubFallbackResolver) Resolve(query string) (SymbolLookupResult, bool) {
	if len(s.items) == 0 {
		return SymbolLookupResult{}, false
	}
	return s.items[0], true
}

func (s stubFallbackResolver) Search(query string, limit int) []SymbolLookupResult {
	out := make([]SymbolLookupResult, len(s.items))
	copy(out, s.items)
	return out
}

func TestRemoteFallbackSymbolLookupResolverUsesRemote(t *testing.T) {
	r := NewRemoteFallbackSymbolLookupResolver(
		stubRemoteSymbolSearcher{
			items: []SymbolLookupResult{
				{Symbol: "009830", Name: "한화솔루션", Market: "KOSPI"},
			},
		},
		stubFallbackResolver{
			items: []SymbolLookupResult{{Symbol: "005930", Name: "삼성전자"}},
		},
		0,
	)

	items := r.Search("한화솔루션", 5)
	if len(items) != 1 {
		t.Fatalf("unexpected remote items length: %d", len(items))
	}
	if items[0].Symbol != "009830" {
		t.Fatalf("unexpected symbol: %s", items[0].Symbol)
	}
}

func TestRemoteFallbackSymbolLookupResolverFallsBackOnError(t *testing.T) {
	r := NewRemoteFallbackSymbolLookupResolver(
		stubRemoteSymbolSearcher{err: errors.New("remote failed")},
		stubFallbackResolver{
			items: []SymbolLookupResult{{Symbol: "005930", Name: "삼성전자"}},
		},
		0,
	)

	items := r.Search("삼성", 5)
	if len(items) != 1 {
		t.Fatalf("unexpected fallback items length: %d", len(items))
	}
	if items[0].Symbol != "005930" {
		t.Fatalf("unexpected symbol: %s", items[0].Symbol)
	}
}
