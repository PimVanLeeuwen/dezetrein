package ns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// Departure is one upcoming departure at a station from the Reisinformatie API
// (api/v2/departures?station=). It carries no stock, so it serves the manual train picker.
type Departure struct {
	Train        string    `json:"train"`
	Category     string    `json:"category"`
	Direction    string    `json:"direction"`
	Planned      time.Time `json:"planned"`
	Actual       time.Time `json:"actual,omitzero"`
	PlannedTrack string    `json:"plannedTrack,omitempty"`
	ActualTrack  string    `json:"actualTrack,omitempty"`
	Cancelled    bool      `json:"cancelled"`
	Messages     []string  `json:"messages,omitempty"` // Dutch, for example "Stopt niet in Zaandam"
}

// ParseDepartures reads a departures response, wrapped in {"payload":...} or bare. Entries
// without a train number or planned time are dropped.
func ParseDepartures(body []byte) ([]Departure, error) {
	type rawList struct {
		Departures []struct {
			Direction       string `json:"direction"`
			PlannedDateTime nsTime `json:"plannedDateTime"`
			ActualDateTime  nsTime `json:"actualDateTime"`
			PlannedTrack    string `json:"plannedTrack"`
			ActualTrack     string `json:"actualTrack"`
			Cancelled       bool   `json:"cancelled"`
			Product         struct {
				Number       flexString `json:"number"`
				CategoryCode string     `json:"categoryCode"`
			} `json:"product"`
			Messages []struct {
				Message string `json:"message"`
			} `json:"messages"`
		} `json:"departures"`
	}
	var doc struct {
		Payload *rawList `json:"payload"`
		rawList
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("ns: parse departures: %w", err)
	}
	raw := doc.rawList
	if doc.Payload != nil {
		raw = *doc.Payload
	}
	out := make([]Departure, 0, len(raw.Departures))
	for _, r := range raw.Departures {
		if r.Product.Number == "" || r.PlannedDateTime.IsZero() {
			continue
		}
		d := Departure{
			Train:        string(r.Product.Number),
			Category:     r.Product.CategoryCode,
			Direction:    r.Direction,
			Planned:      r.PlannedDateTime.Time,
			Actual:       r.ActualDateTime.Time,
			PlannedTrack: r.PlannedTrack,
			ActualTrack:  r.ActualTrack,
			Cancelled:    r.Cancelled,
		}
		for _, m := range r.Messages {
			if m.Message != "" {
				d.Messages = append(d.Messages, m.Message)
			}
		}
		out = append(out, d)
	}
	return out, nil
}

// Departures fetches the upcoming departures at a station by its code, for example UT.
func (c *Client) Departures(ctx context.Context, station string) ([]Departure, error) {
	res, err := c.Get(ctx, "/reisinformatie-api/api/v2/departures", url.Values{"station": {station}})
	if err != nil {
		return nil, err
	}
	return ParseDepartures(res.Body)
}
