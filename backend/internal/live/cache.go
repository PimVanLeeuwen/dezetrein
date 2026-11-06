package live

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// Result is a cached value with its age. Stale means the last refresh failed and this is the
// last good copy.
type Result[T any] struct {
	Value T
	AsOf  time.Time
	Stale bool
}

// Cache keeps one value per key (a train number, a station code) for TTL. Concurrent misses for
// the same key share one fetch, and when a refresh fails the last good value is served as stale
// instead of an error. The zero value is not usable; set at least TTL.
type Cache[T any] struct {
	TTL time.Duration
	// NotFoundTTL is how long an ns.ErrNotFound answer is remembered (default TTL), so repeated
	// lookups of an unknown train do not each reach NS.
	NotFoundTTL time.Duration
	// FetchTimeout bounds one fetch (default 20 s). The fetch is detached from the caller that
	// started it, so callers that leave early do not cancel it for the others.
	FetchTimeout time.Duration
	// MaxEntries caps memory (default 1000). When full, expired entries go first, then the oldest.
	MaxEntries int

	now func() time.Time // tests replace it

	mu      sync.Mutex
	entries map[string]*entry[T]
}

type entry[T any] struct {
	value    T
	asOf     time.Time // zero while no value has been fetched yet
	notFound time.Time // when NS last answered 404 without a value to fall back on
	call     *call[T]
}

type call[T any] struct {
	done  chan struct{}
	value T
	err   error
}

func (c *Cache[T]) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

// Get returns the value for key, calling fetch when it is missing or older than TTL. It fails
// only when fetch fails and there is no earlier value, or when ctx ends while waiting.
func (c *Cache[T]) Get(ctx context.Context, key string, fetch func(ctx context.Context) (T, error)) (Result[T], error) {
	c.mu.Lock()
	if c.entries == nil {
		c.entries = map[string]*entry[T]{}
	}
	e := c.entries[key]
	if e == nil {
		c.evict()
		e = &entry[T]{}
		c.entries[key] = e
	}
	now := c.clock()
	if !e.asOf.IsZero() && now.Sub(e.asOf) < c.TTL {
		r := Result[T]{Value: e.value, AsOf: e.asOf}
		c.mu.Unlock()
		return r, nil
	}
	if e.asOf.IsZero() && !e.notFound.IsZero() && now.Sub(e.notFound) < orDefault(c.NotFoundTTL, c.TTL) {
		c.mu.Unlock()
		return Result[T]{}, ns.ErrNotFound
	}
	cl := e.call
	if cl == nil {
		cl = &call[T]{done: make(chan struct{})}
		e.call = cl
		go c.run(ctx, e, cl, fetch)
	}
	c.mu.Unlock()

	select {
	case <-ctx.Done():
		return Result[T]{}, ctx.Err()
	case <-cl.done:
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if cl.err == nil {
		return Result[T]{Value: cl.value, AsOf: e.asOf}, nil
	}
	if !e.asOf.IsZero() {
		return Result[T]{Value: e.value, AsOf: e.asOf, Stale: true}, nil
	}
	return Result[T]{}, cl.err
}

func (c *Cache[T]) run(ctx context.Context, e *entry[T], cl *call[T], fetch func(context.Context) (T, error)) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), orDefault(c.FetchTimeout, 20*time.Second))
	defer cancel()
	v, err := fetch(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	cl.value, cl.err = v, err
	switch {
	case err == nil:
		e.value, e.asOf, e.notFound = v, c.clock(), time.Time{}
	case errors.Is(err, ns.ErrNotFound):
		e.notFound = c.clock()
	}
	e.call = nil // evict skips entries with a fetch in flight, so e is still in the map
	close(cl.done)
}

// evict makes room for one more entry. Called with mu held.
func (c *Cache[T]) evict() {
	limit := c.MaxEntries
	if limit <= 0 {
		limit = 1000
	}
	if len(c.entries) < limit {
		return
	}
	now := c.clock()
	var oldestKey string
	var oldest time.Time
	for k, e := range c.entries {
		if e.call != nil {
			continue
		}
		seen := e.asOf
		if seen.IsZero() {
			seen = e.notFound
		}
		if now.Sub(seen) >= c.TTL {
			delete(c.entries, k)
			continue
		}
		if oldestKey == "" || seen.Before(oldest) {
			oldestKey, oldest = k, seen
		}
	}
	if len(c.entries) >= limit && oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// Len is the number of keys held, for tests and health output.
func (c *Cache[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
