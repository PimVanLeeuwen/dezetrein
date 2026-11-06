package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/live"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

func TestHealthz(t *testing.T) {
	srv := httptest.NewServer((&Server{Version: "1.2.3"}).Handler())
	defer srv.Close()
	res, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var body struct {
		OK      bool   `json:"ok"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || !body.OK || body.Version != "1.2.3" {
		t.Errorf("status %d, body %+v", res.StatusCode, body)
	}
	if res.Header.Get("X-Content-Type-Options") != "nosniff" || res.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("missing security headers: %v", res.Header)
	}
}

func TestUnknownRoutes(t *testing.T) {
	srv := httptest.NewServer((&Server{}).Handler())
	defer srv.Close()
	for path, want := range map[string]int{"/api/nope": http.StatusNotFound, "/": http.StatusNotFound} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != want {
			t.Errorf("%s: status %d, want %d", path, res.StatusCode, want)
		}
	}
	res, err := http.Post(srv.URL+"/healthz", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST /healthz: status %d, want 405", res.StatusCode)
	}
}

type fakeFeed live.Snapshot

func (f fakeFeed) Snapshot() live.Snapshot { return live.Snapshot(f) }

func TestHealthzReportsVehicleFeed(t *testing.T) {
	asOf := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	for name, tc := range map[string]struct {
		feed fakeFeed
		want string
	}{
		"fresh":         {fakeFeed{Vehicles: make([]ns.Vehicle, 2), AsOf: asOf}, `{"as_of":"2026-11-06T12:00:00Z","count":2,"stale":false}`},
		"never fetched": {fakeFeed{Stale: true}, `{"count":0,"stale":true}`},
	} {
		srv := httptest.NewServer((&Server{Vehicles: tc.feed}).Handler())
		res, err := http.Get(srv.URL + "/healthz")
		if err != nil {
			t.Fatal(err)
		}
		var body struct {
			OK       bool            `json:"ok"`
			Vehicles json.RawMessage `json:"vehicles"`
		}
		err = json.NewDecoder(res.Body).Decode(&body)
		res.Body.Close()
		srv.Close()
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK || !body.OK || string(body.Vehicles) != tc.want {
			t.Errorf("%s: status %d, ok %v, vehicles %s, want %s", name, res.StatusCode, body.OK, body.Vehicles, tc.want)
		}
	}
}
