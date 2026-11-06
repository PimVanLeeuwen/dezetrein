package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// fakeNS answers from the recordings in testdata/ns (train 2450, station UT). Any other train
// or station is unknown to it. Setting fail makes a method return an NS outage error.
type fakeNS struct {
	journey     *ns.Journey
	composition *ns.Composition
	crowding    ns.Crowding
	disruptions []ns.Disruption
	departures  []ns.Departure
	stations    []ns.Station

	mu    sync.Mutex
	calls map[string]int
	fail  map[string]bool
}

var errOutage = fmt.Errorf("ns: GET /x: HTTP 503")

func newFakeNS(t *testing.T) *fakeNS {
	t.Helper()
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ns", name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	f := &fakeNS{calls: map[string]int{}, fail: map[string]bool{}}
	var err error
	if f.journey, err = ns.ParseJourney(read("journey.json")); err != nil {
		t.Fatal(err)
	}
	if f.composition, err = ns.ParseComposition(read("composition.json")); err != nil {
		t.Fatal(err)
	}
	if f.crowding, err = ns.ParseCrowding(read("crowding.json")); err != nil {
		t.Fatal(err)
	}
	if f.disruptions, err = ns.ParseDisruptions(read("disruptions.json")); err != nil {
		t.Fatal(err)
	}
	if f.departures, err = ns.ParseDepartures(read("departures.json")); err != nil {
		t.Fatal(err)
	}
	if f.stations, err = ns.ParseStations(read("stations.json")); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *fakeNS) call(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[name]++
	if f.fail[name] {
		return errOutage
	}
	return nil
}

func (f *fakeNS) count(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[name]
}

func (f *fakeNS) setFail(name string, v bool) {
	f.mu.Lock()
	f.fail[name] = v
	f.mu.Unlock()
}

func (f *fakeNS) Journey(_ context.Context, train string) (*ns.Journey, error) {
	if err := f.call("journey"); err != nil {
		return nil, err
	}
	if train != "2450" {
		return nil, ns.ErrNotFound
	}
	return f.journey, nil
}

func (f *fakeNS) Composition(_ context.Context, train string) (*ns.Composition, error) {
	if err := f.call("composition"); err != nil {
		return nil, err
	}
	if train != "2450" {
		return nil, ns.ErrNotFound
	}
	return f.composition, nil
}

func (f *fakeNS) Crowding(_ context.Context, train string) (ns.Crowding, error) {
	if err := f.call("crowding"); err != nil {
		return nil, err
	}
	if train != "2450" {
		return nil, ns.ErrNotFound
	}
	return f.crowding, nil
}

func (f *fakeNS) Disruptions(context.Context) ([]ns.Disruption, error) {
	if err := f.call("disruptions"); err != nil {
		return nil, err
	}
	return f.disruptions, nil
}

func (f *fakeNS) Departures(_ context.Context, station string) ([]ns.Departure, error) {
	if err := f.call("departures"); err != nil {
		return nil, err
	}
	if station != "UT" {
		return nil, ns.ErrNotFound
	}
	return f.departures, nil
}

func (f *fakeNS) Stations(context.Context) ([]ns.Station, error) {
	if err := f.call("stations"); err != nil {
		return nil, err
	}
	return f.stations, nil
}

// testNow is the fixed time of the tests, an afternoon on the recording day.
var testNow = time.Date(2026, 11, 4, 15, 30, 0, 0, time.UTC)

// newTestServer serves s with a fixed clock.
func newTestServer(t *testing.T, s *Server) *httptest.Server {
	t.Helper()
	s.now = func() time.Time { return testNow }
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return srv
}

func get(t *testing.T, url string) *http.Response {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}
