package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

func fakeNS(t *testing.T) (*ns.Client, *[]string) {
	t.Helper()
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.RequestURI())
		switch {
		case r.URL.Path == ns.VehiclePaths[0]:
			w.Write([]byte(`{"payload":{"treinen":[
				{"treinNummer":7423,"lat":52.1,"lng":5.2,"snelheid":60,"type":"SPR","bron":"KV6"},
				{"treinNummer":3045,"lat":52.0,"lng":5.1,"snelheid":120,"type":"IC","bron":"OBIS"}]}}`))
		case strings.Contains(r.URL.Path, "prognose"):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("upstream broke"))
		default:
			w.Write([]byte(`{"payload":{"ok":true}}`))
		}
	}))
	t.Cleanup(srv.Close)
	return &ns.Client{BaseURL: srv.URL, Key: "k", HTTP: srv.Client()}, &seen
}

func TestRecordWritesFilesAndIndex(t *testing.T) {
	c, seen := fakeNS(t)
	dir := t.TempDir()
	idx, err := Record(context.Background(), c, dir, "", "UT", time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if idx.Train != "3045" {
		t.Errorf("picked train %q, want the moving OBIS intercity 3045", idx.Train)
	}
	for _, f := range []string{"vehicles.json", "journey.json", "composition.json", "crowding.txt", "disruptions.json", "departures.json", "index.json"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
	var onDisk Index
	b, _ := os.ReadFile(filepath.Join(dir, "index.json"))
	if err := json.Unmarshal(b, &onDisk); err != nil {
		t.Fatal(err)
	}
	var crowding Entry
	for _, e := range onDisk.Entries {
		if e.Name == "crowding" {
			crowding = e
		}
	}
	if crowding.Status != 500 || crowding.Error == "" {
		t.Errorf("failed call should be recorded with its status and error: %+v", crowding)
	}
	wantJourney := "/reisinformatie-api/api/v2/journey?omitCrowdForecast=false&train=3045"
	found := false
	for _, s := range *seen {
		found = found || s == wantJourney
	}
	if !found {
		t.Errorf("journey not requested as %s; requests: %v", wantJourney, *seen)
	}
}

func TestRecordUsesGivenTrain(t *testing.T) {
	c, seen := fakeNS(t)
	idx, err := Record(context.Background(), c, t.TempDir(), "1234", "ASD", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if idx.Train != "1234" {
		t.Errorf("train %q", idx.Train)
	}
	joined := strings.Join(*seen, " ")
	if !strings.Contains(joined, "/virtual-train-api/api/v1/trein/1234") || !strings.Contains(joined, "station=ASD") {
		t.Errorf("requests: %v", *seen)
	}
}

func TestPickTrain(t *testing.T) {
	vs := []ns.Vehicle{
		{TrainNumber: "1", Type: "IC", SpeedKmh: 0, Source: "OBIS"},
		{TrainNumber: "2", Type: "SPR", SpeedKmh: 80, Source: "OBIS"},
		{TrainNumber: "3", Type: "IC", SpeedKmh: 100, Source: "OBIS"},
	}
	if got := pickTrain(vs); got != "3" {
		t.Errorf("got %q, want 3", got)
	}
	if got := pickTrain(vs[:2]); got != "2" {
		t.Errorf("got %q, want the sprinter fallback 2", got)
	}
	if got := pickTrain(nil); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestRecordStationsKeepsOtherRecordings(t *testing.T) {
	c, seen := fakeNS(t)
	dir := t.TempDir()
	if _, err := Record(context.Background(), c, dir, "3045", "UT", time.Now()); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(dir, "journey.json"))
	*seen = nil
	for range 2 { // a second run replaces the stations entry rather than adding another
		if _, err := RecordStations(context.Background(), c, dir); err != nil {
			t.Fatal(err)
		}
	}
	if len(*seen) != 2 || (*seen)[0] != StationPaths[0] {
		t.Errorf("calls %v, want only the station list", *seen)
	}
	var idx Index
	b, _ := os.ReadFile(filepath.Join(dir, "index.json"))
	if err := json.Unmarshal(b, &idx); err != nil {
		t.Fatal(err)
	}
	names := map[string]int{}
	for _, e := range idx.Entries {
		names[e.Name]++
	}
	if names["stations"] != 1 || names["journey"] != 1 || idx.Train != "3045" {
		t.Errorf("index %+v", names)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "journey.json"))
	if string(before) != string(after) {
		t.Error("the journey recording changed")
	}
	if _, err := os.Stat(filepath.Join(dir, "stations.json")); err != nil {
		t.Error(err)
	}
}

func TestRecordStationsIntoEmptyFolder(t *testing.T) {
	c, _ := fakeNS(t)
	dir := filepath.Join(t.TempDir(), "new")
	idx, err := RecordStations(context.Background(), c, dir)
	if err != nil || len(idx.Entries) != 1 {
		t.Fatalf("%+v %v", idx, err)
	}
}
