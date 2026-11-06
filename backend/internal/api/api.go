// Package api is the dezetrein HTTP API: the health check, the trip view of one train, the
// departures picker, the location match and the carriage-number lookup. Every NS call goes through the shared caches in
// package live, and an NS failure serves the last good copy marked stale instead of an error.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/live"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// NS is the part of the NS client the handlers use; *ns.Client implements it.
type NS interface {
	Journey(ctx context.Context, train string) (*ns.Journey, error)
	Composition(ctx context.Context, train string) (*ns.Composition, error)
	Crowding(ctx context.Context, train string) (ns.Crowding, error)
	Disruptions(ctx context.Context) ([]ns.Disruption, error)
	Departures(ctx context.Context, station string) ([]ns.Departure, error)
}

// Feed is the shared vehicle list; *live.Poller implements it.
type Feed interface {
	Snapshot() live.Snapshot
	History() []live.Snapshot
}

// Server holds what the handlers need. NS and Vehicles are nil when no NS key is configured;
// the NS routes then answer 503.
type Server struct {
	Version  string
	Vehicles Feed
	NS       NS
	// Log receives NS failures. It never gets locations, IP addresses or request bodies.
	Log *slog.Logger

	now func() time.Time // tests replace it

	once         sync.Once
	journeys     *live.Cache[*ns.Journey]
	compositions *live.Cache[*ns.Composition]
	crowding     *live.Cache[ns.Crowding]
	disruptions  *live.Cache[[]ns.Disruption]
	departures   *live.Cache[[]ns.Departure]
	carriages    *carriageIndex
	apiLimit     *limiter
	matchLimit   *limiter
}

// Cache lifetimes. Times and delays change by the minute; a composition changes only at a stop
// where the train is split or joined; disruptions are one call for the whole network.
const (
	journeyTTL     = 30 * time.Second
	compositionTTL = 10 * time.Minute
	crowdingTTL    = 5 * time.Minute
	disruptionsTTL = time.Minute
	departuresTTL  = 30 * time.Second
	// notFoundTTL keeps NS from being asked again and again about a train number that does not
	// run today.
	notFoundTTL = 5 * time.Minute
)

func (s *Server) init() {
	s.once.Do(func() {
		s.journeys = &live.Cache[*ns.Journey]{TTL: journeyTTL, NotFoundTTL: notFoundTTL}
		s.compositions = &live.Cache[*ns.Composition]{TTL: compositionTTL, NotFoundTTL: notFoundTTL}
		s.crowding = &live.Cache[ns.Crowding]{TTL: crowdingTTL, NotFoundTTL: notFoundTTL}
		s.disruptions = &live.Cache[[]ns.Disruption]{TTL: disruptionsTTL, MaxEntries: 1}
		s.departures = &live.Cache[[]ns.Departure]{TTL: departuresTTL, NotFoundTTL: notFoundTTL}
		s.carriages = &carriageIndex{now: s.clock}
		s.apiLimit = newLimiter(apiPerMinute, apiBurst, s.clock)
		s.matchLimit = newLimiter(matchPerMinute, matchBurst, s.clock)
		if s.Log == nil {
			s.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
		}
	})
}

func (s *Server) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// Handler returns the routes with request logging, security headers and, on /api, per-client
// rate limits. /api/match and /api/carriage, which can cost several NS calls, count against a
// shared lookup limit as well as the general one.
func (s *Server) Handler() http.Handler {
	s.init()
	api := func(h http.HandlerFunc) http.HandlerFunc { return limit(s.apiLimit, h) }
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /api/trains/{train}", api(s.train))
	mux.HandleFunc("GET /api/stations/{code}/departures", api(s.stationDepartures))
	mux.HandleFunc("POST /api/match", api(limit(s.matchLimit, s.match)))
	mux.HandleFunc("POST /api/carriage", api(limit(s.matchLimit, s.carriage)))
	mux.HandleFunc("/api/", api(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found")
	}))
	return headers(s.logRequests(mux))
}

// statusRecorder remembers the status code a handler wrote, for the request log.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// logRequests writes one line per request with the route pattern (for example
// "GET /api/trains/{train}"), never the actual path or query, and no IP address, headers or
// body. Health checks log at debug level, so the monitor's polling does not fill the log.
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := s.clock()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		route := r.Pattern // set by the mux on this request while routing
		if route == "" {
			route = "unmatched"
		}
		level := slog.LevelInfo
		if route == "GET /healthz" {
			level = slog.LevelDebug
		}
		s.Log.Log(r.Context(), level, "request", "route", route, "status", rec.status, "ms", s.clock().Sub(start).Milliseconds())
	})
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

var (
	// Journey numbers are up to six digits (NS uses 1 to 99999 plus some six-digit specials).
	trainPattern = regexp.MustCompile(`^[0-9]{1,6}$`)
	// Station codes are short letter codes such as UT, ASDZ or BDPB.
	stationPattern = regexp.MustCompile(`^[A-Za-z]{1,8}$`)
)

// fail answers an NS failure that left nothing to serve: 404 when NS does not know the train
// or station, else 503. Never a 500.
func (s *Server) fail(w http.ResponseWriter, what string, err error) {
	if errors.Is(err, ns.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if !errors.Is(err, context.Canceled) {
		s.Log.Warn("ns fetch failed", "what", what, "err", err)
	}
	writeError(w, http.StatusServiceUnavailable, "unavailable")
}

func headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		// The API only ever returns JSON: nothing in it may load, run or be framed.
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError answers with {"error": code}; codes are stable strings the frontend translates.
func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}
