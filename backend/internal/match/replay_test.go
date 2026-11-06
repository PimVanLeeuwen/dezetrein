package match

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// TestReplayFieldTrips replays real trips recorded with the field-test app. The recordings are
// personal location data, so they live only in the gitignored backend/data folder, together with
// replay.json naming the trip segments; without them the test is skipped (as in CI).
//
// It simulates a user opening the page every 30 s: the page collects readings for 15 s (and, in
// a second pass, 60 s), posts the five spread over that time, and the matcher sees the poller's
// snapshots of the 5 minutes before, one per 10 s.
func TestReplayFieldTrips(t *testing.T) {
	dir := filepath.Join("..", "..", "data")
	cfgRaw, err := os.ReadFile(filepath.Join(dir, "replay.json"))
	if err != nil {
		t.Skip("no local field recordings (backend/data/replay.json); skipped")
	}
	var cfg struct {
		Fixes, Vehicles string
		Segments        []struct {
			Name, Session, Train string
			From, To             time.Time
		}
	}
	if err := json.Unmarshal(cfgRaw, &cfg); err != nil {
		t.Fatal(err)
	}
	frames := loadFrames(t, filepath.Join(dir, cfg.Vehicles))
	fixes := loadFixes(t, filepath.Join(dir, cfg.Fixes))

	for _, seg := range cfg.Segments {
		var trip []Fix
		for session, fs := range fixes {
			if strings.HasPrefix(session, seg.Session) {
				for _, f := range fs {
					if at := time.UnixMilli(f.UnixMS); !at.Before(seg.From) && at.Before(seg.To) {
						trip = append(trip, f)
					}
				}
			}
		}
		sort.Slice(trip, func(i, j int) bool { return trip[i].UnixMS < trip[j].UnixMS })
		if len(trip) == 0 {
			t.Errorf("%s: no fixes", seg.Name)
			continue
		}
		for _, collect := range []time.Duration{15 * time.Second, 60 * time.Second} {
			s := replay(trip, frames, seg.Train, collect)
			t.Logf("%-20s after %2.0f s: top1 %3.0f%%  top3 %3.0f%%  high and wrong %2.0f%%  no candidates %2.0f%%  (%d openings, %d in recording gaps)",
				seg.Name, collect.Seconds(), s.pct(s.top1), s.pct(s.top3), s.pct(s.wrongHigh), s.pct(s.none), s.n, s.gaps)
			// Phase 0 found top3 97 to 100% after 15 s and a confident wrong answer in 0 to 2%.
			// Segments with few openings are reported but not judged: one miss is 25%.
			if collect == 15*time.Second && s.n >= 10 && (s.pct(s.top3) < 95 || s.pct(s.top1) < 80 || s.pct(s.wrongHigh) > 3) {
				t.Errorf("%s: below the field-test baseline", seg.Name)
			}
		}
	}
}

// replayStats counts openings that had readings and NS snapshots; openings during a gap in the
// recording (no reading, or no snapshot in the last 30 s) are counted apart, as in the Phase 0
// analysis.
type replayStats struct{ n, top1, top3, wrongHigh, none, gaps int }

func (s replayStats) pct(k int) float64 { return 100 * float64(k) / float64(max(s.n, 1)) }

func replay(trip []Fix, all []Frame, train string, collect time.Duration) replayStats {
	var s replayStats
	first, last := trip[0].UnixMS, trip[len(trip)-1].UnixMS
	for open := first; open < last-(3*time.Minute).Milliseconds(); open += (30 * time.Second).Milliseconds() {
		post := open + collect.Milliseconds()
		var got []Fix
		for _, f := range trip {
			if f.UnixMS >= open && f.UnixMS <= post {
				got = append(got, f)
			}
		}
		frames := poll(all, time.UnixMilli(post))
		if len(got) == 0 || len(frames) == 0 || time.UnixMilli(post).Sub(frames[len(frames)-1].At) > 30*time.Second {
			s.gaps++
			continue
		}
		s.n++
		r := Match(spread(got, 5), frames)
		switch {
		case len(r.Candidates) == 0:
			s.none++
		case rank(r, train) == 1:
			s.top1++
		}
		if rank(r, train) <= 3 {
			s.top3++
		}
		if r.Confidence == High && rank(r, train) != 1 {
			s.wrongHigh++
		}
	}
	return s
}

// spread picks n readings evenly over the list, always including the newest.
func spread(fs []Fix, n int) []Fix {
	if len(fs) <= n {
		return fs
	}
	out := make([]Fix, 0, n)
	for i := range n {
		out = append(out, fs[(len(fs)-1)*i/(n-1)])
	}
	return out
}

// poll returns what the poller would hold at `at`: snapshots of the 5 minutes before, one per
// 10 s (the recording polled more often).
func poll(all []Frame, at time.Time) []Frame {
	var out []Frame
	for _, f := range all {
		if f.At.After(at) {
			break
		}
		if f.At.Before(at.Add(-5*time.Minute)) || (len(out) > 0 && f.At.Sub(out[len(out)-1].At) < 9500*time.Millisecond) {
			continue
		}
		out = append(out, f)
	}
	return out
}

func loadFrames(t *testing.T, path string) []Frame {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []Frame
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 8<<20)
	for sc.Scan() {
		var line struct {
			T        time.Time    `json:"t"`
			Vehicles []ns.Vehicle `json:"vehicles"`
		}
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			t.Fatal(err)
		}
		out = append(out, Frame{At: line.T, Vehicles: line.Vehicles})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out
}

func loadFixes(t *testing.T, path string) map[string][]Fix {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]Fix{}
	for i, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var r struct {
			Session  string
			T        int64
			Lat, Lng float64
			Acc      float64
			Speed    *float64
			Heading  *float64
		}
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatal(fmt.Errorf("fixes line %d: %w", i+1, err))
		}
		if r.Acc <= 0 {
			continue
		}
		out[r.Session] = append(out[r.Session], Fix{Lat: r.Lat, Lng: r.Lng, AccuracyM: r.Acc, SpeedMS: r.Speed, Heading: r.Heading, UnixMS: r.T})
	}
	return out
}
