package market

import "testing"

func TestDefaultSymbolLookupResolverResolveByName(t *testing.T) {
	resolver := NewDefaultSymbolLookupResolver()
	item, ok := resolver.Resolve("삼성전자")
	if !ok {
		t.Fatalf("expected symbol for 삼성전자")
	}
	if item.Symbol != "005930" {
		t.Fatalf("unexpected symbol: %s", item.Symbol)
	}
}

func TestDefaultSymbolLookupResolverResolveByAlias(t *testing.T) {
	resolver := NewDefaultSymbolLookupResolver()
	item, ok := resolver.Resolve("네이버")
	if !ok {
		t.Fatalf("expected symbol for 네이버")
	}
	if item.Symbol != "035420" {
		t.Fatalf("unexpected symbol: %s", item.Symbol)
	}
}

func TestDefaultSymbolLookupResolverSearchByPrefix(t *testing.T) {
	resolver := NewDefaultSymbolLookupResolver()
	items := resolver.Search("삼성", 10)
	if len(items) == 0 {
		t.Fatalf("expected non-empty result")
	}
	found := false
	for _, it := range items {
		if it.Symbol == "005930" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 삼성전자 in search result")
	}
}

func TestDefaultSymbolLookupResolverSearchByCodePrefix(t *testing.T) {
	resolver := NewDefaultSymbolLookupResolver()
	items := resolver.Search("0059", 10)
	if len(items) == 0 {
		t.Fatalf("expected non-empty result")
	}
	if items[0].Symbol != "005930" && items[0].Symbol != "005935" {
		t.Fatalf("unexpected first symbol: %s", items[0].Symbol)
	}
}
