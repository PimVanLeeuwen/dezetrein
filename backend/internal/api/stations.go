package api

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/geo"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// The station list changes a few times a year: one NS call a day covers every user.
const (
	stationsTTL       = 24 * time.Hour
	nearestStations   = 5
	nearestStationsKm = 25
)

type nearbyStation struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	DistanceM int    `json:"distanceM"`
}

type nearestResponse struct {
	Stations []nearbyStation `json:"stations"`
	AsOf     time.Time       `json:"as_of"`
	Stale    bool            `json:"stale"`
}

// stationsNearest lists the stations with departure times closest to the newest fix, for the
// departures picker. Like match, the fixes are used only inside this handler.
func (s *Server) stationsNearest(w http.ResponseWriter, r *http.Request) {
	fixes, ok := s.readFixes(w, r)
	if !ok {
		return
	}
	if s.NS == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	res, err := s.stations.Get(r.Context(), "all", func(ctx context.Context) ([]ns.Station, error) { return s.NS.Stations(ctx) })
	if err != nil {
		s.fail(w, "stations", err)
		return
	}
	f := fixes[0]
	for _, x := range fixes[1:] {
		if x.UnixMS > f.UnixMS {
			f = x
		}
	}
	near := []nearbyStation{}
	for _, st := range res.Value {
		if !st.Departures {
			continue
		}
		if d := geo.DistanceM(f.Lat, f.Lng, st.Lat, st.Lng); d <= nearestStationsKm*1000 {
			near = append(near, nearbyStation{Code: st.Code, Name: st.Name, DistanceM: int(d)})
		}
	}
	sort.Slice(near, func(i, j int) bool { return near[i].DistanceM < near[j].DistanceM })
	if len(near) > nearestStations {
		near = near[:nearestStations]
	}
	writeJSON(w, http.StatusOK, nearestResponse{Stations: near, AsOf: res.AsOf, Stale: res.Stale})
}
