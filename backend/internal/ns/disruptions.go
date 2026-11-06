package ns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// Disruption is an active disruption, maintenance work or calamity from the Reisinformatie API
// (api/v3/disruptions). Texts are Dutch, as NS publishes them.
type Disruption struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // DISRUPTION, MAINTENANCE or CALAMITY
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Start       time.Time `json:"start,omitzero"`
	End         time.Time `json:"end,omitzero"`
	Impact      int       `json:"impact,omitempty"` // NS scale, higher is worse
	// Stations are the station codes of the affected sections, used to find the disruptions on
	// a train's route. Calamities have none: they are shown network-wide.
	Stations []string `json:"stations,omitempty"`
	URL      string   `json:"url,omitempty"`
}

// ParseDisruptions reads a disruptions response, a bare JSON array. Entries without an id are
// dropped.
func ParseDisruptions(body []byte) ([]Disruption, error) {
	type station struct {
		StationCode string `json:"stationCode"`
	}
	var raw []struct {
		ID          string `json:"id"`
		Type        string `json:"type"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Start       nsTime `json:"start"`
		End         nsTime `json:"end"`
		URL         string `json:"url"`
		Impact      struct {
			Value int `json:"value"`
		} `json:"impact"`
		Timespans []struct {
			Situation struct {
				Label string `json:"label"`
			} `json:"situation"`
		} `json:"timespans"`
		PublicationSections []struct {
			Section struct {
				Stations []station `json:"stations"`
			} `json:"section"`
			Consequence struct {
				Section struct {
					Stations []station `json:"stations"`
				} `json:"section"`
			} `json:"consequence"`
		} `json:"publicationSections"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("ns: parse disruptions: %w", err)
	}
	out := make([]Disruption, 0, len(raw))
	for _, r := range raw {
		if r.ID == "" {
			continue
		}
		d := Disruption{
			ID:          r.ID,
			Type:        r.Type,
			Title:       r.Title,
			Description: r.Description,
			Start:       r.Start.Time,
			End:         r.End.Time,
			Impact:      r.Impact.Value,
			URL:         r.URL,
		}
		if d.Description == "" && len(r.Timespans) > 0 {
			d.Description = r.Timespans[0].Situation.Label
		}
		seen := map[string]bool{}
		for _, p := range r.PublicationSections {
			for _, list := range [][]station{p.Section.Stations, p.Consequence.Section.Stations} {
				for _, s := range list {
					if s.StationCode != "" && !seen[s.StationCode] {
						seen[s.StationCode] = true
						d.Stations = append(d.Stations, s.StationCode)
					}
				}
			}
		}
		out = append(out, d)
	}
	return out, nil
}

// Disruptions fetches all active disruptions. One call covers the whole network, so callers
// cache it and filter per train.
func (c *Client) Disruptions(ctx context.Context) ([]Disruption, error) {
	res, err := c.Get(ctx, "/reisinformatie-api/api/v3/disruptions", url.Values{"isActive": {"true"}})
	if err != nil {
		return nil, err
	}
	return ParseDisruptions(res.Body)
}
