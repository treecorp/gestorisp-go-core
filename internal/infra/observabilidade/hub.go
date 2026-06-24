package observabilidade

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Nivel     string `json:"nivel"`
	Tag       string `json:"tag"`
	Instancia int    `json:"instancia"`
	Mensagem  string `json:"mensagem"`
	DuracaoMs int64  `json:"duracao_ms"`
	Servico   string `json:"servico"`
}

type SSEHub struct {
	clients map[chan LogEntry]struct{}
	mu      sync.RWMutex
}

var hub = &SSEHub{
	clients: make(map[chan LogEntry]struct{}),
}

func (h *SSEHub) Broadcast(entry LogEntry) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- entry:
		default:
		}
	}
}

func (h *SSEHub) subscribe() chan LogEntry {
	ch := make(chan LogEntry, 256)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *SSEHub) unsubscribe(ch chan LogEntry) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

func HandlerSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming nao suportado", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	ch := hub.subscribe()
	defer hub.unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(entry)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
