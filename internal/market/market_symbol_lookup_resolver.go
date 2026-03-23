package market

import (
	"sort"
	"strings"
	"unicode"
)

type SymbolLookupResult struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Market string `json:"market,omitempty"`
}

type SymbolLookupResolver interface {
	Resolve(query string) (SymbolLookupResult, bool)
	Search(query string, limit int) []SymbolLookupResult
}

type staticSymbolLookupResolver struct {
	entries []symbolLookupEntry
}

type symbolLookupEntry struct {
	result SymbolLookupResult
	keys   []string
}

type symbolCatalogSeed struct {
	Symbol  string
	Name    string
	Market  string
	Aliases []string
}

func NewDefaultSymbolLookupResolver() SymbolLookupResolver {
	seeds := []symbolCatalogSeed{
		{Symbol: "005930", Name: "삼성전자", Market: "KOSPI", Aliases: []string{"samsungelectronics", "samsung", "삼전"}},
		{Symbol: "005935", Name: "삼성전자우", Market: "KOSPI", Aliases: []string{"삼성전자우선주"}},
		{Symbol: "000660", Name: "SK하이닉스", Market: "KOSPI", Aliases: []string{"하이닉스", "skhynix"}},
		{Symbol: "035420", Name: "NAVER", Market: "KOSPI", Aliases: []string{"네이버"}},
		{Symbol: "035720", Name: "카카오", Market: "KOSPI", Aliases: []string{"kakao"}},
		{Symbol: "105560", Name: "KB금융", Market: "KOSPI"},
		{Symbol: "055550", Name: "신한지주", Market: "KOSPI"},
		{Symbol: "086790", Name: "하나금융지주", Market: "KOSPI"},
		{Symbol: "323410", Name: "카카오뱅크", Market: "KOSPI"},
		{Symbol: "207940", Name: "삼성바이오로직스", Market: "KOSPI", Aliases: []string{"삼바"}},
		{Symbol: "068270", Name: "셀트리온", Market: "KOSPI"},
		{Symbol: "006400", Name: "삼성SDI", Market: "KOSPI"},
		{Symbol: "066570", Name: "LG전자", Market: "KOSPI"},
		{Symbol: "051910", Name: "LG화학", Market: "KOSPI"},
		{Symbol: "003550", Name: "LG", Market: "KOSPI"},
		{Symbol: "005380", Name: "현대차", Market: "KOSPI", Aliases: []string{"현대자동차"}},
		{Symbol: "000270", Name: "기아", Market: "KOSPI"},
		{Symbol: "012330", Name: "현대모비스", Market: "KOSPI"},
		{Symbol: "017670", Name: "SK텔레콤", Market: "KOSPI"},
		{Symbol: "096770", Name: "SK이노베이션", Market: "KOSPI", Aliases: []string{"skinnovation"}},
		{Symbol: "028260", Name: "삼성물산", Market: "KOSPI"},
		{Symbol: "015760", Name: "한국전력", Market: "KOSPI", Aliases: []string{"한전"}},
		{Symbol: "034020", Name: "두산에너빌리티", Market: "KOSPI", Aliases: []string{"두산중공업"}},
		{Symbol: "005490", Name: "POSCO홀딩스", Market: "KOSPI", Aliases: []string{"포스코홀딩스", "posco"}},
		{Symbol: "373220", Name: "LG에너지솔루션", Market: "KOSPI", Aliases: []string{"lgenergysolution"}},
		{Symbol: "009830", Name: "한화솔루션", Market: "KOSPI", Aliases: []string{"한화솔루션우선주", "hanwhasolutions"}},
		{Symbol: "009835", Name: "한화솔루션우", Market: "KOSPI"},
		{Symbol: "042660", Name: "한화오션", Market: "KOSPI"},
	}

	entries := make([]symbolLookupEntry, 0, len(seeds))
	for _, s := range seeds {
		keys := []string{
			normalizeLookupKey(s.Symbol),
			normalizeLookupKey(s.Name),
		}
		for _, alias := range s.Aliases {
			keys = append(keys, normalizeLookupKey(alias))
		}
		entries = append(entries, symbolLookupEntry{
			result: SymbolLookupResult{
				Symbol: s.Symbol,
				Name:   s.Name,
				Market: s.Market,
			},
			keys: dedupNonEmpty(keys),
		})
	}
	return &staticSymbolLookupResolver{entries: entries}
}

func (r *staticSymbolLookupResolver) Resolve(query string) (SymbolLookupResult, bool) {
	items := r.Search(query, 1)
	if len(items) == 0 {
		return SymbolLookupResult{}, false
	}
	return items[0], true
}

func (r *staticSymbolLookupResolver) Search(query string, limit int) []SymbolLookupResult {
	q := normalizeLookupKey(query)
	if q == "" {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	type candidate struct {
		result SymbolLookupResult
		score  int
	}
	candidates := make(map[string]candidate)
	for _, entry := range r.entries {
		score, ok := matchLookupScore(entry.keys, q)
		if !ok {
			continue
		}
		prev, exists := candidates[entry.result.Symbol]
		if !exists || score < prev.score {
			candidates[entry.result.Symbol] = candidate{result: entry.result, score: score}
		}
	}

	list := make([]candidate, 0, len(candidates))
	for _, c := range candidates {
		list = append(list, c)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].score != list[j].score {
			return list[i].score < list[j].score
		}
		if list[i].result.Name != list[j].result.Name {
			return list[i].result.Name < list[j].result.Name
		}
		return list[i].result.Symbol < list[j].result.Symbol
	})

	out := make([]SymbolLookupResult, 0, minInt(limit, len(list)))
	for _, c := range list {
		out = append(out, c.result)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func matchLookupScore(keys []string, query string) (int, bool) {
	best := 99
	for _, key := range keys {
		switch {
		case key == query:
			if best > 0 {
				best = 0
			}
		case strings.HasPrefix(key, query):
			if best > 1 {
				best = 1
			}
		case strings.Contains(key, query):
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

func normalizeLookupKey(s string) string {
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
		case r == '-' || r == '_' || r == '/' || r == '.':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func dedupNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
