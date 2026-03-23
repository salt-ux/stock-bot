package kiwoom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/salt-ux/stock-bot/internal/config"
	"github.com/salt-ux/stock-bot/internal/market"
)

var koreaLocation = loadKoreaLocation()

type Provider struct {
	baseURL   string
	tokenPath string
	appKey    string
	appSecret string

	quotePath         string
	quoteAPIID        string
	candlePath        string
	candleMinuteAPIID string
	candleDailyAPIID  string

	http *http.Client

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
	wsQuotes    *kiwoomWebSocketQuoteStream
}

func NewProvider(kiwoomCfg config.KiwoomConfig, marketCfg config.MarketConfig) *Provider {
	p := &Provider{
		baseURL:           strings.TrimRight(kiwoomCfg.BaseURL, "/"),
		tokenPath:         kiwoomCfg.TokenPath,
		appKey:            kiwoomCfg.AppKey,
		appSecret:         kiwoomCfg.AppSecret,
		quotePath:         marketCfg.QuotePath,
		quoteAPIID:        marketCfg.QuoteAPIID,
		candlePath:        marketCfg.CandlePath,
		candleMinuteAPIID: marketCfg.CandleMinuteAPIID,
		candleDailyAPIID:  marketCfg.CandleDailyAPIID,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	if marketCfg.UseWebSocket {
		wsURL := resolveKiwoomWebSocketURL(p.baseURL, marketCfg.WebSocketURL, marketCfg.WebSocketPath)
		if wsURL != "" {
			p.wsQuotes = newKiwoomWebSocketQuoteStream(wsURL, marketCfg.WebSocketQuoteTR)
		}
	}
	return p
}

func (p *Provider) GetQuote(ctx context.Context, symbol string) (market.Quote, error) {
	symbol = strings.TrimSpace(strings.ToUpper(symbol))
	if symbol == "" {
		return market.Quote{}, fmt.Errorf("symbol is required")
	}

	if p.wsQuotes != nil {
		p.wsQuotes.Start()
		p.wsQuotes.Subscribe(symbol)
		if quote, ok := p.wsQuotes.WaitLatest(ctx, symbol, 1200*time.Millisecond); ok {
			return quote, nil
		}
	}

	return p.getQuoteByREST(ctx, symbol)
}

func (p *Provider) Close() error {
	if p.wsQuotes != nil {
		p.wsQuotes.Stop()
	}
	return nil
}

func (p *Provider) getQuoteByREST(ctx context.Context, symbol string) (market.Quote, error) {
	var payload any = map[string]any{"stk_cd": symbol}
	resp, err := p.callMarketAPI(ctx, p.quotePath, p.quoteAPIID, payload)
	if err != nil {
		return market.Quote{}, err
	}

	record := findQuoteRecord(resp)
	if record == nil {
		return market.Quote{}, fmt.Errorf("quote record not found in response")
	}

	price, err := extractFloat(record,
		"cur_prc", "current_price", "stck_prpr", "close", "last", "price",
	)
	if err != nil {
		return market.Quote{}, fmt.Errorf("parse quote price: %w", err)
	}
	price = math.Abs(price)

	asOf := extractTime(record,
		[]string{"tm", "time", "stck_cntg_hour", "as_of"},
		[]string{"dt", "date", "stck_bsop_date", "as_of_date"},
	)
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}

	return market.Quote{Symbol: symbol, Price: price, AsOf: asOf}, nil
}

