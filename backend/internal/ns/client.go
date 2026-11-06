// Package ns talks to the NS API portal (gateway.apiportal.ns.nl).
package ns

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DefaultBaseURL is the NS API gateway.
const DefaultBaseURL = "https://gateway.apiportal.ns.nl"

// maxBody caps a response body; the full vehicle list is well under 1 MB.
const maxBody = 8 << 20

// ErrNotFound is returned for an HTTP 404, so callers can try an alternative path.
var ErrNotFound = errors.New("ns: not found")

// Client calls the NS API with a subscription key. The key is only ever sent in the
// Ocp-Apim-Subscription-Key header and never appears in errors or logs.
type Client struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
	// Retries is how many times a transient failure (network error, HTTP 502, 503 or 504) is
	// retried, waiting Backoff, then twice that, and so on. Zero means no retries. A 429 is
	// never retried: NS asks us to slow down, so the caller serves its cached copy instead.
	Retries int
	Backoff time.Duration
	// sleep waits between attempts; tests replace it so they run without real delays.
	sleep func(ctx context.Context, d time.Duration) error
}

// NewClient returns a client for the NS gateway with a 10 s timeout per attempt and two
// retries on transient failures.
func NewClient(key string) *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		Key:     key,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
		Retries: 2,
		Backoff: 500 * time.Millisecond,
	}
}

// Response is a raw NS response.
type Response struct {
	Status int
	Body   []byte
	Header http.Header
}

// Get fetches path with the given query and returns the raw response, retrying transient
// failures (see Client.Retries). A non-2xx status is returned as an error together with the
// response, so callers can still record it.
func (c *Client) Get(ctx context.Context, path string, query url.Values) (*Response, error) {
	wait := c.Backoff
	for attempt := 0; ; attempt++ {
		res, err := c.get(ctx, path, query)
		if err == nil || attempt >= c.Retries || !transient(res, err) || ctx.Err() != nil {
			return res, err
		}
		if serr := c.pause(ctx, wait); serr != nil {
			return res, err
		}
		wait *= 2
	}
}

// transient reports whether a failed attempt is worth retrying: a network error or a gateway
// error from NS. Client errors (4xx, including 429) and other server errors are not.
func transient(res *Response, err error) bool {
	if res == nil {
		return err != nil
	}
	switch res.Status {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

func (c *Client) pause(ctx context.Context, d time.Duration) error {
	if c.sleep != nil {
		return c.sleep(ctx, d)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Client) get(ctx context.Context, path string, query url.Values) (*Response, error) {
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", c.Key)
	req.Header.Set("Accept", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ns: GET %s: %w", path, err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("ns: read %s: %w", path, err)
	}
	r := &Response{Status: res.StatusCode, Body: body, Header: res.Header}
	switch {
	case res.StatusCode == http.StatusNotFound:
		return r, fmt.Errorf("ns: GET %s: %w", path, ErrNotFound)
	case res.StatusCode < 200 || res.StatusCode > 299:
		return r, fmt.Errorf("ns: GET %s: HTTP %d", path, res.StatusCode)
	}
	return r, nil
}

// VehiclePaths are the known paths of the live vehicle list. Community projects use both, so
// try them in order and keep the first that answers.
var VehiclePaths = []string{"/virtual-train-api/api/vehicle", "/virtual-train-api/vehicle"}

// Vehicles fetches and parses the live vehicle list, returning the path that worked.
func (c *Client) Vehicles(ctx context.Context) ([]Vehicle, string, error) {
	var lastErr error
	for _, p := range VehiclePaths {
		res, err := c.Get(ctx, p, nil)
		if errors.Is(err, ErrNotFound) {
			lastErr = err
			continue
		}
		if err != nil {
			return nil, p, err
		}
		v, err := ParseVehicles(res.Body)
		return v, p, err
	}
	return nil, "", lastErr
}
