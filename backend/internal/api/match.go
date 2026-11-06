package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/match"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// Limits on a match request. The area is the NS network with its border routes, generously.
const (
	maxMatchBody   = 4 << 10
	maxFixes       = 5
	minLat, maxLat = 49.0, 55.0
	minLng, maxLng = 2.0, 11.0
	maxAccuracyM   = 20000
	maxSpeedMS     = 100 // 360 km/h
	maxFixAge      = 10 * time.Minute
	maxFixAhead    = time.Minute // phone clocks drift
	// enrichTimeout bounds the wait for candidate details; the fetches carry on in the cache.
	enrichTimeout = 5 * time.Second
)

// fixIn is one fix as the browser sends it. Speed and heading are null when the phone does not
// report them; t is the Geolocation API timestamp in milliseconds since 1970.
type fixIn struct {
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
	Accuracy *float64 `json:"accuracy"`
	Speed    *float64 `json:"speed"`
	Heading  *float64 `json:"heading"`
	T        int64    `json:"t"`
}

type matchRequest struct {
	Fixes []fixIn `json:"fixes"`
}

type candidateOut struct {
	Train       string   `json:"train"`
	Category    string   `json:"category,omitempty"`
	Destination string   `json:"destination,omitempty"`
	Carriages   []string `json:"carriages"`
	DistanceM   *int     `json:"distanceM,omitempty"` // from the user; absent for an index hit
}

type matchResponse struct {
	Match      *string          `json:"match"` // set only for a high-confidence match
	Confidence match.Confidence `json:"confidence"`
	Candidates []candidateOut   `json:"candidates"`
	AsOf       time.Time        `json:"as_of"`
	Stale      bool             `json:"stale"`
}

// match ranks the trains near the posted fixes. The fixes exist only inside this handler: they
// are not logged, cached or kept.
func (s *Server) match(w http.ResponseWriter, r *http.Request) {
	fixes, ok := s.readFixes(w, r)
	if !ok {
		return
	}
	if s.Vehicles == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	snap := s.Vehicles.Snapshot()
	if snap.AsOf.IsZero() {
		writeError(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	hist := s.Vehicles.History()
	frames := make([]match.Frame, len(hist))
	for i, h := range hist {
		frames[i] = match.Frame{At: h.AsOf, Vehicles: h.Vehicles}
	}
	res := match.Match(fixes, frames)
	resp := matchResponse{Confidence: res.Confidence, Candidates: s.describe(r.Context(), res.Candidates), AsOf: snap.AsOf, Stale: snap.Stale}
	if res.Confidence == match.High {
		resp.Match = &resp.Candidates[0].Train
	}
	writeJSON(w, http.StatusOK, resp)
}

// readFixes decodes and validates the body, answering 400 itself when it is not acceptable.
func (s *Server) readFixes(w http.ResponseWriter, r *http.Request) ([]match.Fix, bool) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxMatchBody))
	dec.DisallowUnknownFields()
	var req matchRequest
	if err := dec.Decode(&req); err != nil || dec.Decode(&struct{}{}) != io.EOF {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeError(w, http.StatusRequestEntityTooLarge, "too_large")
		} else {
			writeError(w, http.StatusBadRequest, "invalid_body")
		}
		return nil, false
	}
	if len(req.Fixes) == 0 || len(req.Fixes) > maxFixes {
		writeError(w, http.StatusBadRequest, "invalid_fixes")
		return nil, false
	}
	now := s.clock()
	fixes := make([]match.Fix, 0, len(req.Fixes))
	for _, f := range req.Fixes {
		fix, ok := validFix(f, now)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid_fixes")
			return nil, false
		}
		fixes = append(fixes, fix)
	}
	return fixes, true
}

func validFix(f fixIn, now time.Time) (match.Fix, bool) {
	if f.Lat == nil || f.Lng == nil || f.Accuracy == nil {
		return match.Fix{}, false
	}
	lat, lng, acc := *f.Lat, *f.Lng, *f.Accuracy
	if !within(lat, minLat, maxLat) || !within(lng, minLng, maxLng) || !within(acc, 0, maxAccuracyM) || acc == 0 {
		return match.Fix{}, false
	}
	if f.Speed != nil && !within(*f.Speed, 0, maxSpeedMS) {
		return match.Fix{}, false
	}
	if f.Heading != nil && !within(*f.Heading, 0, 360) {
		return match.Fix{}, false
	}
	t := time.UnixMilli(f.T)
	if f.T <= 0 || t.Before(now.Add(-maxFixAge)) || t.After(now.Add(maxFixAhead)) {
		return match.Fix{}, false
	}
	return match.Fix{Lat: lat, Lng: lng, AccuracyM: acc, SpeedMS: f.Speed, Heading: f.Heading, UnixMS: f.T}, true
}

func within(v, lo, hi float64) bool { return !math.IsNaN(v) && v >= lo && v <= hi }

// describe adds what users recognise a train by, its destination and carriage numbers, to each
// candidate. Missing details are left empty: a slow or failing NS call never blocks the match.
func (s *Server) describe(ctx context.Context, cands []match.Candidate) []candidateOut {
	out := make([]candidateOut, len(cands))
	if len(cands) == 0 {
		return out
	}
	ctx, cancel := context.WithTimeout(ctx, enrichTimeout)
	defer cancel()
	var wg sync.WaitGroup
	for i, c := range cands {
		wg.Go(func() { out[i] = *s.describeTrain(ctx, c.Vehicle.Number(), c.Vehicle.Type, &c.DistanceM) })
	}
	wg.Wait()
	return out
}

// describeTrain fetches one train's destination and carriages (through the caches) in
// parallel. category (from the vehicle feed) and distanceM are optional; the journey's category
// wins when there is one.
func (s *Server) describeTrain(ctx context.Context, train, category string, distanceM *float64) *candidateOut {
	out := &candidateOut{Train: train, Category: category, Carriages: []string{}}
	if distanceM != nil {
		d := int(math.Round(*distanceM))
		out.DistanceM = &d
	}
	if s.NS == nil {
		return out
	}
	var (
		wg          sync.WaitGroup
		destination string
		jCategory   string
		carriages   []string
	)
	wg.Go(func() {
		res, err := s.journey(ctx, train)
		if err == nil {
			destination, jCategory = res.Value.Destination, res.Value.Category
		}
		s.logExtra("journey", err)
	})
	wg.Go(func() {
		res, err := s.composition(ctx, train)
		if err == nil && res.Value != nil {
			carriages = res.Value.Carriages
		}
		s.logExtra("composition", err)
	})
	wg.Wait()
	out.Destination = destination
	// The journey's category is the precise one (ICD for an Intercity direct); the vehicle feed
	// only knows IC, SPR and ARR.
	if jCategory != "" {
		out.Category = jCategory
	}
	if len(carriages) > 0 {
		out.Carriages = carriages
	}
	return out
}

// logExtra logs a failed optional NS call. "Not found" is normal (not every train has a
// composition) and a caller that left is not a failure.
func (s *Server) logExtra(what string, err error) {
	if err != nil && !errors.Is(err, ns.ErrNotFound) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		s.Log.Warn("ns fetch failed", "what", what, "err", err)
	}
}
