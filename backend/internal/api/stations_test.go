package api

import (
	"net/http"
	"strings"
	"testing"
)

func postNearest(t *testing.T, url, body string) *http.Response {
	t.Helper()
	res, err := http.Post(url+"/api/stations/nearest", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func TestNearestStations(t *testing.T) {
	fake := newFakeNS(t)
	srv := newTestServer(t, &Server{NS: fake})
	// On the platforms of Utrecht Centraal.
	res := postNearest(t, srv.URL, `{"fixes":[`+fixJSON(52.0894, 5.1101, 30)+`]}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	b := decode[nearestResponse](t, res)
	if len(b.Stations) != nearestStations || b.Stations[0].Code != "UT" || b.Stations[0].Name != "Utrecht Centraal" || b.Stations[0].DistanceM > 100 {
		t.Fatalf("unexpected: %+v", b.Stations)
	}
	for i := 1; i < len(b.Stations); i++ {
		if b.Stations[i].DistanceM < b.Stations[i-1].DistanceM {
			t.Errorf("not sorted by distance: %+v", b.Stations)
		}
	}
	if b.AsOf.IsZero() || b.Stale {
		t.Errorf("as_of %v stale %v", b.AsOf, b.Stale)
	}
	postNearest(t, srv.URL, `{"fixes":[`+fixJSON(52.3, 4.76, 30)+`]}`)
	if n := fake.count("stations"); n != 1 {
		t.Errorf("%d station list fetches, want 1 (cached for a day)", n)
	}
}

func TestNearestStationsFarFromAny(t *testing.T) {
	srv := newTestServer(t, &Server{NS: newFakeNS(t)})
	// In the Waddenzee, more than 25 km from a station with departures.
	b := decode[nearestResponse](t, postNearest(t, srv.URL, `{"fixes":[`+fixJSON(53.35, 5.0, 30)+`]}`))
	if b.Stations == nil || len(b.Stations) != 0 {
		t.Errorf("want an empty list, got %+v", b.Stations)
	}
}

func TestNearestStationsErrors(t *testing.T) {
	fake := newFakeNS(t)
	srv := newTestServer(t, &Server{NS: fake})
	if res := postNearest(t, srv.URL, `{"fixes":[]}`); res.StatusCode != http.StatusBadRequest {
		t.Errorf("no fixes: %d", res.StatusCode)
	}
	if res := postNearest(t, srv.URL, `{"fixes":[`+fixJSON(40, -3, 30)+`]}`); res.StatusCode != http.StatusBadRequest {
		t.Errorf("outside the network: %d", res.StatusCode)
	}
	fake.setFail("stations", true)
	if res := postNearest(t, srv.URL, `{"fixes":[`+fixJSON(52.09, 5.11, 30)+`]}`); res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("NS down, nothing cached: %d", res.StatusCode)
	}
	off := newTestServer(t, &Server{})
	if res := postNearest(t, off.URL, `{"fixes":[`+fixJSON(52.09, 5.11, 30)+`]}`); res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("no NS key: %d", res.StatusCode)
	}
}
