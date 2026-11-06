package ns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FixtureBaseURL is the base URL of a fixture client; nothing is ever sent to it.
const FixtureBaseURL = "http://ns-fixture.invalid"

// NewFixtureClient returns a client that answers from recordings made by cmd/nsrecord (an
// index.json plus one file per response) instead of calling NS. It needs no key. A request that
// was not recorded, path and query both, gets a 404, so the rest of the app behaves as it does
// for an unknown train. Used for local development and end-to-end tests (NS_MODE=fixture).
func NewFixtureClient(dir string) (*Client, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		return nil, fmt.Errorf("ns: fixtures: %w", err)
	}
	var idx struct {
		Entries []struct {
			Path   string `json:"path"`
			Query  string `json:"query"`
			Status int    `json:"status"`
			File   string `json:"file"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, fmt.Errorf("ns: fixtures: parse index: %w", err)
	}
	t := fixtureTransport{}
	for _, e := range idx.Entries {
		// Only plain file names: an index cannot point outside its folder.
		if e.File == "" || e.File != filepath.Base(e.File) || strings.HasPrefix(e.File, ".") {
			return nil, fmt.Errorf("ns: fixtures: bad file name %q", e.File)
		}
		body, err := os.ReadFile(filepath.Join(dir, e.File))
		if err != nil {
			return nil, fmt.Errorf("ns: fixtures: %w", err)
		}
		status := e.Status
		if status == 0 {
			status = http.StatusOK
		}
		t[fixtureKey(e.Path, e.Query)] = fixture{status: status, body: body}
	}
	return &Client{BaseURL: FixtureBaseURL, HTTP: &http.Client{Transport: t}}, nil
}

type fixture struct {
	status int
	body   []byte
}

type fixtureTransport map[string]fixture

func fixtureKey(path, query string) string { return path + "?" + query }

func (t fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f, ok := t[fixtureKey(r.URL.Path, r.URL.Query().Encode())]
	if !ok {
		f = fixture{status: http.StatusNotFound, body: []byte(`{"code":404,"message":"not recorded"}`)}
	}
	return &http.Response{
		StatusCode: f.status,
		Status:     http.StatusText(f.status),
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(f.body)),
		Request:    r,
	}, nil
}
