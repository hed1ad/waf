package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/hediad/waf/api/internal/store"
)

type EventsHandler struct {
	ch *store.ClickHouseStore
}

func NewEventsHandler(ch *store.ClickHouseStore) *EventsHandler {
	return &EventsHandler{ch: ch}
}

func (h *EventsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	from := parseTime(q.Get("from"), time.Now().Add(-24*time.Hour))
	to := parseTime(q.Get("to"), time.Now())
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	filter := store.EventFilter{
		From:     from,
		To:       to,
		Action:   q.Get("action"),
		ClientIP: q.Get("ip"),
		Country:  q.Get("country"),
		Limit:    limit,
		Offset:   offset,
	}

	events, err := h.ch.ListEvents(r.Context(), filter)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": events})
}

func (h *EventsHandler) Stats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := parseTime(q.Get("from"), time.Now().Add(-24*time.Hour))
	to := parseTime(q.Get("to"), time.Now())

	total, blocked, err := h.ch.CountEvents(r.Context(), from, to)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}

	timeseries, err := h.ch.GetTimeseries(r.Context(), from, to)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}

	topIPs, err := h.ch.GetTopIPs(r.Context(), 10)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"total":      total,
		"blocked":    blocked,
		"timeseries": timeseries,
		"top_ips":    topIPs,
	})
}

func parseTime(s string, def time.Time) time.Time {
	if s == "" {
		return def
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return def
	}
	return t
}

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
