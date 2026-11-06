package ns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixtureServer serves the recordings in testdata/ns by path, as listed in index.json, and
// fails the test on a request it has no recording for or one without the key.
func fixtureServer(t *testing.T) *Client {
	t.Helper()
	dir := filepath.Join("..", "..", "testdata", "ns")
	raw, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var idx struct {
		Entries []struct {
			Path, Query, File string
		}
	}
	if err := json.Unmarshal(raw, &idx); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, e := range idx.Entries {
		files[e.Path] = filepath.Join(dir, e.File)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Ocp-Apim-Subscription-Key") != "test-key" {
			t.Errorf("%s: missing key", r.URL.Path)
		}
		f, ok := files[r.URL.Path]
		if !ok {
			t.Errorf("no recording for %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, f)
	}))
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL, Key: "test-key", HTTP: srv.Client()}
}

func TestJourneyFromRecording(t *testing.T) {
	j, err := fixtureServer(t).Journey(context.Background(), "2450")
	if err != nil {
		t.Fatal(err)
	}
	if j.Train != "2450" || j.Category != "ICD" || j.Destination != "Rotterdam Centraal" || len(j.Stops) != 15 {
		t.Fatalf("unexpected journey header: %+v (%d stops)", j, len(j.Stops))
	}
	first, last := j.Stops[0], j.Stops[len(j.Stops)-1]
	if first.Code != "LLS" || first.Status != "ORIGIN" || first.Arrival != nil || first.Departure == nil {
		t.Errorf("unexpected origin: %+v", first)
	}
	want := time.Date(2026, 11, 4, 14, 53, 0, 0, time.FixedZone("", 3600))
	if d := first.Departure; !d.Planned.Equal(want) || d.DelaySeconds != 815 || d.ActualTrack != "1" || d.Cancelled {
		t.Errorf("unexpected origin departure: %+v", d)
	}
	if first.Crowd != "LOW" || len(first.Carriages) != 2 || first.Carriages[0] != "3304" || first.Carriages[1] != "3108" {
		t.Errorf("unexpected crowd or carriages at origin: %q %v", first.Crowd, first.Carriages)
	}
	if last.Code != "RTD" || last.Arrival == nil || last.Departure != nil || last.Carriages != nil {
		t.Errorf("unexpected destination (the \"0\" placeholder stock must be dropped): %+v", last)
	}
	passing := j.Stops[1]
	if !passing.Passing() || passing.Arrival != nil || passing.Departure != nil {
		t.Errorf("unexpected passing stop: %+v", passing)
	}
	// Schiphol: planned track 5-6, actual 6.
	shl := j.Stops[12]
	if shl.Code != "SHL" || shl.Departure.PlannedTrack != "5-6" || shl.Departure.ActualTrack != "6" {
		t.Errorf("unexpected Schiphol stop: %+v %+v", shl, shl.Departure)
	}
}

func TestCrowdingFromRecording(t *testing.T) {
	c, err := fixtureServer(t).Crowding(context.Background(), "2450")
	if err != nil {
		t.Fatal(err)
	}
	if len(c) != 6 || c["8400394"] != "LOW" {
		t.Fatalf("unexpected crowding: %v", c)
	}
}

func TestDisruptionsFromRecording(t *testing.T) {
	d, err := fixtureServer(t).Disruptions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 8 {
		t.Fatalf("got %d disruptions, want 8", len(d))
	}
	hsl := d[0]
	if hsl.ID != "6068888" || hsl.Type != "DISRUPTION" || hsl.Impact != 3 || hsl.Start.IsZero() || hsl.End.IsZero() {
		t.Errorf("unexpected disruption: %+v", hsl)
	}
	if hsl.Description == "" || !contains(hsl.Stations, "RTD") || !contains(hsl.Stations, "BD") {
		t.Errorf("want a situation text and the affected stations: %+v", hsl)
	}
	types := map[string]int{}
	for _, x := range d {
		types[x.Type]++
		if x.Type == "CALAMITY" && (x.Description == "" || x.URL == "" || len(x.Stations) != 0) {
			t.Errorf("unexpected calamity: %+v", x)
		}
		if x.Type == "MAINTENANCE" && len(x.Stations) == 0 {
			t.Errorf("maintenance without stations: %+v", x)
		}
	}
	if types["DISRUPTION"] != 4 || types["MAINTENANCE"] != 3 || types["CALAMITY"] != 1 {
		t.Errorf("unexpected types: %v", types)
	}
}

