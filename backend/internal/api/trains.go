package api

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/live"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// trainResponse is the trip view of one train: the journey with crowding merged into its stops,
// the composition, the disruptions on its route and its live position.
type trainResponse struct {
	Train        string          `json:"train"`
	Category     string          `json:"category"`
	CategoryName string          `json:"categoryName"`
	Destination  string          `json:"destination"`
	Stops        []ns.Stop       `json:"stops"`
	Composition  *ns.Composition `json:"composition"`
	Disruptions  []ns.Disruption `json:"disruptions"`
	Position     *position       `json:"position"`
	AsOf         time.Time       `json:"as_of"`
	Stale        bool            `json:"stale"`
}

type position struct {
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	SpeedKmh   float64   `json:"speedKmh"`
	HeadingDeg float64   `json:"headingDeg"`
	AsOf       time.Time `json:"as_of"`
}

func (s *Server) train(w http.ResponseWriter, r *http.Request) {
	train := r.PathValue("train")
	if !trainPattern.MatchString(train) {
		writeError(w, http.StatusBadRequest, "invalid_train")
		return
	}
	if s.NS == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	ctx := r.Context()
	var (
		wg      sync.WaitGroup
		journey live.Result[*ns.Journey]
		jErr    error
		comp    *ns.Composition
		crowd   ns.Crowding
		disr    []ns.Disruption
		mu      sync.Mutex
		stale   bool
	)
	// A missing extra (composition, crowding, disruptions) leaves its field empty; it only marks
	// the response stale when NS failed, not when NS simply has no data for this train.
	extra := func(what string, res bool, err error) {
		mu.Lock()
		defer mu.Unlock()
		if res {
			stale = true
		}
		if err != nil && !errors.Is(err, ns.ErrNotFound) {
			stale = true
		}
		s.logExtra(what, err)
	}
	wg.Go(func() {
		journey, jErr = s.journey(ctx, train)
	})
	wg.Go(func() {
		res, err := s.composition(ctx, train)
		comp = res.Value
		extra("composition", res.Stale, err)
	})
	wg.Go(func() {
		res, err := s.crowding.Get(ctx, train, func(ctx context.Context) (ns.Crowding, error) { return s.NS.Crowding(ctx, train) })
		crowd = res.Value
		extra("crowding", res.Stale, err)
	})
	wg.Go(func() {
		res, err := s.disruptions.Get(ctx, "all", s.NS.Disruptions)
		disr = res.Value
		extra("disruptions", res.Stale, err)
	})
	wg.Wait()
	if jErr != nil {
		s.fail(w, "journey", jErr)
		return
	}
	j := journey.Value
	resp := trainResponse{
		Train:        j.Train,
		Category:     j.Category,
		CategoryName: j.CategoryName,
		Destination:  j.Destination,
		Stops:        withCrowding(j.Stops, crowd),
		Composition:  comp,
		Disruptions:  onRoute(disr, j.Stops),
		Position:     s.position(train),
		AsOf:         journey.AsOf,
		Stale:        journey.Stale || stale,
	}
	writeJSON(w, http.StatusOK, resp)
}

// withCrowding copies the stops (the cached journey is shared and must not change) and fills in
// the forecast from the crowding endpoint where the journey has none.
func withCrowding(stops []ns.Stop, crowd ns.Crowding) []ns.Stop {
	out := slices.Clone(stops)
	for i, st := range out {
		if c := crowd[st.UIC]; c != "" && (st.Crowd == "" || st.Crowd == "UNKNOWN") && !st.Passing() {
			out[i].Crowd = c
		}
	}
	return out
}

// onRoute keeps the disruptions that touch a station on the route (passing stations included),
// plus calamities, which name no stations and concern the whole network.
func onRoute(all []ns.Disruption, stops []ns.Stop) []ns.Disruption {
	route := map[string]bool{}
	for _, st := range stops {
		route[strings.ToUpper(st.Code)] = true
	}
	out := []ns.Disruption{}
	for _, d := range all {
		if len(d.Stations) == 0 {
			if d.Type == "CALAMITY" {
				out = append(out, d)
			}
			continue
		}
		if slices.ContainsFunc(d.Stations, func(c string) bool { return route[strings.ToUpper(c)] }) {
			out = append(out, d)
		}
	}
	return out
}

// position is the train's place in the shared vehicle feed, or nil when it is not in it.
func (s *Server) position(train string) *position {
	if s.Vehicles == nil {
		return nil
	}
	snap := s.Vehicles.Snapshot()
	for _, v := range snap.Vehicles {
		if v.Number() == train {
			return &position{Lat: v.Lat, Lng: v.Lng, SpeedKmh: v.SpeedKmh, HeadingDeg: v.HeadingDeg, AsOf: snap.AsOf}
		}
	}
	return nil
}

type departuresResponse struct {
	Station    string         `json:"station"`
	Departures []ns.Departure `json:"departures"`
	AsOf       time.Time      `json:"as_of"`
	Stale      bool           `json:"stale"`
}

func (s *Server) stationDepartures(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if !stationPattern.MatchString(code) {
		writeError(w, http.StatusBadRequest, "invalid_station")
		return
	}
	if s.NS == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	code = strings.ToUpper(code)
	res, err := s.departures.Get(r.Context(), code, func(ctx context.Context) ([]ns.Departure, error) { return s.NS.Departures(ctx, code) })
	if err != nil {
		s.fail(w, "departures", err)
		return
	}
	deps := res.Value
	if deps == nil {
		deps = []ns.Departure{}
	}
	writeJSON(w, http.StatusOK, departuresResponse{Station: code, Departures: deps, AsOf: res.AsOf, Stale: res.Stale})
}
