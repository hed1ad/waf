package handlers

import (
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"
)

const streamChannel = "waf:events:live"

type StreamHandler struct {
	redis *redis.Client
	log   *slog.Logger
}

func NewStreamHandler(redis *redis.Client, log *slog.Logger) *StreamHandler {
	return &StreamHandler{redis: redis, log: log}
}

// Stream handles WebSocket-like SSE connection for live event feed.
// Uses Server-Sent Events (SSE) — simpler than WS, works without upgrades.
func (h *StreamHandler) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sub := h.redis.Subscribe(r.Context(), streamChannel)
	defer sub.Close()

	ch := sub.Channel()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := w.Write([]byte("data: " + msg.Payload + "\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
