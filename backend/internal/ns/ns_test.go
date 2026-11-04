package ns

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleVehicles = `{"payload":{"treinen":[
 {"treinNummer":3045,"ritId":"3045","lat":52.0894,"lng":5.1101,"snelheid":87.5,"richting":42,"horizontaleNauwkeurigheid":-1,"type":"IC","bron":"OBIS"},
 {"treinNummer":"7423","lat":52.1,"lng":5.2,"snelheid":0,"richting":180,"type":"SPR","bron":"KV6"},
 {"treinNummer":1,"lat":0,"lng":0,"type":"IC"},
 {"treinNummer":2,"lat":52.3,"type":"IC"},
 {"lat":52.3,"lng":4.9,"type":"IC"}
]}}`

func TestParseVehicles(t *testing.T) {
	got, err := ParseVehicles([]byte(sampleVehicles))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d vehicles, want 2 (zero, missing position and missing number dropped): %+v", len(got), got)
	}
	ic := got[0]
	if ic.Number() != "3045" || ic.Lat != 52.0894 || ic.Lng != 5.1101 || ic.SpeedKmh != 87.5 || ic.HeadingDeg != 42 || ic.Type != "IC" || ic.Source != "OBIS" {
		t.Errorf("unexpected first vehicle: %+v", ic)
	}
	if ic.AccuracyM != nil {
		t.Errorf("negative accuracy should be dropped, got %v", *ic.AccuracyM)
	}
	if got[1].Number() != "7423" {
		t.Errorf("string train number not parsed: %+v", got[1])
	}
}

func TestParseVehiclesAccuracy(t *testing.T) {
	got, err := ParseVehicles([]byte(`{"treinen":[{"ritId":"800","lat":51.4,"lng":5.4,"horizontaleNauwkeurigheid":194.4}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].AccuracyM == nil || *got[0].AccuracyM != 194.4 {
		t.Fatalf("unexpected: %+v", got)
	}
}

// TestParseRecordedVehicles runs the parser on a real recording from cmd/nsrecord.
func TestParseRecordedVehicles(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ns", "vehicles.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseVehicles(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 50 {
		t.Fatalf("parsed only %d vehicles from a real recording", len(got))
	}
	for _, v := range got {
		if v.Number() == "" || v.Lat < 49 || v.Lat > 55 || v.Lng < 2 || v.Lng > 11 {
			t.Errorf("implausible vehicle: %+v", v)
		}
	}
}

func TestParseVehiclesUnwrapped(t *testing.T) {
	got, err := ParseVehicles([]byte(`{"treinen":[{"ritId":"800","lat":51.4,"lng":5.4,"type":"SPR"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Number() != "800" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestParseVehiclesInvalid(t *testing.T) {
	if _, err := ParseVehicles([]byte(`<html>`)); err == nil {
		t.Fatal("want an error for a non-JSON body")
	}
}

func TestGetSendsKeyAndQuery(t *testing.T) {
	var gotKey, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Ocp-Apim-Subscription-Key")
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Key: "secret", HTTP: srv.Client()}
	if _, err := c.Get(context.Background(), "/x", url.Values{"train": {"3045"}}); err != nil {
		t.Fatal(err)
	}
	if gotKey != "secret" || gotQuery != "train=3045" {
		t.Errorf("key %q query %q", gotKey, gotQuery)
	}
}

func TestGetErrorDoesNotLeakKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Key: "secret-key-value", HTTP: srv.Client()}
	res, err := c.Get(context.Background(), "/x", nil)
	if err == nil || res == nil || res.Status != http.StatusUnauthorized {
		t.Fatalf("want a 401 error with the response, got %v %v", res, err)
	}
	if strings.Contains(err.Error(), "secret-key-value") {
		t.Errorf("error leaks the key: %v", err)
	}
}

func TestVehiclesFallsBackToSecondPath(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == VehiclePaths[0] {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(sampleVehicles))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Key: "k", HTTP: srv.Client()}
	v, path, err := c.Vehicles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if path != VehiclePaths[1] || len(v) != 2 || len(paths) != 2 {
		t.Errorf("path %q, %d vehicles, requests %v", path, len(v), paths)
	}
}

func TestVehiclesStopsOnOtherErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Key: "k", HTTP: srv.Client()}
	_, _, err := c.Vehicles(context.Background())
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("want a 429 error, got %v", err)
	}
}
