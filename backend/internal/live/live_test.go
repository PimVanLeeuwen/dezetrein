package live

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// fakeClock is a settable clock shared by a test and the code under test.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newClock() *fakeClock { return &fakeClock{t: time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

var errNS = errors.New("ns: GET /x: HTTP 503")

func TestPollerSnapshotAgeAndStale(t *testing.T) {
	clk := newClock()
	fail := false
	p := &Poller{now: clk.Now, Fetch: func(context.Context) ([]ns.Vehicle, error) {
		if fail {
			return nil, errNS
		}
		return []ns.Vehicle{{RitID: "3522"}}, nil
	}}
	if s := p.Snapshot(); !s.Stale || !s.AsOf.IsZero() || s.Vehicles != nil {
		t.Fatalf("before the first poll: %+v", s)
	}
	if err := p.Poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	start := clk.Now()
	clk.Advance(30 * time.Second)
	if s := p.Snapshot(); s.Stale || !s.AsOf.Equal(start) || len(s.Vehicles) != 1 {
		t.Fatalf("30 s after a poll: %+v", s)
	}
	fail = true
	if err := p.Poll(context.Background()); !errors.Is(err, errNS) {
		t.Fatalf("want the fetch error, got %v", err)
	}
	clk.Advance(time.Second)
	if s := p.Snapshot(); !s.Stale || !s.AsOf.Equal(start) || len(s.Vehicles) != 1 {
		t.Fatalf("after a failed poll, want the last good list marked stale: %+v", s)
	}
}

func TestPollerBacksOffOnFailures(t *testing.T) {
	clk := newClock()
	results := []error{nil, errNS, errNS, errNS, errNS, nil}
	var waits []time.Duration
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := &Poller{
		now: clk.Now,
		Fetch: func(context.Context) ([]ns.Vehicle, error) {
			err := results[0]
			results = results[1:]
			return nil, err
		},
		wait: func(_ context.Context, d time.Duration) error {
			waits = append(waits, d)
			clk.Advance(d)
			if len(results) == 0 {
				cancel()
				return context.Canceled
			}
			return nil
		},
	}
	p.Run(ctx)
	want := []time.Duration{10 * time.Second, 20 * time.Second, 40 * time.Second, time.Minute, time.Minute, 10 * time.Second}
	if fmt.Sprint(waits) != fmt.Sprint(want) {
		t.Errorf("waits %v, want %v", waits, want)
	}
}

