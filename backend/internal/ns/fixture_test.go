package ns

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFixtureClientReplaysRecordings(t *testing.T) {
	c, err := NewFixtureClient(filepath.Join("..", "..", "testdata", "ns"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	v, _, err := c.Vehicles(ctx)
	if err != nil || len(v) < 50 {
		t.Fatalf("vehicles: %d, %v", len(v), err)
	}
	if j, err := c.Journey(ctx, "2450"); err != nil || j.Destination != "Rotterdam Centraal" {
		t.Errorf("journey: %+v %v", j, err)
	}
	if d, err := c.Departures(ctx, "UT"); err != nil || len(d) == 0 {
		t.Errorf("departures: %d %v", len(d), err)
	}
	if _, err := c.Journey(ctx, "3522"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unrecorded train: want ErrNotFound, got %v", err)
	}
	if _, err := c.Departures(ctx, "ASD"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unrecorded station: want ErrNotFound, got %v", err)
	}
}

func TestFixtureClientRejectsBadIndex(t *testing.T) {
	write := func(index string) string {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte(index), 0o600); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	for name, dir := range map[string]string{
		"missing index": t.TempDir(),
		"not json":      write(`nope`),
		"path escape":   write(`{"entries":[{"path":"/x","file":"../../etc/passwd"}]}`),
		"hidden file":   write(`{"entries":[{"path":"/x","file":".env"}]}`),
		"missing file":  write(`{"entries":[{"path":"/x","file":"gone.json"}]}`),
	} {
		if _, err := NewFixtureClient(dir); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
