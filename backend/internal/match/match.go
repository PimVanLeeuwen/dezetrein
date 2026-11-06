// Package match finds the train a user is on from their location fixes and recent snapshots of
// the live vehicle list. It is a pure function: no I/O, no clock, no logging, so location never
// leaves it.
//
// The method was validated in a field test on real trips (docs/matching.md):
//
//  1. Dead reckoning. An NS position is old when it arrives: the accuracy field grows with its
//     age, and the feed lags reality by about 6 s. Each train is pushed forward by its speed
//     times that age, along the heading of its own recent movement (the reported heading can be
//     stale).
//  2. A window of fixes. Each fix is compared with the snapshot taken closest to it, and trains
//     are ranked by their mean corrected distance over all fixes of the last few minutes, so a
//     train that only passes by, or runs on a parallel track the other way, falls behind.
//  3. Speed and heading from the phone count only when it reports them.
package match

import (
	"math"
	"sort"
	"time"

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

// Frame is one snapshot of the vehicle list and when it was taken.
type Frame struct {
	At       time.Time
	Vehicles []ns.Vehicle
}

// Confidence says how sure the match is. The page always confirms with the user; High only
// changes the wording ("Are you on the train to Venlo?" versus "Which train are you on?").
type Confidence string

const (
	None Confidence = "none"
	Low  Confidence = "low"
	High Confidence = "high"
)

// Candidate is a train near the user, best first. Vehicle is its latest position; DistanceM is
// its mean corrected distance from the fixes (plus any speed or heading penalty).
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

// Tuning, from the field test unless noted.
const (
	minRadiusM = 3000
	maxRadiusM = 10000
	// radiusMarginM widens the search around raw NS positions, which lag behind the train.
	radiusMarginM = 2000
	// window is how far back fixes count, measured from the newest fix.
	window = 3 * time.Minute
	// maxFrameGap is how far a fix may be from the nearest snapshot in time.
	maxFrameGap = 20 * time.Second

	// Dead reckoning: the accuracy field of a fresh position, the feed's lag behind reality,
	// the speed below which a train counts as standing, the longest extrapolation (a frozen
	// position on a curve drifts off the track), the least movement that gives a heading, and
	// the speed below which the accuracy growth is scaled down (see reckonM).
	baseAccuracyM = 200
	feedLag       = 6 * time.Second
	standingKmh   = 5
	maxProjection = 2 * time.Minute
	minCourseM    = 30
	dampKmh       = 80

	// A High match needs the runner-up at least highRatio times as far as the best, measured
	// from at least highFloorM (two trains 20 and 60 m away are not clearly apart).
	highRatio  = 2.0
	highFloorM = 150

	// Phone speed and heading, when given: metres of penalty per km/h of speed difference
	// beyond speedSlackKmh, and per degree of heading difference beyond headingSlackDeg, each
	// capped at maxPenaltyM. Heading counts only when both phone and train are moving.
	speedSlackKmh    = 25
	speedPenaltyM    = 10
	headingSlackDeg  = 45
	headingPenaltyM  = 8
	maxPenaltyM      = 1000
	movingPhoneMS    = 5
	movingTrainKmh   = 20
	preciseForMotion = 100 // phone speed and heading only count from fixes this precise
)

// Match ranks the trains near the user. frames must be oldest first; the result is empty when
// there are no fixes or frames.
func Match(fixes []Fix, frames []Frame) Result {
	if len(fixes) == 0 || len(frames) == 0 {
		return Result{Confidence: None}
	}
	fixes = recent(fixes)
	h := newHistory(frames)
	pairs := align(fixes, frames, 0)
	if len(pairs) == 0 {
		// The phone clock is off: anchor the newest fix to the newest snapshot.
		pairs = align(fixes, frames, frames[len(frames)-1].At.UnixMilli()-fixes[len(fixes)-1].UnixMS)
	}
	if len(pairs) == 0 {
		return Result{Confidence: None}
	}

	// sum holds, per train, the total corrected distance; count how many fixes it was near.
	sum := map[string]float64{}
	count := map[string]int{}
	for _, p := range pairs {
		f := p.fix
		radius := min(max(2*f.AccuracyM, minRadiusM), maxRadiusM) + radiusMarginM
		for _, v := range frames[p.frame].Vehicles {
			if geo.DistanceM(f.Lat, f.Lng, v.Lat, v.Lng) > radius {
				continue
			}
			k := v.Number()
			lat, lng := h.project(p.frame, v, time.UnixMilli(f.UnixMS+p.offsetMS))
			sum[k] += geo.DistanceM(f.Lat, f.Lng, lat, lng) + penalty(f, v, h.course(p.frame, v))
			count[k]++
		}
	}
	// Only trains near every fix count: the user cannot have left their train.
	latest := h.latest()
	var cands []Candidate
	for k, n := range count {
		if n != len(pairs) {
			continue
		}
		v, ok := latest[k]
		if !ok {
			continue // gone from the feed (the journey ended)
		}
		cands = append(cands, Candidate{Vehicle: v, DistanceM: sum[k] / float64(n)})
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].DistanceM != cands[j].DistanceM {
			return cands[i].DistanceM < cands[j].DistanceM
		}
		return cands[i].Vehicle.Number() < cands[j].Vehicle.Number()
	})
	r := Result{Confidence: None}
	switch {
	case len(cands) == 1:
		r.Confidence = High
	case len(cands) > 1 && cands[1].DistanceM >= highRatio*max(cands[0].DistanceM, highFloorM):
		r.Confidence = High
	case len(cands) > 1:
		r.Confidence = Low
	}
	if len(cands) > MaxCandidates {
		cands = cands[:MaxCandidates]
	}
	r.Candidates = cands
	return r
}

