package match

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// The simulator models what the field test measured: NS refreshes a train's position every 15 s,
// each refresh is the true position of 6 s earlier, and between refreshes the position stays put
// while the accuracy field grows by speed times the position's age. Coordinates are synthetic.

var (
	simOrigin = [2]float64{52.0, 5.0}
	simStart  = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
)

const (
	simRefresh = 15 * time.Second
	simLag     = 6 * time.Second
	simPoll    = 10 * time.Second
)

type simTrain struct {
	id      string
	kmh     float64
	heading float64 // direction of travel
	offsetM float64 // sideways from the reference line (east is positive): a parallel track
	startM  float64 // position along its heading at simStart
	// frozenFrom, when set, stops NS refreshing this train from that moment (seen in the field
	// test: a position stuck for a minute while the train ran on).
	frozenFrom time.Duration
	// richting, when set, is a stale reported heading instead of the true one.
	richting *float64
}

// trueAt is where the train really is at t after simStart.
func (s simTrain) trueAt(t time.Duration) (float64, float64) {
	lat, lng := ahead(simOrigin[0], simOrigin[1], 90, s.offsetM)
	return ahead(lat, lng, s.heading, s.startM+s.kmh/3.6*t.Seconds())
}

// reported is the train as the NS feed shows it at t.
func (s simTrain) reported(t time.Duration) ns.Vehicle {
	refresh := t.Truncate(simRefresh)
	if s.frozenFrom > 0 && refresh > s.frozenFrom {
		refresh = s.frozenFrom.Truncate(simRefresh)
	}
	lat, lng := s.trueAt(refresh - simLag)
	acc := baseAccuracyM + s.kmh/3.6*(t-refresh).Seconds()
	h := s.heading
	if s.richting != nil {
		h = *s.richting
	}
	return ns.Vehicle{RitID: s.id, Lat: lat, Lng: lng, SpeedKmh: s.kmh, HeadingDeg: h, AccuracyM: &acc}
}

// simFrames are the poller's snapshots from 5 minutes before until `until`.
func simFrames(trains []simTrain, until time.Duration) []Frame {
	var out []Frame
	for t := until - 5*time.Minute; t <= until; t += simPoll {
		f := Frame{At: simStart.Add(t)}
		for _, s := range trains {
			f.Vehicles = append(f.Vehicles, s.reported(t))
		}
		out = append(out, f)
	}
	return out
}

// phone describes the user's location readings.
type phone struct {
	accuracyM float64 // reported accuracy
	noiseM    float64 // actual error (standard deviation per axis)
	motion    bool    // reports speed and heading
	seed      uint64
}

// simFixes are the readings a phone on train `on` takes every 3 s from `from` for `span`, as
// the page would post them (the five most recent).
func simFixes(on simTrain, p phone, from, span time.Duration) []Fix {
	rng := rand.New(rand.NewPCG(p.seed, 1))
	var out []Fix
	for t := from; t <= from+span; t += 3 * time.Second {
		lat, lng := on.trueAt(t)
		lat, lng = ahead(lat, lng, 0, rng.NormFloat64()*p.noiseM)
		lat, lng = ahead(lat, lng, 90, rng.NormFloat64()*p.noiseM)
		f := Fix{Lat: lat, Lng: lng, AccuracyM: p.accuracyM, UnixMS: simStart.Add(t).UnixMilli()}
		if p.motion {
			speed, heading := on.kmh/3.6, on.heading
			f.SpeedMS, f.Heading = &speed, &heading
		}
		out = append(out, f)
	}
	if len(out) > 5 {
		out = out[len(out)-5:]
	}
	return out
}

func ids(r Result) []string {
	var out []string
	for _, c := range r.Candidates {
		out = append(out, c.Vehicle.Number())
	}
	return out
}

func rank(r Result, id string) int {
	for i, c := range r.Candidates {
		if c.Vehicle.Number() == id {
			return i + 1
		}
	}
	return math.MaxInt
}

var (
	precise = phone{accuracyM: 10, noiseM: 8, motion: true}
	coarse  = phone{accuracyM: 1414, noiseM: 400}
)
