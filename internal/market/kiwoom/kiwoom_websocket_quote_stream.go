package kiwoom

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/salt-ux/stock-bot/internal/market"
)

type websocketDialFunc func(ctx context.Context, urlStr string, reqHeader http.Header) (*websocket.Conn, *http.Response, error)

type kiwoomWebSocketQuoteStream struct {
	endpoint    string
	quoteTRName string
	dialFn      websocketDialFunc

	startOnce sync.Once
	stopFn    context.CancelFunc

	mu         sync.RWMutex
	conn       *websocket.Conn
	subscribed map[string]struct{}
	lastQuote  map[string]market.Quote
}

func newKiwoomWebSocketQuoteStream(endpoint, quoteAPIID string) *kiwoomWebSocketQuoteStream {
	trName := strings.TrimSpace(quoteAPIID)
	if trName == "" {
		trName = "0B"
	}
	return &kiwoomWebSocketQuoteStream{
		endpoint:    endpoint,
		quoteTRName: trName,
		dialFn:      websocket.DefaultDialer.DialContext,
		subscribed:  map[string]struct{}{},
		lastQuote:   map[string]market.Quote{},
	}
}

func (s *kiwoomWebSocketQuoteStream) Start() {
	s.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		s.stopFn = cancel
		go s.runReconnectLoop(ctx)
	})
}

func (s *kiwoomWebSocketQuoteStream) Stop() {
	if s.stopFn != nil {
		s.stopFn()
	}
	s.mu.Lock()
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
	s.mu.Unlock()
}

func (s *kiwoomWebSocketQuoteStream) Subscribe(symbol string) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return
	}

	s.mu.Lock()
	s.subscribed[symbol] = struct{}{}
	conn := s.conn
	s.mu.Unlock()

	if conn != nil {
		_ = s.sendSubscribe(conn, symbol)
	}
}

func (s *kiwoomWebSocketQuoteStream) WaitLatest(ctx context.Context, symbol string, wait time.Duration) (market.Quote, bool) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return market.Quote{}, false
	}
	deadline := time.Now().Add(wait)
	for {
		if quote, ok := s.getLatest(symbol); ok {
			return quote, true
		}
		if time.Now().After(deadline) {
			return market.Quote{}, false
		}
		select {
		case <-ctx.Done():
			return market.Quote{}, false
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (s *kiwoomWebSocketQuoteStream) runReconnectLoop(ctx context.Context) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		if err := s.connectAndRead(ctx); err != nil {
			if shouldLogWSReconnectError(err) {
				log.Printf("[market/ws] reconnecting after error: %v", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 15*time.Second {
			backoff *= 2
			if backoff > 15*time.Second {
				backoff = 15 * time.Second
			}
		}
	}
}

func (s *kiwoomWebSocketQuoteStream) connectAndRead(ctx context.Context) error {
	headers := make(http.Header)
	conn, _, err := s.dialFn(ctx, s.endpoint, headers)
	if err != nil {
		return err
	}
	defer conn.Close()

	s.mu.Lock()
	s.conn = conn
	symbols := make([]string, 0, len(s.subscribed))
	for symbol := range s.subscribed {
		symbols = append(symbols, symbol)
	}
	s.mu.Unlock()

	for _, symbol := range symbols {
		if err := s.sendSubscribe(conn, symbol); err != nil {
			log.Printf("[market/ws] subscribe failed symbol=%s: %v", symbol, err)
		}
	}

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			s.mu.Lock()
			if s.conn == conn {
				s.conn = nil
			}
			s.mu.Unlock()
			return err
		}
		s.consumePayload(payload)
	}
}

func (s *kiwoomWebSocketQuoteStream) sendSubscribe(conn *websocket.Conn, symbol string) error {
	// 키움 공식 WebSocket 포맷(trnm/data)을 우선 사용합니다.
	primary := map[string]any{
		"trnm":    s.quoteTRName,
		"refresh": "0",
		"data": []map[string]string{
			{"item": symbol},
		},
	}
	if err := conn.WriteJSON(primary); err == nil {
		return nil
	}

	// 서버 구현 차이에 대비한 보조 포맷을 1회 시도합니다.
	fallback := map[string]any{
		"trnm": s.quoteTRName,
		"item": symbol,
	}
	return conn.WriteJSON(fallback)
}

func (s *kiwoomWebSocketQuoteStream) consumePayload(payload []byte) {
	var obj any
	if err := json.Unmarshal(payload, &obj); err != nil {
		return
	}

	record := findQuoteRecord(obj)
	if record == nil {
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(firstString(record, "symbol", "stk_cd", "isu_cd", "code", "item")))
	if symbol == "" {
		return
	}
	price, err := extractFloat(record, "cur_prc", "current_price", "stck_prpr", "close", "last", "price")
	if err != nil {
		return
	}
	at := extractTime(record, []string{"tm", "time", "stck_cntg_hour", "as_of"}, []string{"dt", "date", "stck_bsop_date", "as_of_date"})
	if at.IsZero() {
		at = time.Now().UTC()
	}

	s.mu.Lock()
	s.lastQuote[symbol] = market.Quote{Symbol: symbol, Price: price, AsOf: at}
	s.mu.Unlock()
}

func (s *kiwoomWebSocketQuoteStream) getLatest(symbol string) (market.Quote, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	quote, ok := s.lastQuote[symbol]
	return quote, ok
}

func shouldLogWSReconnectError(err error) bool {
	if err == nil {
		return false
	}
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		if closeErr.Code == websocket.CloseNormalClosure {
			return false
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}