func TestDeparturesFromRecording(t *testing.T) {
	d, err := fixtureServer(t).Departures(context.Background(), "UT")
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 40 {
		t.Fatalf("got %d departures, want 40", len(d))
	}
	first := d[0]
	if first.Train != "3957" || first.Category != "IC" || first.Direction != "Heerlen" || first.ActualTrack != "18" || first.Planned.IsZero() || first.Actual.IsZero() {
		t.Errorf("unexpected first departure: %+v", first)
	}
	withMessage := 0
	for _, x := range d {
		withMessage += min(len(x.Messages), 1)
	}
	if withMessage == 0 {
		t.Error("want the departure messages parsed")
	}
}

func TestParseJourneyLenient(t *testing.T) {
	j, err := ParseJourney([]byte(`{"stops":[{"id":"UT_0","stop":{"name":"Utrecht Centraal"},"status":"STOP",
		"departures":[{"plannedTime":"not a time","stockIdentifiers":["0","9402",""]}],
		"arrivals":[{"plannedTime":"2026-11-04T15:05:00+0100","product":{"number":3522,"categoryCode":"IC"},"destination":{"name":"Venlo"}}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	s := j.Stops[0]
	if j.Train != "3522" || j.Destination != "Venlo" || s.Code != "UT" || s.Arrival == nil || s.Departure != nil {
		t.Errorf("unexpected: %+v %+v", j, s)
	}
	if len(s.Carriages) != 1 || s.Carriages[0] != "9402" {
		t.Errorf("carriages %v, want [9402]", s.Carriages)
	}
}

func TestParsersRejectInvalidJSON(t *testing.T) {
	for name, parse := range map[string]func([]byte) error{
		"journey":     func(b []byte) error { _, err := ParseJourney(b); return err },
		"crowding":    func(b []byte) error { _, err := ParseCrowding(b); return err },
		"disruptions": func(b []byte) error { _, err := ParseDisruptions(b); return err },
		"departures":  func(b []byte) error { _, err := ParseDepartures(b); return err },
	} {
		if parse([]byte(`<html>`)) == nil {
			t.Errorf("%s: want an error for a non-JSON body", name)
		}
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func TestStationsFromRecording(t *testing.T) {
	s, err := fixtureServer(t).Stations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(s) < 700 {
		t.Fatalf("got %d stations", len(s))
	}
	var ut *Station
	nl := 0
	for i := range s {
		if s[i].Code == "UT" {
			ut = &s[i]
		}
		if s[i].Country == "NL" {
			nl++
		}
	}
	if ut == nil || ut.Name != "Utrecht Centraal" || ut.UIC != "8400621" || !ut.Departures || ut.Lat < 52 || ut.Lng < 5 {
		t.Errorf("unexpected Utrecht: %+v", ut)
	}
	if nl < 350 {
		t.Errorf("%d Dutch stations, want about 400", nl)
	}
}

func TestParseStationsBareAndInvalid(t *testing.T) {
	s, err := ParseStations([]byte(`[{"code":"X","namen":{"lang":""},"lat":52,"lng":5},{"code":"","lat":1,"lng":1},{"code":"Z","lat":0,"lng":0}]`))
	if err != nil || len(s) != 1 || s[0].Name != "X" {
		t.Fatalf("got %+v %v", s, err)
	}
	if _, err := ParseStations([]byte(`<html>`)); err == nil {
		t.Error("want an error")
	}
}
