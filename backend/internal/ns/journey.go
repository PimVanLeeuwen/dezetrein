package ns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// timeLayout is how the Reisinformatie API writes times, for example 2026-11-04T14:53:00+0100.
const timeLayout = "2006-01-02T15:04:05-0700"

// nsTime parses an NS timestamp; an empty or unparsable value becomes the zero time, so one odd
// field never drops a whole journey.
type nsTime struct{ time.Time }

func (t *nsTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil || s == "" {
		return nil
	}
	if v, err := time.Parse(timeLayout, s); err == nil {
		t.Time = v
	} else if v, err := time.Parse(time.RFC3339, s); err == nil {
		t.Time = v
	}
	return nil
}

// Journey is one train's run from the Reisinformatie API (api/v2/journey?train=).
type Journey struct {
	Train        string `json:"train"`
	Category     string `json:"category"` // for example IC, SPR, ICD
	CategoryName string `json:"categoryName"`
	Destination  string `json:"destination"`
	Stops        []Stop `json:"stops"`
}

// Stop is a station on the route. Passing stations (the train does not stop) are kept, so the
// route can be drawn, but have no times.
type Stop struct {
	Code      string   `json:"code"` // station code, for example UT
	Name      string   `json:"name"`
	UIC       string   `json:"uic"`
	Lat       float64  `json:"lat"`
	Lng       float64  `json:"lng"`
	Status    string   `json:"status"` // ORIGIN, STOP, PASSING or DESTINATION
	Arrival   *Event   `json:"arrival,omitempty"`
	Departure *Event   `json:"departure,omitempty"`
	Crowd     string   `json:"crowd,omitempty"`     // LOW, MEDIUM, HIGH or UNKNOWN, on departure
	Carriages []string `json:"carriages,omitempty"` // trainset numbers leaving this stop
}

// Passing reports whether the train runs through this station without stopping.
func (s Stop) Passing() bool { return s.Status == "PASSING" }

// Event is an arrival or departure at a stop.
type Event struct {
	Planned      time.Time `json:"planned"`
	Actual       time.Time `json:"actual,omitzero"`
	DelaySeconds int       `json:"delaySeconds"`
	PlannedTrack string    `json:"plannedTrack,omitempty"`
	ActualTrack  string    `json:"actualTrack,omitempty"`
	Cancelled    bool      `json:"cancelled"`
}

type rawEvent struct {
	Product struct {
		Number           flexString `json:"number"`
		CategoryCode     string     `json:"categoryCode"`
		LongCategoryName string     `json:"longCategoryName"`
	} `json:"product"`
	Destination struct {
		Name string `json:"name"`
	} `json:"destination"`
	PlannedTime      nsTime   `json:"plannedTime"`
	ActualTime       nsTime   `json:"actualTime"`
	DelayInSeconds   int      `json:"delayInSeconds"`
	PlannedTrack     string   `json:"plannedTrack"`
	ActualTrack      string   `json:"actualTrack"`
	Cancelled        bool     `json:"cancelled"`
	CrowdForecast    string   `json:"crowdForecast"`
	StockIdentifiers []string `json:"stockIdentifiers"`
}

func (r *rawEvent) event() *Event {
	if r == nil || r.PlannedTime.IsZero() {
		return nil
	}
	return &Event{
		Planned:      r.PlannedTime.Time,
		Actual:       r.ActualTime.Time,
		DelaySeconds: r.DelayInSeconds,
		PlannedTrack: r.PlannedTrack,
		ActualTrack:  r.ActualTrack,
		Cancelled:    r.Cancelled,
	}
}

// ParseJourney reads a journey response, wrapped in {"payload":...} or bare.
func ParseJourney(body []byte) (*Journey, error) {
	type rawJourney struct {
		ProductNumbers []flexString `json:"productNumbers"`
		Stops          []struct {
			ID   string `json:"id"`
			Stop struct {
				Name    string  `json:"name"`
				Lat     float64 `json:"lat"`
				Lng     float64 `json:"lng"`
				UICCode string  `json:"uicCode"`
			} `json:"stop"`
			Destination string     `json:"destination"`
			Status      string     `json:"status"`
			Arrivals    []rawEvent `json:"arrivals"`
			Departures  []rawEvent `json:"departures"`
		} `json:"stops"`
	}
	var doc struct {
		Payload *rawJourney `json:"payload"`
		rawJourney
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("ns: parse journey: %w", err)
	}
	raw := doc.rawJourney
	if doc.Payload != nil {
		raw = *doc.Payload
	}
	j := &Journey{Stops: make([]Stop, 0, len(raw.Stops))}
	if len(raw.ProductNumbers) > 0 {
		j.Train = string(raw.ProductNumbers[0])
	}
	for _, rs := range raw.Stops {
		s := Stop{
			Code:   stationCode(rs.ID),
			Name:   rs.Stop.Name,
			UIC:    rs.Stop.UICCode,
			Lat:    rs.Stop.Lat,
			Lng:    rs.Stop.Lng,
			Status: rs.Status,
		}
		var arr, dep *rawEvent
		if len(rs.Arrivals) > 0 {
			arr = &rs.Arrivals[0]
		}
		if len(rs.Departures) > 0 {
			dep = &rs.Departures[0]
		}
		s.Arrival, s.Departure = arr.event(), dep.event()
		if dep != nil {
			s.Crowd = dep.CrowdForecast
			s.Carriages = carriages(dep.StockIdentifiers)
		}
		for _, e := range []*rawEvent{dep, arr} {
			if e == nil {
				continue
			}
			if j.Train == "" {
				j.Train = string(e.Product.Number)
			}
			if j.Category == "" {
				j.Category, j.CategoryName = e.Product.CategoryCode, e.Product.LongCategoryName
			}
			if j.Destination == "" {
				j.Destination = e.Destination.Name
			}
		}
		if j.Destination == "" {
			j.Destination = rs.Destination
		}
		j.Stops = append(j.Stops, s)
	}
	return j, nil
}

// stationCode turns a stop id such as "LLS_0" into the station code "LLS".
func stationCode(id string) string {
	if i := strings.LastIndexByte(id, '_'); i > 0 {
		return id[:i]
	}
	return id
}

// carriages drops the "0" placeholders NS puts where the stock is unknown (seen at the last
// stop), and returns nil when nothing real is left.
func carriages(ids []string) []string {
	var out []string
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" && strings.Trim(id, "0") != "" {
			out = append(out, id)
		}
	}
	return out
}

// Journey fetches the run of one train by its journey number (ritnummer), with the crowd
// forecast per stop.
func (c *Client) Journey(ctx context.Context, train string) (*Journey, error) {
	res, err := c.Get(ctx, "/reisinformatie-api/api/v2/journey", url.Values{"train": {train}, "omitCrowdForecast": {"false"}})
	if err != nil {
		return nil, err
	}
	j, err := ParseJourney(res.Body)
	if err != nil {
		return nil, err
	}
	if j.Train == "" {
		j.Train = train
	}
	return j, nil
}
