package api

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/live"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

func decode[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestTrainMergesJourneyCompositionAndDisruptions(t *testing.T) {
	feed := fakeFeed{AsOf: testNow, Vehicles: []ns.Vehicle{{RitID: "2450", Lat: 52.3, Lng: 4.8, SpeedKmh: 120, HeadingDeg: 200}}}
	srv := newTestServer(t, &Server{NS: newFakeNS(t), Vehicles: feed})
	res := get(t, srv.URL+"/api/trains/2450")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	body := decode[trainResponse](t, res)
	if body.Train != "2450" || body.Category != "ICD" || body.Destination != "Rotterdam Centraal" || len(body.Stops) != 15 || body.Stale || body.AsOf.IsZero() {
		t.Fatalf("unexpected header: %+v", body)
	}
	if body.Composition == nil || len(body.Composition.Carriages) == 0 {
		t.Errorf("want the composition: %+v", body.Composition)
	}
	if body.Position == nil || body.Position.Lat != 52.3 || body.Position.SpeedKmh != 120 || !body.Position.AsOf.Equal(testNow) {
		t.Errorf("want the live position: %+v", body.Position)
	}
	var ids []string
	for _, d := range body.Disruptions {
		ids = append(ids, d.ID)
	}
	// The route runs Lelystad, Amsterdam Zuid, Schiphol, Rotterdam: the HSL disruptions and the
	// network-wide calamity apply; Utrecht-Leiden and Haarlem-Alkmaar works do not.
	for _, want := range []string{"6068888", "6068889", "6b158242-eebf-499c-96c7-9d8956f3caa5"} {
		if !slices.Contains(ids, want) {
			t.Errorf("missing disruption %s in %v", want, ids)
		}
	}
	for _, not := range []string{"7006842", "7006847", "7008601"} {
		if slices.Contains(ids, not) {
			t.Errorf("disruption %s is not on the route: %v", not, ids)
		}
	}
}

func TestTrainUsesCache(t *testing.T) {
	fake := newFakeNS(t)
	srv := newTestServer(t, &Server{NS: fake})
	for range 3 {
		if res := get(t, srv.URL+"/api/trains/2450"); res.StatusCode != http.StatusOK {
			t.Fatalf("status %d", res.StatusCode)
		}
	}
	for _, name := range []string{"journey", "composition", "crowding", "disruptions"} {
		if n := fake.count(name); n != 1 {
			t.Errorf("%s: %d NS calls for 3 requests, want 1", name, n)
		}
	}
}

func TestTrainMissingExtras(t *testing.T) {
	fake := newFakeNS(t)
	fake.setFail("composition", true)
	srv := newTestServer(t, &Server{NS: fake})
	res := get(t, srv.URL+"/api/trains/2450")
	body := decode[trainResponse](t, res)
	if res.StatusCode != http.StatusOK || body.Composition != nil || !body.Stale || len(body.Stops) != 15 {
		t.Errorf("composition outage: want the trip without composition, marked stale: %d %+v", res.StatusCode, body)
	}
}

func TestTrainErrors(t *testing.T) {
	fake := newFakeNS(t)
	srv := newTestServer(t, &Server{NS: fake})
	for path, want := range map[string]int{
		"/api/trains/9999":    http.StatusNotFound,
		"/api/trains/abc":     http.StatusBadRequest,
		"/api/trains/1234567": http.StatusBadRequest,
		"/api/trains/12%2F3":  http.StatusBadRequest,
	} {
		if res := get(t, srv.URL+path); res.StatusCode != want {
			t.Errorf("%s: status %d, want %d", path, res.StatusCode, want)
		}
	}
	if fake.count("journey") != 1 {
		t.Errorf("invalid numbers must not reach NS: %d journey calls", fake.count("journey"))
	}

	fake.setFail("journey", true)
	res := get(t, srv.URL+"/api/trains/3522")
	if res.StatusCode != http.StatusServiceUnavailable || decode[map[string]string](t, res)["error"] != "unavailable" {
		t.Errorf("NS outage without a cached copy: status %d, want 503", res.StatusCode)
	}

	off := newTestServer(t, &Server{})
	if res := get(t, off.URL+"/api/trains/2450"); res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("no NS key: status %d, want 503", res.StatusCode)
	}
}

func TestWithCrowdingFillsGapsWithoutChangingCache(t *testing.T) {
	stops := []ns.Stop{
		{UIC: "1", Status: "STOP", Crowd: "UNKNOWN"},
		{UIC: "2", Status: "STOP", Crowd: "LOW"},
		{UIC: "3", Status: "PASSING"},
	}
	got := withCrowding(stops, ns.Crowding{"1": "HIGH", "2": "HIGH", "3": "HIGH"})
	if got[0].Crowd != "HIGH" || got[1].Crowd != "LOW" || got[2].Crowd != "" {
		t.Errorf("unexpected: %+v", got)
	}
	if stops[0].Crowd != "UNKNOWN" {
		t.Error("the cached journey was modified")
	}
}

func TestDepartures(t *testing.T) {
	fake := newFakeNS(t)
	srv := newTestServer(t, &Server{NS: fake})
	res := get(t, srv.URL+"/api/stations/ut/departures")
	body := decode[departuresResponse](t, res)
	if res.StatusCode != http.StatusOK || body.Station != "UT" || len(body.Departures) != 40 || body.Stale {
		t.Fatalf("status %d, %+v", res.StatusCode, body)
	}
	get(t, srv.URL+"/api/stations/UT/departures")
	if fake.count("departures") != 1 {
		t.Errorf("want ut and UT to share one cache entry: %d calls", fake.count("departures"))
	}
	for path, want := range map[string]int{
		"/api/stations/XYZ/departures":       http.StatusNotFound,
		"/api/stations/U1/departures":        http.StatusBadRequest,
		"/api/stations/ABCDEFGHI/departures": http.StatusBadRequest,
	} {
		if res := get(t, srv.URL+path); res.StatusCode != want {
			t.Errorf("%s: status %d, want %d", path, res.StatusCode, want)
		}
	}
}

var _ interface{ Snapshot() live.Snapshot } = fakeFeed{}
