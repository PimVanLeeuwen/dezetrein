package match

import (
	"math"
	"testing"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// Scenarios from the field test, on synthetic tracks. Each checks the result after the first
// 15 s of readings, the moment the page shows its question.

func TestScenarioOppositeTrainOnParallelTrack(t *testing.T) {
	ic := simTrain{id: "ic", kmh: 130, heading: 0}
	// Passes the user, on the track 15 m east, in the middle of the 15 s.
	other := simTrain{id: "other", kmh: 130, heading: 180, offsetM: 15, startM: -2 * 130 / 3.6 * 40}
	trains := []simTrain{ic, other}
	now := 40 * time.Second
	for name, p := range map[string]phone{"precise": precise, "coarse": coarse} {
		for seed := range uint64(20) {
			p.seed = seed
			r := Match(simFixes(ic, p, now-15*time.Second, 15*time.Second), simFrames(trains, now))
			if name == "precise" && (rank(r, "ic") != 1 || r.Confidence != High) {
				t.Fatalf("%s seed %d: want the IC first with high confidence, got %v %s", name, seed, ids(r), r.Confidence)
			}
			if rank(r, "ic") > 2 {
				t.Fatalf("%s seed %d: IC not offered: %v", name, seed, ids(r))
			}
		}
	}
}

func TestScenarioSprinterAlongside(t *testing.T) {
	// Approaching a big station: an intercity and a sprinter on parallel tracks, same direction.
	ic := simTrain{id: "ic", kmh: 120, heading: 30}
	spr := simTrain{id: "spr", kmh: 80, heading: 30, offsetM: 20, startM: 120}
	trains := []simTrain{ic, spr}
	now := 3 * time.Minute
	for seed := range uint64(20) {
		p := precise
		p.seed = seed
		r := Match(simFixes(ic, p, now-15*time.Second, 15*time.Second), simFrames(trains, now))
		if rank(r, "ic") != 1 {
			t.Fatalf("seed %d: want the IC first, got %v", seed, ids(r))
		}
		// Without speed from the phone the sprinter is still close: both must be offered.
		p.motion = false
		r = Match(simFixes(ic, p, now-15*time.Second, 15*time.Second), simFrames(trains, now))
		if rank(r, "ic") > 2 || rank(r, "spr") > 2 {
			t.Fatalf("seed %d without motion: want both offered, got %v", seed, ids(r))
		}
	}
}

func TestScenarioFrozenPosition(t *testing.T) {
	// NS stopped refreshing the train a minute ago, so its reported position is 2 km behind the
	// user. A standing train waits at a siding 1 km ahead of that position, nearer than the stale
	// position: only dead reckoning puts the moving train back next to the user.
	ic := simTrain{id: "ic", kmh: 130, heading: 0, frozenFrom: 2 * time.Minute}
	now := 3 * time.Minute
	frozen := ic.reported(now)
	stand := simTrain{id: "stand", kmh: 0, heading: 0, offsetM: 40, startM: 0}
	frames := simFrames([]simTrain{ic}, now)
	for i := range frames {
		v := stand.reported(0)
		v.Lat, v.Lng = ahead(frozen.Lat, frozen.Lng, 0, 1000)
		v.Lat, v.Lng = ahead(v.Lat, v.Lng, 90, 40)
		frames[i].Vehicles = append(frames[i].Vehicles, v)
	}
	p := precise
	p.motion = false
	r := Match(simFixes(ic, p, now-15*time.Second, 15*time.Second), frames)
	if rank(r, "ic") != 1 {
		t.Fatalf("want the frozen IC first, got %v (distances %+v)", ids(r), r.Candidates)
	}
}

func TestScenarioStaleReportedHeading(t *testing.T) {
	// richting stuck at 262 deg while the train runs north: the heading comes from movement.
	stale := 262.0
	ic := simTrain{id: "ic", kmh: 130, heading: 0, richting: &stale}
	now := 3 * time.Minute
	p := precise
	p.motion = false
	r := Match(simFixes(ic, p, now-15*time.Second, 15*time.Second), simFrames([]simTrain{ic}, now))
	if len(r.Candidates) != 1 || r.Candidates[0].DistanceM > 150 {
		t.Fatalf("want the IC within 150 m after dead reckoning, got %+v", r.Candidates)
	}
}

func TestScenarioBusyStation(t *testing.T) {
	// Four trains standing side by side; the user is in the first. No clear winner: ask.
	var trains []simTrain
	for i, id := range []string{"a", "b", "c", "d"} {
		trains = append(trains, simTrain{id: id, offsetM: float64(i) * 25})
	}
	now := 3 * time.Minute
	p := phone{accuracyM: 30, noiseM: 15, seed: 1}
	r := Match(simFixes(trains[0], p, now-15*time.Second, 15*time.Second), simFrames(trains, now))
	if r.Confidence != Low || len(r.Candidates) != MaxCandidates || rank(r, "a") > 2 {
		t.Fatalf("want low confidence with the user's train offered: %v %s", ids(r), r.Confidence)
	}
}

func TestScenarioAloneIsHigh(t *testing.T) {
	ic := simTrain{id: "ic", kmh: 100, heading: 45}
	now := 3 * time.Minute
	r := Match(simFixes(ic, coarse, now-15*time.Second, 15*time.Second), simFrames([]simTrain{ic}, now))
	if r.Confidence != High || rank(r, "ic") != 1 {
		t.Fatalf("one train in range: %v %s", ids(r), r.Confidence)
	}
}

func TestPhoneClockAhead(t *testing.T) {
	ic := simTrain{id: "ic", kmh: 100, heading: 45}
	now := 3 * time.Minute
	fixes := simFixes(ic, precise, now-15*time.Second, 15*time.Second)
	for i := range fixes {
		fixes[i].UnixMS += (45 * time.Second).Milliseconds()
	}
	if r := Match(fixes, simFrames([]simTrain{ic}, now)); rank(r, "ic") != 1 {
		t.Errorf("fixes 45 s ahead of every snapshot: want them anchored, got %v", ids(r))
	}
}

func TestOldFixesOutsideWindowIgnored(t *testing.T) {
	a := simTrain{id: "a", kmh: 100, heading: 0}
	b := simTrain{id: "b", kmh: 100, heading: 0, offsetM: 8000}
	now := 5 * time.Minute
	fixes := simFixes(a, precise, now-15*time.Second, 15*time.Second)
	// A reading from 4 minutes earlier, far from train a: it must not knock a out.
	old := fixes[0]
	old.Lat, old.Lng = b.trueAt(now - 4*time.Minute)
	old.UnixMS = simStart.Add(now - 4*time.Minute).UnixMilli()
	if r := Match(append([]Fix{old}, fixes...), simFrames([]simTrain{a, b}, now)); rank(r, "a") != 1 {
		t.Errorf("want train a, got %v", ids(r))
	}
}

func TestTrainMustBeNearEveryFix(t *testing.T) {
	ic := simTrain{id: "ic", kmh: 130, heading: 0}
	now := 3 * time.Minute
	fixes := simFixes(ic, precise, now-15*time.Second, 15*time.Second)
	frames := simFrames([]simTrain{ic}, now)
	// A train that appears in the feed only in the newest snapshot, next to the user.
	lat, lng := ic.trueAt(now)
	frames[len(frames)-1].Vehicles = append(frames[len(frames)-1].Vehicles, ns.Vehicle{RitID: "new", Lat: lat, Lng: lng})
	if r := Match(fixes, frames); rank(r, "new") <= MaxCandidates || rank(r, "ic") != 1 {
		t.Errorf("a train missing at earlier fixes must not be offered: %v", ids(r))
	}
}

func TestTrainGoneFromFeedNotOffered(t *testing.T) {
	ic := simTrain{id: "ic", kmh: 100, heading: 0}
	now := 3 * time.Minute
	frames := simFrames([]simTrain{ic}, now)
	fixes := simFixes(ic, precise, now-30*time.Second, 10*time.Second) // matched to older frames
	frames[len(frames)-1].Vehicles = nil
	if r := Match(fixes, frames); len(r.Candidates) != 0 {
		t.Errorf("a train no longer in the feed must not be offered: %v", ids(r))
	}
}

func TestMatchEmpty(t *testing.T) {
	frames := simFrames([]simTrain{{id: "a"}}, time.Minute)
	if r := Match(nil, frames); r.Confidence != None || r.Candidates != nil {
		t.Errorf("no fixes: %+v", r)
	}
	if r := Match(simFixes(simTrain{id: "a"}, precise, 0, 0), nil); r.Confidence != None {
		t.Errorf("no frames: %+v", r)
	}
	far := []Fix{{Lat: 53.0, Lng: 6.5, AccuracyM: 10, UnixMS: simStart.Add(time.Minute).UnixMilli()}}
	if r := Match(far, frames); r.Confidence != None || len(r.Candidates) != 0 {
		t.Errorf("no train in range: %+v", r)
	}
}

func TestReckon(t *testing.T) {
	acc := func(m float64) *float64 { return &m }
	for name, tc := range map[string]struct {
		v     ns.Vehicle
		extra time.Duration
		want  float64
	}{
		"fresh":                 {ns.Vehicle{SpeedKmh: 108, AccuracyM: acc(200)}, 0, 180},
		"grown 100 m":           {ns.Vehicle{SpeedKmh: 108, AccuracyM: acc(300)}, 0, 280},
		"slow: growth damped":   {ns.Vehicle{SpeedKmh: 18, AccuracyM: acc(505)}, 0, 305*18.0/80 + 30},
		"no accuracy":           {ns.Vehicle{SpeedKmh: 108}, 0, 180},
		"reading 4 s later":     {ns.Vehicle{SpeedKmh: 108, AccuracyM: acc(200)}, 4 * time.Second, 300},
		"reading 10 s earlier":  {ns.Vehicle{SpeedKmh: 108, AccuracyM: acc(200)}, -10 * time.Second, -120},
		"capped at two minutes": {ns.Vehicle{SpeedKmh: 108, AccuracyM: acc(9000)}, 0, 3600},
	} {
		if got := reckonM(tc.v, tc.extra); math.Abs(got-tc.want) > 0.01 {
			t.Errorf("%s: %v m, want %v", name, got, tc.want)
		}
	}
}

func TestPenalty(t *testing.T) {
	speed, north, south := 36.0, 0.0, 180.0 // 129.6 km/h
	v := ns.Vehicle{SpeedKmh: 130}
	if p := penalty(Fix{AccuracyM: 10, SpeedMS: &speed, Heading: &north}, v, 0); p != 0 {
		t.Errorf("matching speed and heading: penalty %v", p)
	}
	if p := penalty(Fix{AccuracyM: 10, SpeedMS: &speed, Heading: &south}, v, 0); p != maxPenaltyM {
		t.Errorf("opposite heading: penalty %v, want %v", p, maxPenaltyM)
	}
	if p := penalty(Fix{AccuracyM: 10, SpeedMS: &speed}, ns.Vehicle{SpeedKmh: 0}, 0); p != min((129.6-speedSlackKmh)*speedPenaltyM, maxPenaltyM) {
		t.Errorf("moving phone, standing train: penalty %v", p)
	}
	if p := penalty(Fix{AccuracyM: 1414, SpeedMS: &speed, Heading: &south}, v, 0); p != 0 {
		t.Errorf("coarse fix: motion must not count, penalty %v", p)
	}
}
