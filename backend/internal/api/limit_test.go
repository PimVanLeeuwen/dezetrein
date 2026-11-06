package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLimiterBurstAndRefill(t *testing.T) {
	now := testNow
	l := newLimiter(60, 3, func() time.Time { return now }) // one token per second
	for i := range 3 {
		if ok, _ := l.allow("a"); !ok {
			t.Fatalf("request %d within the burst was refused", i+1)
		}
	}
	ok, wait := l.allow("a")
	if ok || wait <= 0 || wait > time.Second {
		t.Fatalf("over the burst: ok %v, wait %v", ok, wait)
	}
	if ok, _ := l.allow("b"); !ok {
		t.Error("another client must have its own bucket")
	}
	now = now.Add(time.Second)
	if ok, _ := l.allow("a"); !ok {
		t.Error("one token should have refilled after a second")
	}
}

func TestLimiterForgetsIdleClients(t *testing.T) {
	now := testNow
	l := newLimiter(60, 3, func() time.Time { return now })
	l.allow("a")
	l.allow("b")
	now = now.Add(2 * time.Minute)
	l.allow("c")
	if l.size() != 1 {
		t.Errorf("idle clients kept: %d buckets, want 1", l.size())
	}
}

func TestClientKey(t *testing.T) {
	for name, tc := range map[string]struct{ header, remote, want string }{
		"cloudflare":      {"203.0.113.7", "172.18.0.3:51000", "203.0.113.7"},
		"cloudflare ipv6": {"2001:db8::1", "172.18.0.3:51000", "2001:db8::1"},
		"no header":       {"", "192.0.2.4:1234", "192.0.2.4"},
		"garbage header":  {"not-an-ip, 1.2.3.4", "192.0.2.4:1234", "192.0.2.4"},
		"remote w/o port": {"", "192.0.2.4", "192.0.2.4"},
	} {
		r := httptest.NewRequest(http.MethodGet, "/api/x", nil)
		r.RemoteAddr = tc.remote
		if tc.header != "" {
			r.Header.Set("CF-Connecting-IP", tc.header)
		}
		if got := clientKey(r); got != tc.want {
			t.Errorf("%s: key %q, want %q", name, got, tc.want)
		}
	}
}

func TestAPIRateLimit(t *testing.T) {
	srv := newTestServer(t, &Server{NS: newFakeNS(t)})
	req := func(path, ip string) *http.Response {
		r, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		r.Header.Set("CF-Connecting-IP", ip)
		res, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res
	}
	// The clock is frozen in tests, so no tokens refill: exactly the burst gets through.
	for i := range apiBurst {
		if res := req("/api/trains/2450", "198.51.100.1"); res.StatusCode != http.StatusOK {
			t.Fatalf("request %d: status %d", i+1, res.StatusCode)
		}
	}
	res := req("/api/trains/2450", "198.51.100.1")
	if res.StatusCode != http.StatusTooManyRequests || res.Header.Get("Retry-After") == "" {
		t.Fatalf("over the limit: status %d, Retry-After %q", res.StatusCode, res.Header.Get("Retry-After"))
	}
	if res := req("/api/trains/2450", "198.51.100.2"); res.StatusCode != http.StatusOK {
		t.Errorf("another client: status %d", res.StatusCode)
	}
	if res := req("/healthz", "198.51.100.1"); res.StatusCode != http.StatusOK {
		t.Errorf("the health check is not rate limited: status %d", res.StatusCode)
	}
}

func TestMatchRateLimit(t *testing.T) {
	srv := newTestServer(t, &Server{NS: newFakeNS(t), Vehicles: matchFeed})
	body := `{"fixes":[` + fixJSON(52.3092, 4.7622, 30) + `]}`
	codes := map[int]int{}
	for range matchBurst + 1 {
		codes[postMatch(t, srv.URL, body).StatusCode]++
	}
	if codes[http.StatusOK] != matchBurst || codes[http.StatusTooManyRequests] != 1 {
		t.Errorf("want %d matches then a 429, got %v", matchBurst, codes)
	}
}

func TestRequestLogHasRouteNotPath(t *testing.T) {
	var logs bytes.Buffer
	srv := newTestServer(t, &Server{NS: newFakeNS(t), Log: slog.New(slog.NewJSONHandler(&logs, nil))})
	r, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/trains/2450?utm=secret", nil)
	r.Header.Set("CF-Connecting-IP", "203.0.113.99")
	res, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	get(t, srv.URL+"/healthz") // debug level: not in an info log
	get(t, srv.URL+"/nope")

	var lines []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("not a JSON log line: %q", l)
		}
		lines = append(lines, m)
	}
	if len(lines) != 2 {
		t.Fatalf("want 2 request lines (health check at debug), got %v", lines)
	}
	if lines[0]["route"] != "GET /api/trains/{train}" || lines[0]["status"] != float64(200) {
		t.Errorf("unexpected line: %v", lines[0])
	}
	if lines[1]["route"] != "unmatched" || lines[1]["status"] != float64(404) {
		t.Errorf("unexpected line: %v", lines[1])
	}
	for _, leak := range []string{"2450", "secret", "203.0.113.99", "127.0.0.1"} {
		if strings.Contains(logs.String(), leak) {
			t.Errorf("log contains %q: %s", leak, logs.String())
		}
	}
}

func TestAPISecurityHeaders(t *testing.T) {
	srv := newTestServer(t, &Server{})
	res := get(t, srv.URL+"/api/nope")
	if res.Header.Get("Content-Security-Policy") != "default-src 'none'; frame-ancestors 'none'" {
		t.Errorf("CSP %q", res.Header.Get("Content-Security-Policy"))
	}
}
