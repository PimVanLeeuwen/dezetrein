package match

import (
	"testing"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// Utrecht Centraal and points north of it along the line to Amersfoort (about 1.1 km per 0.01°).
var vehicles = []ns.Vehicle{
	{RitID: "3522", Lat: 52.0900, Lng: 5.1100},
	{RitID: "7423", Lat: 52.0950, Lng: 5.1100},
	{RitID: "800", Lat: 52.1500, Lng: 5.1100},
	{RitID: "1", Lat: 52.3700, Lng: 4.9000}, // Amsterdam: far away
}

func TestMatchRanksNearestFirst(t *testing.T) {
	r := Match([]Fix{{Lat: 52.0901, Lng: 5.1100, AccuracyM: 20, UnixMS: 1}}, vehicles)
	if len(r.Candidates) != 2 || r.Candidates[0].Vehicle.RitID != "3522" || r.Candidates[1].Vehicle.RitID != "7423" {
		t.Fatalf("unexpected candidates: %+v", r.Candidates)
	}
	if r.Confidence != Low {
		t.Errorf("two trains 550 m apart: want low confidence, got %s", r.Confidence)
	}
	if d := r.Candidates[0].DistanceM; d < 5 || d > 20 {
		t.Errorf("distance %.1f m, want about 11", d)
	}
}

func TestMatchHighWhenAlone(t *testing.T) {
	r := Match([]Fix{{Lat: 52.1500, Lng: 5.1100, AccuracyM: 50}}, vehicles)
	if r.Confidence != High || len(r.Candidates) != 1 || r.Candidates[0].Vehicle.RitID != "800" {
		t.Fatalf("unexpected: %+v", r)
	}
}

func TestMatchHighWhenRunnerUpFarAway(t *testing.T) {
	vs := []ns.Vehicle{{RitID: "a", Lat: 52.09, Lng: 5.11}, {RitID: "b", Lat: 52.11, Lng: 5.11}}
	if r := Match([]Fix{{Lat: 52.09, Lng: 5.11, AccuracyM: 10}}, vs); r.Confidence != High {
		t.Errorf("runner-up 2.2 km farther: want high, got %s", r.Confidence)
	}
	// The same gap with a coarse fix (±2.5 km) is not convincing.
	if r := Match([]Fix{{Lat: 52.09, Lng: 5.11, AccuracyM: 2500}}, vs); r.Confidence != Low {
		t.Errorf("coarse fix: want low, got %s", r.Confidence)
	}
}

func TestMatchUsesLatestFix(t *testing.T) {
	fixes := []Fix{
		{Lat: 52.1500, Lng: 5.1100, AccuracyM: 20, UnixMS: 2000},
		{Lat: 52.0900, Lng: 5.1100, AccuracyM: 20, UnixMS: 1000},
	}
	if r := Match(fixes, vehicles); len(r.Candidates) == 0 || r.Candidates[0].Vehicle.RitID != "800" {
		t.Errorf("want the train at the newest fix first: %+v", r.Candidates)
	}
}

func TestMatchRadius(t *testing.T) {
	far := []ns.Vehicle{{RitID: "x", Lat: 52.1350, Lng: 5.1100}} // 5 km north
	if r := Match([]Fix{{Lat: 52.09, Lng: 5.11, AccuracyM: 10}}, far); r.Confidence != None || len(r.Candidates) != 0 {
		t.Errorf("precise fix: radius is 3 km, got %+v", r)
	}
	if r := Match([]Fix{{Lat: 52.09, Lng: 5.11, AccuracyM: 3000}}, far); len(r.Candidates) != 1 {
		t.Errorf("coarse fix (±3 km): radius widens to 6 km, got %+v", r)
	}
	huge := []ns.Vehicle{{RitID: "y", Lat: 52.2100, Lng: 5.1100}} // 13 km north
	if r := Match([]Fix{{Lat: 52.09, Lng: 5.11, AccuracyM: 9000}}, huge); len(r.Candidates) != 0 {
		t.Errorf("radius is capped at 10 km, got %+v", r)
	}
}

func TestMatchCapsCandidates(t *testing.T) {
	var vs []ns.Vehicle
	for i := range 6 {
		vs = append(vs, ns.Vehicle{RitID: string(rune('a' + i)), Lat: 52.09 + float64(i)*0.001, Lng: 5.11})
	}
	if r := Match([]Fix{{Lat: 52.09, Lng: 5.11, AccuracyM: 10}}, vs); len(r.Candidates) != MaxCandidates || r.Candidates[0].Vehicle.RitID != "a" {
		t.Errorf("unexpected: %+v", r.Candidates)
	}
}

func TestMatchEmpty(t *testing.T) {
	if r := Match(nil, vehicles); r.Confidence != None || r.Candidates != nil {
		t.Errorf("no fixes: %+v", r)
	}
	if r := Match([]Fix{{Lat: 52.09, Lng: 5.11, AccuracyM: 10}}, nil); r.Confidence != None {
		t.Errorf("no vehicles: %+v", r)
	}
}
