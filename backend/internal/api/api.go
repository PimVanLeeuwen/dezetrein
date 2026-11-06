// Package api is the dezetrein HTTP API. For now it serves only the health check; the match,
// carriage and trip endpoints follow later in Phase 2.
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/live"
)

// Server holds what the handlers need.
type Server struct {
	Version string
	// Vehicles is the shared vehicle feed; nil when no NS key is configured.
	Vehicles interface{ Snapshot() live.Snapshot }
}

// Handler returns the routes with security headers applied.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	})
	return headers(mux)
}

// healthz is a liveness check: it answers 200 while the process runs, and reports the vehicle
// feed's age so a monitor can alert on stale NS data separately.
func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	body := map[string]any{"ok": true, "version": s.Version}
	if s.Vehicles != nil {
		snap := s.Vehicles.Snapshot()
		feed := map[string]any{"count": len(snap.Vehicles), "stale": snap.Stale}
		if !snap.AsOf.IsZero() {
			feed["as_of"] = snap.AsOf.UTC().Format(time.RFC3339)
		}
		body["vehicles"] = feed
	}
	writeJSON(w, http.StatusOK, body)
}

func headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
