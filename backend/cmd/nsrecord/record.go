package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// Entry describes one recorded call in index.json.
type Entry struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Query  string `json:"query,omitempty"`
	Status int    `json:"status"`
	Bytes  int    `json:"bytes"`
	File   string `json:"file,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Index is written next to the recordings.
type Index struct {
	RecordedAt time.Time `json:"recordedAt"`
	Train      string    `json:"train"`
	Station    string    `json:"station"`
	Entries    []Entry   `json:"entries"`
}

type call struct {
	name  string
	path  string
	query url.Values
}

// trainCalls are the per-train and general endpoints recorded after the vehicle list.
func trainCalls(train, station string) []call {
	return []call{
		{"journey", "/reisinformatie-api/api/v2/journey", url.Values{"train": {train}, "omitCrowdForecast": {"false"}}},
		{"composition", "/virtual-train-api/api/v1/trein/" + url.PathEscape(train), nil},
		{"crowding", "/virtual-train-api/api/v1/prognose/" + url.PathEscape(train), nil},
		{"disruptions", "/reisinformatie-api/api/v3/disruptions", url.Values{"isActive": {"true"}}},
		{"departures", "/reisinformatie-api/api/v2/departures", url.Values{"station": {station}}},
	}
}

// Record fetches the vehicle list and the endpoints for one train, and writes each response to
// outDir as <name>.json plus an index.json. A failing call is recorded and does not stop the run.
// When train is empty, a moving NS intercity or sprinter from the vehicle list is used.
func Record(ctx context.Context, c *ns.Client, outDir, train, station string, now time.Time) (*Index, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	idx := &Index{RecordedAt: now.UTC(), Station: station}

	var vehicles []ns.Vehicle
	for _, p := range ns.VehiclePaths {
		e, res := record(ctx, c, outDir, call{"vehicles", p, nil})
		idx.Entries = append(idx.Entries, e)
		if res == nil || res.Status == 404 {
			continue
		}
		if e.Error == "" {
			v, err := ns.ParseVehicles(res.Body)
			if err != nil {
				idx.Entries[len(idx.Entries)-1].Error = err.Error()
			}
			vehicles = v
		}
		break
	}

	if train == "" {
		train = pickTrain(vehicles)
	}
	idx.Train = train
	for _, cl := range trainCalls(train, station) {
		if train == "" && cl.name != "disruptions" && cl.name != "departures" {
			idx.Entries = append(idx.Entries, Entry{Name: cl.name, Path: cl.path, Error: "skipped: no train number (pass -train)"})
			continue
		}
		e, _ := record(ctx, c, outDir, cl)
		idx.Entries = append(idx.Entries, e)
	}

	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return idx, err
	}
	return idx, os.WriteFile(filepath.Join(outDir, "index.json"), append(b, '\n'), 0o644)
}

func record(ctx context.Context, c *ns.Client, outDir string, cl call) (Entry, *ns.Response) {
	e := Entry{Name: cl.name, Path: cl.path, Query: cl.query.Encode()}
	res, err := c.Get(ctx, cl.path, cl.query)
	if err != nil {
		e.Error = err.Error()
	}
	if res == nil {
		return e, nil
	}
	e.Status, e.Bytes = res.Status, len(res.Body)
	if res.Status == 404 && errors.Is(err, ns.ErrNotFound) && cl.name == "vehicles" {
		return e, res
	}
	body := res.Body
	var pretty bytes.Buffer
	ext := ".txt"
	if json.Indent(&pretty, body, "", "  ") == nil {
		body, ext = append(pretty.Bytes(), '\n'), ".json"
	}
	e.File = cl.name + ext
	if werr := os.WriteFile(filepath.Join(outDir, e.File), body, 0o644); werr != nil {
		e.Error = fmt.Sprintf("write: %v", werr)
	}
	return e, res
}

// pickTrain prefers a moving NS (OBIS) intercity, then any moving IC or sprinter.
func pickTrain(vs []ns.Vehicle) string {
	var fallback string
	for _, v := range vs {
		if v.SpeedKmh < 20 || (v.Type != "IC" && v.Type != "SPR" && v.Type != "ICD") {
			continue
		}
		if v.Source == "OBIS" && v.Type == "IC" {
			return v.Number()
		}
		if fallback == "" {
			fallback = v.Number()
		}
	}
	return fallback
}

// StationPaths are the known paths of the station list, tried in order.
var StationPaths = []string{"/reisinformatie-api/api/v2/stations", "/reisinformatie-api/api/v3/stations"}

// RecordStations records only the station list into outDir and adds it to the existing
// index.json (replacing an earlier "stations" entry), leaving the other recordings alone.
func RecordStations(ctx context.Context, c *ns.Client, outDir string) (*Index, error) {
	idx := &Index{}
	if b, err := os.ReadFile(filepath.Join(outDir, "index.json")); err == nil {
		if err := json.Unmarshal(b, idx); err != nil {
			return nil, fmt.Errorf("index.json: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	} else if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	var got Entry
	for _, p := range StationPaths {
		e, res := record(ctx, c, outDir, call{"stations", p, nil})
		got = e
		if res != nil && res.Status != 404 {
			break
		}
	}
	kept := idx.Entries[:0]
	for _, e := range idx.Entries {
		if e.Name != "stations" {
			kept = append(kept, e)
		}
	}
	idx.Entries = append(kept, got)
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return idx, err
	}
	return idx, os.WriteFile(filepath.Join(outDir, "index.json"), append(b, '\n'), 0o644)
}