func (p *Provider) GetCandles(ctx context.Context, symbol string, interval string, limit int) ([]market.Candle, error) {
	symbol = strings.TrimSpace(strings.ToUpper(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if interval == "" {
		interval = "1d"
	}
	interval = strings.ToLower(strings.TrimSpace(interval))
	switch interval {
	case "1d", "1h", "1m", "5m":
	default:
		return nil, fmt.Errorf("unsupported interval: %s (use 1m, 5m, 1h, 1d)", interval)
	}
	if limit < 1 || limit > 500 {
		return nil, fmt.Errorf("limit must be between 1 and 500")
	}

	payload := map[string]any{
		"stk_cd": symbol,
	}
	sourceInterval := interval
	switch interval {
	case "1m":
		payload["tic_scope"] = "1"
		payload["timefr"] = "M"
		payload["timevl"] = "1"
		payload["upd_stkpc_tp"] = "1"
	case "5m":
		// 1일 차트는 1분 원본으로 받아 서버에서 5분 집계합니다.
		// 환경에 따라 5분 직접 조회 응답이 비거나 형태가 달라지는 경우가 있습니다.
		payload["tic_scope"] = "1"
		payload["timefr"] = "M"
		payload["timevl"] = "1"
		payload["upd_stkpc_tp"] = "1"
		sourceInterval = "1m"
	case "1h":
		payload["tic_scope"] = "60"
		payload["timefr"] = "M"
		payload["timevl"] = "60"
		payload["upd_stkpc_tp"] = "1"
	default:
		payload["base_dt"] = time.Now().Format("20060102")
		payload["timefr"] = "D"
		payload["upd_stkpc_tp"] = "1"
	}
	candleAPIID := p.candleDailyAPIID
	if interval == "1m" || interval == "5m" || interval == "1h" {
		candleAPIID = p.candleMinuteAPIID
	}
	resp, err := p.callMarketAPI(ctx, p.candlePath, candleAPIID, payload)
	if err != nil {
		return nil, err
	}

	records := findCandleRecords(resp)
	if len(records) == 0 {
		return nil, fmt.Errorf("candle records not found in response")
	}

	candles := parseCandlesFromRecords(records, symbol, sourceInterval)

	if len(candles) == 0 {
		return nil, fmt.Errorf("failed to parse candle records")
	}

	sort.Slice(candles, func(i, j int) bool {
		return candles[i].Time.Before(candles[j].Time)
	})

	if interval == "5m" {
		agg := aggregateCandlesByMinutes(candles, 5, limit, symbol)
		if len(agg) == 0 {
			return nil, fmt.Errorf("failed to aggregate 5m candles")
		}
		return agg, nil
	}

	if len(candles) > limit {
		candles = candles[len(candles)-limit:]
	}
	return candles, nil
}

func (p *Provider) callMarketAPI(ctx context.Context, path string, apiID string, payload any) (map[string]any, error) {
	token, err := p.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("authorization", "Bearer "+token)
	req.Header.Set("api-id", apiID)

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("market api call failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("market api status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode market response: %w", err)
	}
	return out, nil
}

func (p *Provider) getAccessToken(ctx context.Context) (string, error) {
	p.mu.Lock()
	if p.token != "" && time.Now().Before(p.tokenExpiry) {
		token := p.token
		p.mu.Unlock()
		return token, nil
	}
	p.mu.Unlock()

	payload, _ := json.Marshal(map[string]string{
		"grant_type": "client_credentials",
		"appkey":     p.appKey,
		"secretkey":  p.appSecret,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+p.tokenPath, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("token status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	token := firstString(out, "token", "access_token")
	if token == "" {
		return "", fmt.Errorf("token field not found")
	}

	expiresAt := time.Now().Add(50 * time.Minute)
	if sec, ok := tryInt(out["expires_in"]); ok && sec > 30 {
		expiresAt = time.Now().Add(time.Duration(sec-30) * time.Second)
	}

	p.mu.Lock()
	p.token = token
	p.tokenExpiry = expiresAt
	p.mu.Unlock()
	return token, nil
}

func resolveKiwoomWebSocketURL(baseHTTPURL, overrideWSURL, wsPath string) string {
	if strings.TrimSpace(overrideWSURL) != "" {
		return strings.TrimRight(strings.TrimSpace(overrideWSURL), "/")
	}
	base := strings.TrimSpace(baseHTTPURL)
	if base == "" {
		return ""
	}
	base = strings.TrimRight(base, "/")
	switch {
	case strings.HasPrefix(base, "https://"):
		base = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		base = "ws://" + strings.TrimPrefix(base, "http://")
	default:
		return ""
	}
	if withPort, ok := ensureWebSocketPort(base); ok {
		base = withPort
	}
	path := strings.TrimSpace(wsPath)
	if path == "" {
		return base
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func findQuoteRecord(v any) map[string]any {
	for _, item := range findMapNodes(v) {
		if _, ok := tryFloat(item["cur_prc"]); ok {
			return item
		}
		if _, ok := tryFloat(item["stck_prpr"]); ok {
			return item
		}
	}
	maps := findMapNodes(v)
	if len(maps) > 0 {
		return maps[0]
	}
	return nil
}

func findCandleRecords(v any) []map[string]any {
	arrays := findArrayNodes(v)
	for _, arr := range arrays {
		items := make([]map[string]any, 0, len(arr))
		for _, it := range arr {
			m, ok := it.(map[string]any)
			if !ok {
				items = nil
				break
			}
			dateOrTime := firstString(m,
				"dt", "date", "stck_bsop_date", "bas_dt",
				"tm", "time", "stck_cntg_hour", "cntg_tm", "cntr_tm", "bsop_hour",
			)
			if dateOrTime == "" {
				continue
			}
			if _, ok := tryFloat(m["cur_prc"]); !ok {
				if _, ok := tryFloat(m["close"]); !ok {
					if _, ok := tryFloat(m["stck_clpr"]); !ok {
						continue
					}
				}
			}
			if firstString(m, "dt", "date", "stck_bsop_date", "bas_dt") == "" &&
				firstString(m, "tm", "time", "stck_cntg_hour", "cntg_tm", "cntr_tm", "bsop_hour") == "" {
				continue
			}
			items = append(items, m)
		}
		if len(items) > 0 {
			return items
		}
	}
	return nil
}

func findMapNodes(v any) []map[string]any {
	res := []map[string]any{}
	var walk func(any)
	walk = func(cur any) {
		switch t := cur.(type) {
		case map[string]any:
			res = append(res, t)
			for _, val := range t {
				walk(val)
			}
		case []any:
			for _, val := range t {
				walk(val)
			}
		}
	}
	walk(v)
	return res
}

func findArrayNodes(v any) [][]any {
	res := [][]any{}
	var walk func(any)
	walk = func(cur any) {
		switch t := cur.(type) {
		case []any:
			res = append(res, t)
			for _, val := range t {
				walk(val)
			}
		case map[string]any:
			for _, val := range t {
				walk(val)
			}
		}
	}
	walk(v)
	return res
}

func parseCandlesFromRecords(records []map[string]any, symbol, interval string) []market.Candle {
	candles := make([]market.Candle, 0, len(records))
	for _, rec := range records {
		closePrice, err := extractFloat(rec, "cur_prc", "close", "stck_clpr")
		if err != nil {
			continue
		}
		openPrice, _ := extractFloat(rec, "open_pric", "open", "stck_oprc")
		highPrice, _ := extractFloat(rec, "high_pric", "high", "stck_hgpr")
		lowPrice, _ := extractFloat(rec, "low_pric", "low", "stck_lwpr")
		volume, _ := extractInt(rec, "trde_qty", "volume", "acml_vol")
		closePrice = math.Abs(closePrice)
		openPrice = math.Abs(openPrice)
		highPrice = math.Abs(highPrice)
		lowPrice = math.Abs(lowPrice)

		if openPrice <= 0 {
			openPrice = closePrice
		}
		if highPrice <= 0 {
			highPrice = math.Max(openPrice, closePrice)
		}
		if lowPrice <= 0 {
			lowPrice = math.Min(openPrice, closePrice)
		}

		tm := extractTime(rec,
			[]string{"tm", "time", "stck_cntg_hour", "cntr_tm"},
			[]string{"dt", "date", "stck_bsop_date", "bas_dt"},
		)
		if tm.IsZero() {
			tm = time.Now().UTC()
		}

		candles = append(candles, market.Candle{
			Symbol:   symbol,
			Interval: interval,
			Time:     tm,
			Open:     openPrice,
			High:     highPrice,
			Low:      lowPrice,
			Close:    closePrice,
			Volume:   volume,
		})
	}
	return candles
}

func aggregateCandlesByMinutes(src []market.Candle, minutes int, limit int, symbol string) []market.Candle {
	if len(src) == 0 || minutes <= 0 {
		return nil
	}
	sort.Slice(src, func(i, j int) bool {
		return src[i].Time.Before(src[j].Time)
	})

	out := make([]market.Candle, 0, len(src))
	var cur market.Candle
	var curKey time.Time
	for i, c := range src {
		t := c.Time.In(koreaLocation)
		bucket := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), (t.Minute()/minutes)*minutes, 0, 0, koreaLocation)
		if i == 0 || !bucket.Equal(curKey) {
			if i > 0 {
				out = append(out, cur)
			}
			curKey = bucket
			cur = market.Candle{
				Symbol:   symbol,
				Interval: fmt.Sprintf("%dm", minutes),
				Time:     bucket.UTC(),
				Open:     c.Open,
				High:     c.High,
				Low:      c.Low,
				Close:    c.Close,
				Volume:   c.Volume,
			}
			continue
		}
		if c.High > cur.High {
			cur.High = c.High
		}
		if c.Low < cur.Low {
			cur.Low = c.Low
		}
		cur.Close = c.Close
		cur.Volume += c.Volume
	}
	out = append(out, cur)
	if len(out) > limit && limit > 0 {
		out = out[len(out)-limit:]
	}
	return out
}

func extractFloat(m map[string]any, keys ...string) (float64, error) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if f, ok := tryFloat(v); ok {
				return f, nil
			}
		}
	}
	return 0, fmt.Errorf("keys not found: %v", keys)
}

func extractInt(m map[string]any, keys ...string) (int64, error) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if n, ok := tryInt(v); ok {
				return n, nil
			}
		}
	}
	return 0, fmt.Errorf("keys not found: %v", keys)
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func extractTime(m map[string]any, timeKeys, dateKeys []string) time.Time {
	dateRaw := strings.ReplaceAll(firstString(m, dateKeys...), "-", "")
	timeRaw := strings.ReplaceAll(firstString(m, timeKeys...), ":", "")
	dateRaw = padRightDigits(dateRaw, len(dateRaw))
	timeRaw = padRightDigits(timeRaw, len(timeRaw))

	if len(dateRaw) >= 14 {
		if ts, err := time.ParseInLocation("20060102150405", dateRaw[:14], koreaLocation); err == nil {
			return ts
		}
	}
	if len(dateRaw) >= 12 {
		if ts, err := time.ParseInLocation("200601021504", dateRaw[:12], koreaLocation); err == nil {
			return ts
		}
	}

	if dateRaw != "" && len(dateRaw) == 8 && timeRaw != "" {
		timeRaw = padRightDigits(timeRaw, 6)
		if ts, err := time.ParseInLocation("20060102150405", dateRaw+timeRaw, koreaLocation); err == nil {
			return ts
		}
	}
	if dateRaw == "" && timeRaw != "" {
		only := padRightDigits(timeRaw, 14)
		if len(only) >= 14 {
			if ts, err := time.ParseInLocation("20060102150405", only[:14], koreaLocation); err == nil {
				return ts
			}
		}
	}
	if dateRaw != "" {
		if ts, err := time.ParseInLocation("20060102", dateRaw, koreaLocation); err == nil {
			return ts
		}
	}
	if asof := firstString(m, "as_of"); asof != "" {
		if ts, err := time.Parse(time.RFC3339, asof); err == nil {
			return ts
		}
	}
	return time.Time{}
}

func loadKoreaLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return time.FixedZone("KST", 9*60*60)
	}
	return loc
}

func padRightDigits(s string, n int) string {
	only := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			only = append(only, r)
		}
	}
	for len(only) < n {
		only = append(only, '0')
	}
	if len(only) > n {
		only = only[:n]
	}
	return string(only)
}

func tryFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		s := strings.TrimSpace(t)
		s = strings.ReplaceAll(s, ",", "")
		s = strings.TrimPrefix(s, "+")
		if s == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func tryInt(v any) (int64, bool) {
	switch t := v.(type) {
	case int:
		return int64(t), true
	case int64:
		return t, true
	case float64:
		return int64(t), true
	case json.Number:
		i, err := t.Int64()
		if err == nil {
			return i, true
		}
		f, err := t.Float64()
		if err != nil {
			return 0, false
		}
		return int64(f), true
	case string:
		s := strings.TrimSpace(t)
		s = strings.ReplaceAll(s, ",", "")
		s = strings.TrimPrefix(s, "+")
		if s == "" {
			return 0, false
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			return i, true
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false
		}
		return int64(f), true
	default:
		return 0, false
	}
}

func ensureWebSocketPort(base string) (string, bool) {
	u, err := url.Parse(base)
	if err != nil {
		return "", false
	}
	host := u.Host
	if host == "" {
		return "", false
	}
	if _, _, err := net.SplitHostPort(host); err == nil {
		return base, true
	}
	u.Host = net.JoinHostPort(host, "10000")
	return strings.TrimRight(u.String(), "/"), true
}
