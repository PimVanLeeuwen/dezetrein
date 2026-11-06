package ns

import (
	"context"
	"encoding/json"
	"fmt"
)

// StationsPath is the station list of the Reisinformatie API: every station NS knows, in the
// Netherlands and abroad, with coordinates. It changes a few times a year.
const StationsPath = "/reisinformatie-api/api/v2/stations"

// Station is one station from the station list.
type Station struct {
	Code       string  `json:"code"` // for example UT
	Name       string  `json:"name"` // full name, for example Utrecht Centraal
	UIC        string  `json:"uic"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	Country    string  `json:"country"`    // NL, D, B, ...
	Departures bool    `json:"departures"` // NS publishes departure times for it
}

// ParseStations reads the station list, wrapped in {"payload":[...]} or bare. Entries without a
// code or position are dropped.
func ParseStations(body []byte) ([]Station, error) {
	type raw struct {
		Code               string     `json:"code"`
		UICCode            flexString `json:"UICCode"`
		Land               string     `json:"land"`
		Lat                float64    `json:"lat"`
		Lng                float64    `json:"lng"`
		HeeftVertrektijden bool       `json:"heeftVertrektijden"`
		Namen              struct {
			Lang string `json:"lang"`
		} `json:"namen"`
	}
	var doc struct {
		Payload []raw `json:"payload"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		var bare []raw
		if json.Unmarshal(body, &bare) != nil {
			return nil, fmt.Errorf("ns: parse stations: %w", err)
		}
		doc.Payload = bare
	}
	out := make([]Station, 0, len(doc.Payload))
	for _, r := range doc.Payload {
		if r.Code == "" || (r.Lat == 0 && r.Lng == 0) {
			continue
		}
		name := r.Namen.Lang
		if name == "" {
			name = r.Code
		}
		out = append(out, Station{Code: r.Code, Name: name, UIC: string(r.UICCode), Lat: r.Lat, Lng: r.Lng, Country: r.Land, Departures: r.HeeftVertrektijden})
	}
	return out, nil
}

// Stations fetches the station list.
func (c *Client) Stations(ctx context.Context) ([]Station, error) {
	res, err := c.Get(ctx, StationsPath, nil)
	if err != nil {
		return nil, err
	}
	return ParseStations(res.Body)
}
