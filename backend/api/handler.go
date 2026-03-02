// Package api implements the public REST JSON API consumed by the Hugo widget.
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/meirongdev/timeslot/config"
	"github.com/meirongdev/timeslot/engine"
)

// Handler holds dependencies for the public API.
type Handler struct {
	Cfg    *config.Config
	Engine *engine.Engine
}

// RegisterRoutes attaches all public API routes to mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/slots", h.withCORS(h.handleSlots))
	mux.HandleFunc("/api/schedule", h.withCORS(h.handleSchedule))
}

// ---- GET /api/slots ---------------------------------------------------------

func (h *Handler) handleSlots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	from, to, err := parseTimeRange(q.Get("from"), q.Get("to"))
	if err != nil {
		http.Error(w, "invalid from/to parameters (use RFC3339)", http.StatusBadRequest)
		return
	}
	// Cap to MaxDaysAhead.
	maxTo := time.Now().Add(time.Duration(h.Cfg.MaxDaysAhead) * 24 * time.Hour)
	if to.After(maxTo) {
		to = maxTo
	}

	slots, err := h.Engine.ComputeSlots(from, to)
	if err != nil {
		log.Printf("api: compute slots: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, slots)
}

// ---- GET /api/schedule ------------------------------------------------------

// handleSchedule returns a privacy-first schedule: each entry is "available" or
// "occupied" with no internal event details exposed.
func (h *Handler) handleSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	from, to, err := parseTimeRange(q.Get("from"), q.Get("to"))
	if err != nil {
		http.Error(w, "invalid from/to parameters (use RFC3339)", http.StatusBadRequest)
		return
	}
	maxTo := time.Now().Add(time.Duration(h.Cfg.MaxDaysAhead) * 24 * time.Hour)
	if to.After(maxTo) {
		to = maxTo
	}

	entries, err := h.Engine.ComputeSchedule(from, to)
	if err != nil {
		log.Printf("api: compute schedule: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, entries)
}

// ---- helpers ----------------------------------------------------------------

func parseTimeRange(fromStr, toStr string) (time.Time, time.Time, error) {
	from := time.Now()
	to := from.Add(7 * 24 * time.Hour)
	var err error
	if fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	if toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	return from, to, nil
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}
