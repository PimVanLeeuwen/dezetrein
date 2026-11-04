package ns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// Vehicle is one live train position from the Virtual Train API. Field names follow the NS
// payload (Dutch); the API has no public schema, so parsing is lenient.
type Vehicle struct {
	TrainNumber string  `json:"treinNummer"`
	RitID       string  `json:"ritId"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	SpeedKmh    float64 `json:"snelheid"`
	HeadingDeg  float64 `json:"richting"`
	Type        string  `json:"type"`
	Source      string  `json:"bron"`
	// AccuracyM is horizontaleNauwkeurigheid, kept for analysis only: older reports call it
	// nonsense, but the 2026-11-04 recording held plausible values. Nil when absent or invalid.
	AccuracyM *float64 `json:"horizontaleNauwkeurigheid,omitempty"`
}

// Number is the train number (ritnummer) used by the other NS endpoints.
func (v Vehicle) Number() string {
	if v.RitID != "" {
		return v.RitID
	}
	return v.TrainNumber
}

type rawVehicle struct {
	TrainNumber flexString `json:"treinNummer"`
	RitID       flexString `json:"ritId"`
	Lat         *float64   `json:"lat"`
	Lng         *float64   `json:"lng"`
	Speed       *float64   `json:"snelheid"`
	Heading     *float64   `json:"richting"`
	Accuracy    *float64   `json:"horizontaleNauwkeurigheid"`
	Type        string     `json:"type"`
	Source      string     `json:"bron"`
}

// ParseVehicles reads the vehicle list, either wrapped as {"payload":{"treinen":[...]}} or as
// {"treinen":[...]}. Entries without a usable position or train number are dropped.
func ParseVehicles(body []byte) ([]Vehicle, error) {
	var doc struct {
		Payload *struct {
			Treinen []rawVehicle `json:"treinen"`
		} `json:"payload"`
		Treinen []rawVehicle `json:"treinen"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("ns: parse vehicles: %w", err)
	}
	raw := doc.Treinen
	if doc.Payload != nil {
		raw = doc.Payload.Treinen
	}
	out := make([]Vehicle, 0, len(raw))
	for _, r := range raw {
		if r.Lat == nil || r.Lng == nil || !finite(*r.Lat) || !finite(*r.Lng) || (*r.Lat == 0 && *r.Lng == 0) {
			continue
		}
		v := Vehicle{
			TrainNumber: string(r.TrainNumber),
			RitID:       string(r.RitID),
			Lat:         *r.Lat,
			Lng:         *r.Lng,
			Type:        r.Type,
			Source:      r.Source,
		}
		if v.Number() == "" {
			continue
		}
		if r.Speed != nil && finite(*r.Speed) {
			v.SpeedKmh = *r.Speed
		}
		if r.Heading != nil && finite(*r.Heading) {
			v.HeadingDeg = *r.Heading
		}
		if r.Accuracy != nil && finite(*r.Accuracy) && *r.Accuracy >= 0 {
			a := *r.Accuracy
			v.AccuracyM = &a
		}
		out = append(out, v)
	}
	return out, nil
}

// flexString accepts a JSON string or number (NS is not consistent about train numbers).
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if bytes.Equal(b, []byte("null")) {
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	if i, err := strconv.ParseInt(string(n), 10, 64); err == nil {
		*f = flexString(strconv.FormatInt(i, 10))
		return nil
	}
	*f = flexString(n.String())
	return nil
}

func finite(f float64) bool { return !math.IsNaN(f) && !math.IsInf(f, 0) }
