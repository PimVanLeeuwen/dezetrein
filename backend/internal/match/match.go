// Package match finds the train a user is on from their location fixes and the live vehicle
// list. It is a pure function: no I/O, no clock, no logging, so location never leaves it.
//
// This is the interim matcher: nearest train to the latest fix. Phase 3 replaces the scoring
// (dead reckoning of NS positions, a sliding window over the fixes, speed and heading when the
// phone reports them) behind the same signature.
package match

import (
	"sort"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/geo"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// Fix is one browser location reading. Speed and heading are optional: phones inside a train
// often report neither.
type Fix struct {
	Lat, Lng  float64
	AccuracyM float64
	SpeedMS   *float64 // metres per second, as the Geolocation API gives it
	Heading   *float64 // degrees clockwise from north
	UnixMS    int64
}

// Confidence says how sure the match is. The page always confirms with the user; High only
// changes the wording ("Are you on the train to Venlo?" versus "Which train are you on?").
type Confidence string

const (
	None Confidence = "none"
	Low  Confidence = "low"
	High Confidence = "high"
)

// Candidate is a train near the user, nearest first.
type Candidate struct {
	Vehicle   ns.Vehicle
	DistanceM float64
}

// Result is the outcome of one match.
type Result struct {
	Confidence Confidence
	Candidates []Candidate // at most MaxCandidates
}

// MaxCandidates is how many trains a result offers to choose from.
const MaxCandidates = 3

const (
	minRadiusM = 3000
	maxRadiusM = 10000
	// minGapM is how much farther the runner-up must be for a High match.
	minGapM = 1000
)

// Match ranks the trains near the latest fix. It returns no candidates for an empty input.
func Match(fixes []Fix, vehicles []ns.Vehicle) Result {
	if len(fixes) == 0 {
		return Result{Confidence: None}
	}
	f := fixes[0]
	for _, x := range fixes[1:] {
		if x.UnixMS > f.UnixMS {
			f = x
		}
	}
	radius := min(max(2*f.AccuracyM, minRadiusM), maxRadiusM)
	var near []Candidate
	for _, v := range vehicles {
		if d := geo.DistanceM(f.Lat, f.Lng, v.Lat, v.Lng); d <= radius {
			near = append(near, Candidate{Vehicle: v, DistanceM: d})
		}
	}
	sort.SliceStable(near, func(i, j int) bool { return near[i].DistanceM < near[j].DistanceM })
	if len(near) > MaxCandidates {
		near = near[:MaxCandidates]
	}
	r := Result{Candidates: near, Confidence: None}
	switch {
	case len(near) == 1:
		r.Confidence = High
	case len(near) > 1 && near[1].DistanceM-near[0].DistanceM >= max(minGapM, f.AccuracyM):
		r.Confidence = High
	case len(near) > 1:
		r.Confidence = Low
	}
	return r
}