func TestPollerBoundsOnePoll(t *testing.T) {
	p := &Poller{Interval: 20 * time.Millisecond, Fetch: func(ctx context.Context) ([]ns.Vehicle, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	if err := p.Poll(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("want the poll cut off at the interval, got %v", err)
	}
}

func TestPollerRunStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var polls atomic.Int32
	p := &Poller{Interval: time.Hour, Fetch: func(context.Context) ([]ns.Vehicle, error) {
		polls.Add(1)
		cancel()
		return nil, nil
	}}
	done := make(chan struct{})
	go func() { p.Run(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop when its context ended")
	}
	if polls.Load() != 1 {
		t.Errorf("polls %d, want 1", polls.Load())
	}
}

func TestCacheServesFreshThenRefreshes(t *testing.T) {
	clk := newClock()
	c := &Cache[string]{TTL: 30 * time.Second, now: clk.Now}
	var calls atomic.Int32
	fetch := func(context.Context) (string, error) { return fmt.Sprint("v", calls.Add(1)), nil }
	r, err := c.Get(context.Background(), "3522", fetch)
	if err != nil || r.Value != "v1" || r.Stale || !r.AsOf.Equal(clk.Now()) {
		t.Fatalf("first get: %+v %v", r, err)
	}
	clk.Advance(29 * time.Second)
	if r, _ := c.Get(context.Background(), "3522", fetch); r.Value != "v1" || calls.Load() != 1 {
		t.Fatalf("within TTL, want the cached value without a fetch: %+v, %d calls", r, calls.Load())
	}
	clk.Advance(time.Second)
	if r, _ := c.Get(context.Background(), "3522", fetch); r.Value != "v2" || calls.Load() != 2 {
		t.Fatalf("after TTL, want a refresh: %+v, %d calls", r, calls.Load())
	}
}

func TestCacheServesStaleOnFailure(t *testing.T) {
	clk := newClock()
	c := &Cache[string]{TTL: 30 * time.Second, now: clk.Now}
	ok := func(context.Context) (string, error) { return "good", nil }
	bad := func(context.Context) (string, error) { return "", errNS }
	if _, err := c.Get(context.Background(), "UT", bad); !errors.Is(err, errNS) {
		t.Fatalf("no earlier value: want the error, got %v", err)
	}
	c.Get(context.Background(), "UT", ok)
	asOf := clk.Now()
	clk.Advance(time.Minute)
	r, err := c.Get(context.Background(), "UT", bad)
	if err != nil || r.Value != "good" || !r.Stale || !r.AsOf.Equal(asOf) {
		t.Fatalf("want the last good value as stale: %+v %v", r, err)
	}
}

func TestCacheSharesOneFetch(t *testing.T) {
	c := &Cache[int]{TTL: time.Minute}
	var calls atomic.Int32
	release := make(chan struct{})
	fetch := func(context.Context) (int, error) {
		calls.Add(1)
		<-release
		return 42, nil
	}
	var wg sync.WaitGroup
	results := make(chan int, 20)
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := c.Get(context.Background(), "3522", fetch)
			if err != nil {
				t.Error(err)
			}
			results <- r.Value
		}()
	}
	// Let the goroutines pile up on the one in-flight fetch, then release it.
	for c.inFlight("3522") == false {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(10 * time.Millisecond)
	close(release)
	wg.Wait()
	close(results)
	for v := range results {
		if v != 42 {
			t.Errorf("got %d", v)
		}
	}
	if calls.Load() != 1 {
		t.Errorf("%d fetches for 20 concurrent misses, want 1", calls.Load())
	}
}

func TestCacheCallerLeavingDoesNotCancelFetch(t *testing.T) {
	c := &Cache[string]{TTL: time.Minute}
	release := make(chan struct{})
	fetch := func(ctx context.Context) (string, error) {
		<-release
		return "done", ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for !c.inFlight("k") {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	if _, err := c.Get(ctx, "k", fetch); !errors.Is(err, context.Canceled) {
		t.Fatalf("leaving caller: want context.Canceled, got %v", err)
	}
	close(release)
	r, err := c.Get(context.Background(), "k", fetch)
	if err != nil || r.Value != "done" {
		t.Fatalf("the detached fetch should have completed for the next caller: %+v %v", r, err)
	}
}

func TestCacheRemembersNotFound(t *testing.T) {
	clk := newClock()
	c := &Cache[string]{TTL: time.Minute, NotFoundTTL: 10 * time.Second, now: clk.Now}
	var calls atomic.Int32
	fetch := func(context.Context) (string, error) {
		calls.Add(1)
		return "", fmt.Errorf("ns: GET /journey: %w", ns.ErrNotFound)
	}
	for range 3 {
		if _, err := c.Get(context.Background(), "99999", fetch); !errors.Is(err, ns.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("%d fetches, want 1 while the 404 is remembered", calls.Load())
	}
	clk.Advance(10 * time.Second)
	c.Get(context.Background(), "99999", fetch)
	if calls.Load() != 2 {
		t.Errorf("%d fetches, want a retry after NotFoundTTL", calls.Load())
	}
}

func TestCacheEvictsWhenFull(t *testing.T) {
	clk := newClock()
	c := &Cache[string]{TTL: time.Minute, MaxEntries: 3, now: clk.Now}
	ok := func(context.Context) (string, error) { return "v", nil }
	for _, k := range []string{"a", "b", "c"} {
		c.Get(context.Background(), k, ok)
		clk.Advance(time.Second)
	}
	c.Get(context.Background(), "d", ok)
	if c.Len() != 3 || c.has("a") || !c.has("d") {
		t.Errorf("want the oldest entry evicted: len %d, a %v, d %v", c.Len(), c.has("a"), c.has("d"))
	}
	clk.Advance(2 * time.Minute)
	c.Get(context.Background(), "e", ok)
	if c.Len() != 1 || !c.has("e") {
		t.Errorf("want all expired entries dropped: len %d", c.Len())
	}
}

func (c *Cache[T]) inFlight(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[key]
	return e != nil && e.call != nil
}

func (c *Cache[T]) has(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.entries[key]
	return ok
}

func TestPollerKeepsHistory(t *testing.T) {
	clk := newClock()
	n := 0
	p := &Poller{now: clk.Now, Keep: time.Minute, Fetch: func(context.Context) ([]ns.Vehicle, error) {
		n++
		if n == 3 {
			return nil, errNS
		}
		return []ns.Vehicle{{RitID: fmt.Sprint(n)}}, nil
	}}
	if h := p.History(); len(h) != 0 {
		t.Fatalf("before the first poll: %d snapshots", len(h))
	}
	for range 8 { // one poll every 10 s; poll 3 fails
		p.Poll(context.Background())
		clk.Advance(10 * time.Second)
	}
	h := p.History()
	// The last poll ran at +70 s, so +10 s and later are kept: polls 2, 4, 5, 6, 7, 8.
	var got []string
	for _, s := range h {
		got = append(got, s.Vehicles[0].RitID)
	}
	if fmt.Sprint(got) != "[2 4 5 6 7 8]" {
		t.Errorf("history %v, want [2 4 5 6 7 8]", got)
	}
	if !h[len(h)-1].AsOf.Equal(p.Snapshot().AsOf) {
		t.Error("the newest history entry must be the current snapshot")
	}
}