// recent sorts the fixes by time and keeps those within window of the newest.
func recent(fixes []Fix) []Fix {
	s := append([]Fix(nil), fixes...)
	sort.SliceStable(s, func(i, j int) bool { return s[i].UnixMS < s[j].UnixMS })
	cut := s[len(s)-1].UnixMS - window.Milliseconds()
	i := 0
	for s[i].UnixMS < cut {
		i++
	}
	return s[i:]
}

type pair struct {
	fix      Fix
	frame    int
	offsetMS int64 // added to the fix time when the phone clock was anchored
}

// align pairs each fix (its time shifted by offsetMS) with the nearest frame, dropping fixes
// with no frame within maxFrameGap.
func align(fixes []Fix, frames []Frame, offsetMS int64) []pair {
	var out []pair
	for _, f := range fixes {
		t := f.UnixMS + offsetMS
		i := sort.Search(len(frames), func(i int) bool { return frames[i].At.UnixMilli() >= t })
		best, gap := -1, int64(math.MaxInt64)
		for _, j := range []int{i - 1, i} {
			if j < 0 || j >= len(frames) {
				continue
			}
			if d := abs(frames[j].At.UnixMilli() - t); d < gap {
				best, gap = j, d
			}
		}
		if best >= 0 && gap <= maxFrameGap.Milliseconds() {
			out = append(out, pair{fix: f, frame: best, offsetMS: offsetMS})
		}
	}
	return out
}

// penalty turns a speed or heading mismatch into extra metres, for fixes precise enough to
// carry them.
func penalty(f Fix, v ns.Vehicle, course float64) float64 {
	if f.AccuracyM > preciseForMotion || f.SpeedMS == nil {
		return 0
	}
	p := 0.0
	phoneKmh := *f.SpeedMS * 3.6
	if d := math.Abs(phoneKmh-v.SpeedKmh) - speedSlackKmh; d > 0 {
		p += min(d*speedPenaltyM, maxPenaltyM)
	}
	if f.Heading != nil && *f.SpeedMS >= movingPhoneMS && v.SpeedKmh >= movingTrainKmh {
		if d := geo.HeadingDiff(*f.Heading, course) - headingSlackDeg; d > 0 {
			p += min(d*headingPenaltyM, maxPenaltyM)
		}
	}
	return p
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
