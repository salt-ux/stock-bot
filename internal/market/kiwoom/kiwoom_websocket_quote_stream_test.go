package kiwoom

import (
	"context"
	"testing"
	"time"
)

func TestResolveKiwoomWebSocketURL(t *testing.T) {
	got := resolveKiwoomWebSocketURL("https://mockapi.kiwoom.com", "", "/api/dostk/websocket")
	if got != "wss://mockapi.kiwoom.com:10000/api/dostk/websocket" {
		t.Fatalf("unexpected ws url: %s", got)
	}
	got = resolveKiwoomWebSocketURL("https://mockapi.kiwoom.com", "wss://custom/ws", "/ignored")
	if got != "wss://custom/ws" {
		t.Fatalf("unexpected override ws url: %s", got)
	}
}

func TestKiwoomWebSocketQuoteStreamWaitLatest(t *testing.T) {
	stream := newKiwoomWebSocketQuoteStream("wss://mock", "0B")
	stream.consumePayload([]byte(`{"symbol":"005930","cur_prc":"70100","dt":"20260221","tm":"153000"}`))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	quote, ok := stream.WaitLatest(ctx, "005930", time.Second)
	if !ok {
		t.Fatal("expected websocket quote")
	}
	if quote.Price != 70100 {
		t.Fatalf("unexpected price: %f", quote.Price)
	}
}
