package match

import (
	"math"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/geo"
	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// history indexes the frames by train, for looking back at a train's earlier positions.
type history struct {
	frames []Frame
	byID   []map[string]ns.Vehicle // built on first use per frame
}

func newHistory(frames []Frame) *history {
	return &history{frames: frames, byID: make([]map[string]ns.Vehicle, len(frames))}
}

func (h *history) at(i int) map[string]ns.Vehicle {
	if h.byID[i] == nil {
		m := make(map[string]ns.Vehicle, len(h.frames[i].Vehicles))
		for _, v := range h.frames[i].Vehicles {
			m[v.Number()] = v
		}
		h.byID[i] = m
	}
	return h.byID[i]
}

func (h *history) latest() map[string]ns.Vehicle { return h.at(len(h.frames) - 1) }

// course is the train's heading from its own movement: the bearing from the most recent earlier
// position at least minCourseM away. NS's reported heading is the fallback.
func (h *history) course(frame int, v ns.Vehicle) float64 {
	k := v.Number()
	for i := frame - 1; i >= 0; i-- {
		p, ok := h.at(i)[k]
		if !ok {
			break
		}
		if geo.DistanceM(p.Lat, p.Lng, v.Lat, v.Lng) >= minCourseM {
			return bearing(p.Lat, p.Lng, v.Lat, v.Lng)
		}
	}
	return v.HeadingDeg
}

// project is where the train is estimated to be at time at (the reading's time). NS positions
// are old when they arrive: the accuracy field grows by the distance the train covered since the
// position was taken (on top of a base of about 200 m), and the feed lags reality by about 6 s.
// So the position is pushed along the train's course by that distance, plus speed times the lag
// and the time between the snapshot and the reading (negative when the reading came first:
// snapshots are 10 s apart, and at 130 km/h a train covers 180 m in 5 s).
//
// Below dampKmh the growth is scaled down with speed: around stations the field grows faster
// than a braking or starting train moves (in the field-test replays top 1 rose from 78 to 84%
// with coarse GPS and from 82 to 89% with precise GPS).
func (h *history) project(frame int, v ns.Vehicle, at time.Time) (lat, lng float64) {
	if v.SpeedKmh < standingKmh {
		return v.Lat, v.Lng
	}
	return ahead(v.Lat, v.Lng, h.course(frame, v), reckonM(v, at.Sub(h.frames[frame].At)))
}

// reckonM is how far ahead of its reported position the train is, extra after the snapshot.
func reckonM(v ns.Vehicle, extra time.Duration) float64 {
	grown := 0.0
	if v.AccuracyM != nil {
		grown = max(0, *v.AccuracyM-baseAccuracyM) * min(1, v.SpeedKmh/dampKmh)
	}
	mps := v.SpeedKmh / 3.6
	d := grown + mps*(feedLag+extra).Seconds()
	limit := mps * maxProjection.Seconds()
	return min(max(d, -limit), limit)
}

// ahead moves a point distM metres along a compass heading (flat-earth step, fine for the few
// kilometres involved).
func ahead(lat, lng, headingDeg, distM float64) (float64, float64) {
	const mPerDeg = 111195.0
	h := headingDeg * math.Pi / 180
	return lat + distM*math.Cos(h)/mPerDeg, lng + distM*math.Sin(h)/(mPerDeg*math.Cos(lat*math.Pi/180))
}

// bearing is the compass heading from one point to another.
func bearing(lat1, lng1, lat2, lng2 float64) float64 {
	const mPerDeg = 111195.0
	dy := (lat2 - lat1) * mPerDeg
	dx := (lng2 - lng1) * mPerDeg * math.Cos(lat2*math.Pi/180)
	return math.Mod(math.Atan2(dx, dy)*180/math.Pi+360, 360)
}
