package api

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Rate limits per client. Generous on purpose: train Wi-Fi puts a whole train behind one IP
// address, and NS is protected by the shared caches, not by this. The limits stop scripts and
// runaway loops.
const (
	apiPerMinute = 300
	apiBurst     = 100
	// The lookup limit, shared by /api/match and /api/carriage.
	matchPerMinute = 60
	matchBurst     = 20
	// maxClients caps the limiter's memory; idle clients are dropped first.
	maxClients = 10000
)

// limiter is a token bucket per client key. Keys are kept in memory only and never logged.
type limiter struct {
	rate  float64 // tokens per second
	burst float64
	now   func() time.Time

	mu      sync.Mutex
	buckets map[string]*bucket
	swept   time.Time
}

type bucket struct {
	tokens float64
	at     time.Time
}

func newLimiter(perMinute, burst int, now func() time.Time) *limiter {
	return &limiter{rate: float64(perMinute) / 60, burst: float64(burst), now: now, buckets: map[string]*bucket{}}
}

// allow takes one token for key. When none is left it returns false and how long until the
// next token.
func (l *limiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: l.burst, at: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.at).Seconds()*l.rate)
	b.at = now
	if b.tokens < 1 {
		return false, time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	}
	b.tokens--
	return true, 0
}

// sweep drops clients whose bucket has refilled (they have been idle long enough to be
// forgotten), once a minute or when the map is full. Called with mu held.
func (l *limiter) sweep(now time.Time) {
	if now.Sub(l.swept) < time.Minute && len(l.buckets) < maxClients {
		return
	}
	l.swept = now
	full := time.Duration(l.burst / l.rate * float64(time.Second))
	for k, b := range l.buckets {
		if now.Sub(b.at) >= full {
			delete(l.buckets, k)
		}
	}
	// Still full after dropping idle clients: start over rather than grow without bound.
	if len(l.buckets) >= maxClients {
		clear(l.buckets)
	}
}

func (l *limiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}

// clientKey identifies the client for rate limiting: the CF-Connecting-IP header that
// Cloudflare sets in front of the site, or the connection's address when it is absent (local
// development). Only a valid IP address is accepted from the header.
func clientKey(r *http.Request) string {
	if ip := net.ParseIP(r.Header.Get("CF-Connecting-IP")); ip != nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// limit applies l to every request through next, answering 429 with Retry-After when the
// client is over its limit.
func limit(l *limiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ok, wait := l.allow(clientKey(r)); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
			writeError(w, http.StatusTooManyRequests, "rate_limited")
			return
		}
		next(w, r)
	}
}
