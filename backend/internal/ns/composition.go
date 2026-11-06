package ns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Composition is the stock of one journey from the Virtual Train API (api/v1/trein/{ritnummer}).
type Composition struct {
	Train     string   `json:"train"`
	Type      string   `json:"type"`
	Carriages []string `json:"carriages"` // trainset numbers (materieelnummer), as printed in the train
	LengthM   int      `json:"lengthM,omitempty"`
}

// ParseComposition reads a composition response. Carriages keeps the order NS gives.
func ParseComposition(body []byte) (*Composition, error) {
	var raw struct {
		Ritnummer      flexString `json:"ritnummer"`
		Type           string     `json:"type"`
		LengteInMeters int        `json:"lengteInMeters"`
		Materieeldelen []struct {
			Materieelnummer flexString `json:"materieelnummer"`
		} `json:"materieeldelen"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("ns: parse composition: %w", err)
	}
	c := &Composition{Train: string(raw.Ritnummer), Type: raw.Type, LengthM: raw.LengteInMeters, Carriages: []string{}}
	for _, m := range raw.Materieeldelen {
		if m.Materieelnummer != "" {
			c.Carriages = append(c.Carriages, string(m.Materieelnummer))
		}
	}
	return c, nil
}

// Composition fetches the stock of one journey by its journey number (ritnummer).
func (c *Client) Composition(ctx context.Context, train string) (*Composition, error) {
	res, err := c.Get(ctx, "/virtual-train-api/api/v1/trein/"+url.PathEscape(train), nil)
	if err != nil {
		return nil, err
	}
	comp, err := ParseComposition(res.Body)
	if err != nil {
		return nil, err
	}
	if comp.Train == "" {
		comp.Train = train
	}
	return comp, nil
}
