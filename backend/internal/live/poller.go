// Package live holds the shared, in-memory copies of NS data: the vehicle list, refreshed in the
// background by one Poller, and per-key caches for everything fetched on demand. All users share
// them, so NS sees one caller however many people use the site.
package live

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// Snapshot is the vehicle list as last fetched. Vehicles is shared between callers and must not
// be modified.
type Snapshot struct {
	Vehicles []ns.Vehicle
	AsOf     time.Time // zero until the first successful poll
	Stale    bool      // no successful poll within StaleAfter
}

// Poller refreshes the vehicle list in the background. Set Fetch; the other fields have defaults.
type Poller struct {
	Fetch func(ctx context.Context) ([]ns.Vehicle, error)
	// Interval between polls (default 10 s: NS updates positions every 10 to 20 s). After a
	// failure the wait doubles, up to MaxInterval (default 60 s), so an NS outage is not hammered.
	Interval    time.Duration
	MaxInterval time.Duration
	// StaleAfter marks the snapshot stale when the last success is older (default 30 s).
	StaleAfter time.Duration
	Log        *slog.Logger

	// now and wait are the clock; tests replace them.
	now  func() time.Time
	wait func(ctx context.Context, d time.Duration) error

	mu       sync.RWMutex
	snap     Snapshot
	failures int
}

func (p *Poller) interval() time.Duration    { return orDefault(p.Interval, 10*time.Second) }
func (p *Poller) maxInterval() time.Duration { return orDefault(p.MaxInterval, time.Minute) }
func (p *Poller) staleAfter() time.Duration  { return orDefault(p.StaleAfter, 30*time.Second) }

func (p *Poller) clock() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

// Poll fetches the vehicle list once. On success it replaces the snapshot; on failure the last
// good list is kept and the error is returned.
func (p *Poller) Poll(ctx context.Context) error {
	// One poll never outlasts the interval, retries included, so polls cannot pile up.
	ctx, cancel := context.WithTimeout(ctx, p.interval())
	defer cancel()
	v, err := p.Fetch(ctx)
	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		p.failures++
		return err
	}
	p.failures = 0
	p.snap = Snapshot{Vehicles: v, AsOf: p.clock()}
	return nil
}

// Snapshot returns the current vehicle list with its age.
func (p *Poller) Snapshot() Snapshot {
	p.mu.RLock()
	s := p.snap
	p.mu.RUnlock()
	s.Stale = s.AsOf.IsZero() || p.clock().Sub(s.AsOf) > p.staleAfter()
	return s
}

// next is the wait before the following poll: the interval, doubled per consecutive failure.
func (p *Poller) next() time.Duration {
	p.mu.RLock()
	f := p.failures
	p.mu.RUnlock()
	d := p.interval()
	for i := 0; i < f && d < p.maxInterval(); i++ {
		d *= 2
	}
	return min(d, p.maxInterval())
}

// Run polls until ctx ends. Failures are logged (the error names the NS path, never the key).
func (p *Poller) Run(ctx context.Context) {
	wait := p.wait
	if wait == nil {
		wait = sleep
	}
	for {
		if err := p.Poll(ctx); err != nil && ctx.Err() == nil && p.Log != nil {
			p.Log.Warn("vehicle poll failed", "err", err, "next", p.next().String())
		}
		if wait(ctx, p.next()) != nil {
			return
		}
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func orDefault(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}
