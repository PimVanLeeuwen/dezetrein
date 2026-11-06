package ns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Crowding is the crowd forecast of one journey from the Virtual Train API
// (api/v1/prognose/{ritnummer}), keyed by station UIC code. Values are LOW, MEDIUM, HIGH or
// UNKNOWN, as NS gives them.
type Crowding map[string]string

// ParseCrowding reads a crowd forecast response. Entries without a station are dropped.
func ParseCrowding(body []byte) (Crowding, error) {
	var raw struct {
		Prognoses []struct {
			StationUIC     flexString `json:"stationUic"`
			Classification string     `json:"classification"`
		} `json:"prognoses"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("ns: parse crowding: %w", err)
	}
	c := Crowding{}
	for _, p := range raw.Prognoses {
		if p.StationUIC != "" && p.Classification != "" {
			c[string(p.StationUIC)] = p.Classification
		}
	}
	return c, nil
}

// Crowding fetches the crowd forecast of one journey by its journey number (ritnummer).
func (c *Client) Crowding(ctx context.Context, train string) (Crowding, error) {
	res, err := c.Get(ctx, "/virtual-train-api/api/v1/prognose/"+url.PathEscape(train), nil)
	if err != nil {
		return nil, err
	}
	return ParseCrowding(res.Body)
}
