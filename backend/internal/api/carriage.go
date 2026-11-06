package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/geo"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/live"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/match"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// NS has no lookup from a carriage (trainset) number to its train: the link exists only per
// journey. So the site remembers every link it sees in the compositions and journeys it fetches
// anyway (a passive index, no extra NS calls), and with a location it checks the compositions of
// the trains nearby.
const (
	// carriageTTL is how long a seen link is trusted, the same as a cached composition.
	carriageTTL = compositionTTL
	// maxIndexed caps the index; the network has a few thousand trainsets.
	maxIndexed = 20000
	// nearbyTrains is how many of the trains closest to the user have their composition checked.
	nearbyTrains = 12
	// lookupTimeout bounds the nearby search; the fetches carry on in the cache.
	lookupTimeout = 8 * time.Second
)

// Trainset numbers are digits, for example 9402 (VIRM) or 3304 (ICNG); locomotives have six.
var carriagePattern = regexp.MustCompile(`^[0-9]{1,6}$`)

// carriageIndex maps a carriage number to the train it was last seen in.
type carriageIndex struct {
	now func() time.Time
	mu  sync.Mutex
	m   map[string]seenIn
}

type seenIn struct {
	train string
	at    time.Time
}

// record notes that the carriages run in train. A later sighting in another train wins.
func (x *carriageIndex) record(train string, carriages []string) {
	if train == "" || len(carriages) == 0 {
		return
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	now := x.now()
	if x.m == nil {
		x.m = map[string]seenIn{}
	}
	if len(x.m)+len(carriages) > maxIndexed {
		for k, v := range x.m {
			if now.Sub(v.at) >= carriageTTL {
				delete(x.m, k)
			}
		}
		if len(x.m)+len(carriages) > maxIndexed {
			clear(x.m)
		}
	}
	for _, c := range carriages {
		x.m[c] = seenIn{train: train, at: now}
	}
}

// lookup returns the train the carriage was last seen in, if that was within carriageTTL.
func (x *carriageIndex) lookup(carriage string) (string, time.Time, bool) {
	x.mu.Lock()
	defer x.mu.Unlock()
	v, ok := x.m[carriage]
	if !ok || x.now().Sub(v.at) >= carriageTTL {
		return "", time.Time{}, false
	}
	return v.train, v.at, true
}

// journey and composition fetch through the shared caches and feed the carriage index.
func (s *Server) journey(ctx context.Context, train string) (live.Result[*ns.Journey], error) {
	return s.journeys.Get(ctx, train, func(ctx context.Context) (*ns.Journey, error) {
		j, err := s.NS.Journey(ctx, train)
		if err == nil {
			seen := map[string]bool{}
			var all []string
			for _, st := range j.Stops {
				for _, c := range st.Carriages {
					if !seen[c] {
						seen[c] = true
						all = append(all, c)
					}
				}
			}
			s.carriages.record(j.Train, all)
		}
		return j, err
	})
}

func (s *Server) composition(ctx context.Context, train string) (live.Result[*ns.Composition], error) {
	return s.compositions.Get(ctx, train, func(ctx context.Context) (*ns.Composition, error) {
		c, err := s.NS.Composition(ctx, train)
		if err == nil {
			s.carriages.record(c.Train, c.Carriages)
		}
		return c, err
	})
}

type carriageRequest struct {
	Carriage string  `json:"carriage"`
	Fixes    []fixIn `json:"fixes"`
}

// carriageResponse names the train running the carriage, or Train null when it was not found.
// Located says whether a location was used: when false, the page can ask for one.
type carriageResponse struct {
	Carriage string        `json:"carriage"`
	Train    *candidateOut `json:"train"`
	Source   string        `json:"source,omitempty"` // "index" or "nearby"
	Located  bool          `json:"located"`
	Searched int           `json:"searched"` // nearby trains checked
	AsOf     time.Time     `json:"as_of,omitzero"`
	Stale    bool          `json:"stale"`
}

// carriage finds the train a carriage number runs in. Like match, the fixes are used only here.
func (s *Server) carriage(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxMatchBody))
	dec.DisallowUnknownFields()
	var req carriageRequest
	if err := dec.Decode(&req); err != nil || dec.Decode(&struct{}{}) != io.EOF {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeError(w, http.StatusRequestEntityTooLarge, "too_large")
		} else {
			writeError(w, http.StatusBadRequest, "invalid_body")
		}
		return
	}
	if !carriagePattern.MatchString(req.Carriage) {
		writeError(w, http.StatusBadRequest, "invalid_carriage")
		return
	}
	if len(req.Fixes) > maxFixes {
		writeError(w, http.StatusBadRequest, "invalid_fixes")
		return
	}
	now := s.clock()
	fixes := make([]match.Fix, 0, len(req.Fixes))
	for _, f := range req.Fixes {
		fix, ok := validFix(f, now)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid_fixes")
			return
		}
		fixes = append(fixes, fix)
	}
	if s.NS == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	resp := carriageResponse{Carriage: req.Carriage, Located: len(fixes) > 0}
	ctx, cancel := context.WithTimeout(r.Context(), lookupTimeout)
	defer cancel()

	if train, at, ok := s.carriages.lookup(req.Carriage); ok {
		resp.Source, resp.AsOf = "index", at
		resp.Train = s.describeTrain(ctx, train, "", nil)
		writeJSON(w, http.StatusOK, resp)
		return
	}
	if len(fixes) > 0 && s.Vehicles != nil {
		snap := s.Vehicles.Snapshot()
		near := nearest(fixes, snap.Vehicles, nearbyTrains)
		resp.Searched, resp.AsOf, resp.Stale = len(near), snap.AsOf, snap.Stale
		if c := s.findIn(ctx, near, req.Carriage); c != nil {
			resp.Source = "nearby"
			resp.Train = s.describeTrain(ctx, c.vehicle.Number(), c.vehicle.Type, &c.distanceM)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

type nearTrain struct {
	vehicle   ns.Vehicle
	distanceM float64
}

// nearest is up to n trains closest to the newest fix, within the matcher's search radius.
func nearest(fixes []match.Fix, vehicles []ns.Vehicle, n int) []nearTrain {
	f := fixes[0]
	for _, x := range fixes[1:] {
		if x.UnixMS > f.UnixMS {
			f = x
		}
	}
	radius := min(max(2*f.AccuracyM, 3000), 10000) + 2000
	var out []nearTrain
	for _, v := range vehicles {
		if d := geo.DistanceM(f.Lat, f.Lng, v.Lat, v.Lng); d <= radius {
			out = append(out, nearTrain{vehicle: v, distanceM: d})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].distanceM < out[j].distanceM })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// findIn checks the trains' compositions in parallel and returns the nearest that has the
// carriage. Trains whose composition is unknown or does not arrive in time are skipped.
func (s *Server) findIn(ctx context.Context, trains []nearTrain, carriage string) *nearTrain {
	found := make([]bool, len(trains))
	var wg sync.WaitGroup
	for i, t := range trains {
		wg.Go(func() {
			res, err := s.composition(ctx, t.vehicle.Number())
			s.logExtra("composition", err)
			if err == nil && res.Value != nil {
				for _, c := range res.Value.Carriages {
					if c == carriage {
						found[i] = true
					}
				}
			}
		})
	}
	wg.Wait()
	for i := range trains {
		if found[i] {
			return &trains[i]
		}
	}
	return nil
}
