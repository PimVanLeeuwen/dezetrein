package ns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// retryClient answers with the given statuses in turn (the last one repeats) and records the
// waits between attempts instead of sleeping.
func retryClient(t *testing.T, statuses ...int) (*Client, *atomic.Int32, *[]time.Duration) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(calls.Add(1)) - 1
		w.WriteHeader(statuses[min(n, len(statuses)-1)])
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	var waits []time.Duration
	c := &Client{BaseURL: srv.URL, Key: "k", HTTP: srv.Client(), Retries: 2, Backoff: 100 * time.Millisecond}
	c.sleep = func(ctx context.Context, d time.Duration) error {
		waits = append(waits, d)
		return ctx.Err()
	}
	return c, &calls, &waits
}

func TestGetRetriesGatewayErrorsWithBackoff(t *testing.T) {
	c, calls, waits := retryClient(t, http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusOK)
	res, err := c.Get(context.Background(), "/x", nil)
	if err != nil || res.Status != http.StatusOK {
		t.Fatalf("want success on the third attempt, got %v %v", res, err)
	}
	if calls.Load() != 3 || len(*waits) != 2 || (*waits)[0] != 100*time.Millisecond || (*waits)[1] != 200*time.Millisecond {
		t.Errorf("calls %d, waits %v", calls.Load(), *waits)
	}
}

func TestGetGivesUpAfterRetries(t *testing.T) {
	c, calls, _ := retryClient(t, http.StatusGatewayTimeout)
	res, err := c.Get(context.Background(), "/x", nil)
	if err == nil || res == nil || res.Status != http.StatusGatewayTimeout {
		t.Fatalf("want the last 504, got %v %v", res, err)
	}
	if calls.Load() != 3 {
		t.Errorf("calls %d, want 1 plus 2 retries", calls.Load())
	}
}

func TestGetDoesNotRetryClientErrors(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusUnauthorized, http.StatusNotFound, http.StatusInternalServerError} {
		c, calls, _ := retryClient(t, status)
		if _, err := c.Get(context.Background(), "/x", nil); err == nil {
			t.Errorf("%d: want an error", status)
		}
		if calls.Load() != 1 {
			t.Errorf("%d: %d calls, want no retry", status, calls.Load())
		}
	}
}

func TestGetRetriesNetworkErrors(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens any more: every attempt is a connection error
	attempts := 0
	c := &Client{BaseURL: url, Key: "k", HTTP: &http.Client{Timeout: time.Second}, Retries: 1, Backoff: time.Millisecond}
	c.sleep = func(ctx context.Context, d time.Duration) error { attempts++; return nil }
	if _, err := c.Get(context.Background(), "/x", nil); err == nil {
		t.Fatal("want a connection error")
	}
	if attempts != 1 {
		t.Errorf("waited %d times, want 1 retry", attempts)
	}
}

func TestGetStopsRetryingWhenContextEnds(t *testing.T) {
	c, calls, _ := retryClient(t, http.StatusServiceUnavailable)
	ctx, cancel := context.WithCancel(context.Background())
	c.sleep = func(context.Context, time.Duration) error { cancel(); return context.Canceled }
	if _, err := c.Get(ctx, "/x", nil); err == nil {
		t.Fatal("want an error")
	}
	if calls.Load() != 1 {
		t.Errorf("calls %d, want 1", calls.Load())
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("k")
	if c.BaseURL != DefaultBaseURL || c.HTTP.Timeout != 10*time.Second || c.Retries != 2 || c.Backoff != 500*time.Millisecond {
		t.Errorf("unexpected defaults: %+v", c)
	}
}
