package kiwoom

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/salt-ux/stock-bot/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestGetQuote(t *testing.T) {
	p := NewProvider(
		config.KiwoomConfig{BaseURL: "https://api.test", TokenPath: "/oauth2/token", AppKey: "app-key-1234", AppSecret: "app-secret-1234"},
		config.MarketConfig{QuotePath: "/api/dostk/stkinfo", QuoteAPIID: "ka10001", CandlePath: "/api/dostk/chart", CandleMinuteAPIID: "ka10080", CandleDailyAPIID: "ka10081"},
	)

	p.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/oauth2/token":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"token":"tkn"}`)), Header: make(http.Header)}, nil
		case "/api/dostk/stkinfo":
			if got := r.Header.Get("authorization"); got != "Bearer tkn" {
				t.Fatalf("missing bearer token header: %s", got)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"output":{"cur_prc":"70000","dt":"20260218","tm":"153000"}}`)), Header: make(http.Header)}, nil
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
			return nil, nil
		}
	})}

	q, err := p.GetQuote(context.Background(), "005930")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Symbol != "005930" {
		t.Fatalf("unexpected symbol: %s", q.Symbol)
	}
	if q.Price != 70000 {
		t.Fatalf("unexpected price: %f", q.Price)
	}
}

func TestGetCandles(t *testing.T) {
	p := NewProvider(
		config.KiwoomConfig{BaseURL: "https://api.test", TokenPath: "/oauth2/token", AppKey: "app-key-1234", AppSecret: "app-secret-1234"},
		config.MarketConfig{QuotePath: "/api/dostk/stkinfo", QuoteAPIID: "ka10001", CandlePath: "/api/dostk/chart", CandleMinuteAPIID: "ka10080", CandleDailyAPIID: "ka10081"},
	)

	p.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/oauth2/token":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"token":"tkn"}`)), Header: make(http.Header)}, nil
		case "/api/dostk/chart":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"output":[{"dt":"20260218","open_pric":"100","high_pric":"120","low_pric":"90","cur_prc":"110","trde_qty":"1000"},{"dt":"20260217","open_pric":"99","high_pric":"121","low_pric":"95","cur_prc":"108","trde_qty":"950"}]}`)), Header: make(http.Header)}, nil
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
			return nil, nil
		}
	})}

	candles, err := p.GetCandles(context.Background(), "005930", "1d", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) != 2 {
		t.Fatalf("unexpected candle len: %d", len(candles))
	}
	if candles[0].Close != 108 {
		t.Fatalf("unexpected close: %f", candles[0].Close)
	}
	if candles[1].Close != 110 {
		t.Fatalf("unexpected latest close: %f", candles[1].Close)
	}
}

func TestGetCandles5mAggregatesFrom1m(t *testing.T) {
	p := NewProvider(
		config.KiwoomConfig{BaseURL: "https://api.test", TokenPath: "/oauth2/token", AppKey: "app-key-1234", AppSecret: "app-secret-1234"},
		config.MarketConfig{QuotePath: "/api/dostk/stkinfo", QuoteAPIID: "ka10001", CandlePath: "/api/dostk/chart", CandleMinuteAPIID: "ka10080", CandleDailyAPIID: "ka10081"},
	)

	p.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/oauth2/token":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"token":"tkn"}`)), Header: make(http.Header)}, nil
		case "/api/dostk/chart":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"output":[
				{"dt":"20260218","tm":"100000","open_pric":"100","high_pric":"101","low_pric":"99","cur_prc":"100","trde_qty":"10"},
				{"dt":"20260218","tm":"100100","open_pric":"100","high_pric":"102","low_pric":"100","cur_prc":"101","trde_qty":"11"},
				{"dt":"20260218","tm":"100200","open_pric":"101","high_pric":"103","low_pric":"101","cur_prc":"102","trde_qty":"12"},
				{"dt":"20260218","tm":"100300","open_pric":"102","high_pric":"104","low_pric":"102","cur_prc":"103","trde_qty":"13"},
				{"dt":"20260218","tm":"100400","open_pric":"103","high_pric":"105","low_pric":"103","cur_prc":"104","trde_qty":"14"},
				{"dt":"20260218","tm":"100500","open_pric":"104","high_pric":"106","low_pric":"104","cur_prc":"105","trde_qty":"15"}
			]}`)), Header: make(http.Header)}, nil
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
			return nil, nil
		}
	})}

	candles, err := p.GetCandles(context.Background(), "005930", "5m", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) != 2 {
		t.Fatalf("unexpected aggregated candle len: %d", len(candles))
	}
	if candles[0].Open != 100 || candles[0].Close != 104 {
		t.Fatalf("unexpected first aggregate candle: open=%f close=%f", candles[0].Open, candles[0].Close)
	}
	if candles[1].Open != 104 || candles[1].Close != 105 {
		t.Fatalf("unexpected second aggregate candle: open=%f close=%f", candles[1].Open, candles[1].Close)
	}
}

func TestGetQuoteSignedPriceNormalized(t *testing.T) {
	p := NewProvider(
		config.KiwoomConfig{BaseURL: "https://api.test", TokenPath: "/oauth2/token", AppKey: "app-key-1234", AppSecret: "app-secret-1234"},
		config.MarketConfig{QuotePath: "/api/dostk/stkinfo", QuoteAPIID: "ka10001", CandlePath: "/api/dostk/chart", CandleMinuteAPIID: "ka10080", CandleDailyAPIID: "ka10081"},
	)

	p.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/oauth2/token":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"token":"tkn"}`)), Header: make(http.Header)}, nil
		case "/api/dostk/stkinfo":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"output":{"cur_prc":"-70000","dt":"20260218","tm":"153000"}}`)), Header: make(http.Header)}, nil
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
			return nil, nil
		}
	})}

	q, err := p.GetQuote(context.Background(), "005930")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Price != 70000 {
		t.Fatalf("unexpected normalized price: %f", q.Price)
	}
}

func TestGetCandlesSignedPricesNormalized(t *testing.T) {
	p := NewProvider(
		config.KiwoomConfig{BaseURL: "https://api.test", TokenPath: "/oauth2/token", AppKey: "app-key-1234", AppSecret: "app-secret-1234"},
		config.MarketConfig{QuotePath: "/api/dostk/stkinfo", QuoteAPIID: "ka10001", CandlePath: "/api/dostk/chart", CandleMinuteAPIID: "ka10080", CandleDailyAPIID: "ka10081"},
	)

	p.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/oauth2/token":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"token":"tkn"}`)), Header: make(http.Header)}, nil
		case "/api/dostk/chart":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"output":[{"dt":"20260218","tm":"090000","open_pric":"-100","high_pric":"-120","low_pric":"-90","cur_prc":"-110","trde_qty":"1000"},{"dt":"20260218","tm":"090100","open_pric":"+111","high_pric":"+121","low_pric":"+101","cur_prc":"+115","trde_qty":"900"}]}`)), Header: make(http.Header)}, nil
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
			return nil, nil
		}
	})}

	candles, err := p.GetCandles(context.Background(), "005930", "1m", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) != 2 {
		t.Fatalf("unexpected candle len: %d", len(candles))
	}
	if candles[0].Open <= 0 || candles[0].High <= 0 || candles[0].Low <= 0 || candles[0].Close <= 0 {
		t.Fatalf("signed prices should be normalized to positive values: %+v", candles[0])
	}
}
