package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type sseEventEnvelope struct {
	Type string `json:"type"`
	At   string `json:"at"`
	Data any    `json:"data"`
}

type sseEventsHub struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

func newSSEEventsHub() *sseEventsHub {
	return &sseEventsHub{
		clients: map[chan string]struct{}{},
	}
}

func (h *sseEventsHub) Subscribe() chan string {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan string, 16)
	h.clients[ch] = struct{}{}
	return ch
}

func (h *sseEventsHub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[ch]; !ok {
		return
	}
	delete(h.clients, ch)
	close(ch)
}

func (h *sseEventsHub) Publish(eventType string, payload any) {
	envelope := sseEventEnvelope{
		Type: eventType,
		At:   time.Now().UTC().Format(time.RFC3339Nano),
		Data: payload,
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return
	}
	frame := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, raw)

	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- frame:
		default:
		}
	}
}

func sseEventsHandler(hub *sseEventsHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		client := hub.Subscribe()
		defer hub.Unsubscribe(client)

		_, _ = fmt.Fprint(w, ": connected\n\n")
		flusher.Flush()

		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case msg := <-client:
				_, _ = fmt.Fprint(w, msg)
				flusher.Flush()
			case <-heartbeat.C:
				_, _ = fmt.Fprint(w, ": ping\n\n")
				flusher.Flush()
			}
		}
	}
}
