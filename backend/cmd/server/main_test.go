package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestNSClientModes(t *testing.T) {
	c, err := nsClient(env(nil))
	if err != nil || c != nil {
		t.Errorf("live without a key: want no client and no error, got %v %v", c, err)
	}
	c, err = nsClient(env(map[string]string{"NS_API_KEY": "k"}))
	if err != nil || c == nil || c.BaseURL != ns.DefaultBaseURL {
		t.Errorf("live with a key: got %+v %v", c, err)
	}
	c, err = nsClient(env(map[string]string{"NS_MODE": "fixture", "NS_FIXTURES": filepath.Join("..", "..", "testdata", "ns")}))
	if err != nil || c == nil || c.BaseURL != ns.FixtureBaseURL {
		t.Errorf("fixture: got %+v %v", c, err)
	}
	if _, err := nsClient(env(map[string]string{"NS_MODE": "fixture", "NS_FIXTURES": t.TempDir()})); err == nil {
		t.Error("fixture without recordings: want an error")
	}
	if _, err := nsClient(env(map[string]string{"NS_MODE": "mock"})); err == nil {
		t.Error("unknown mode: want an error")
	}
}

func TestHealthcheck(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"ok":true,"vehicles":{"count":0,"stale":true}}`))
	}))
	defer ok.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(ok.URL, "http://"))
	c := &http.Client{Timeout: time.Second}
	// A stale feed is still healthy: liveness only.
	for _, addr := range []string{":" + port, "0.0.0.0:" + port, "127.0.0.1:" + port} {
		if err := healthcheck(addr, c); err != nil {
			t.Errorf("%s: %v", addr, err)
		}
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer bad.Close()
	_, badPort, _ := net.SplitHostPort(strings.TrimPrefix(bad.URL, "http://"))
	if err := healthcheck(":"+badPort, c); err == nil {
		t.Error("503: want unhealthy")
	}
	notOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"ok":false}`)) }))
	defer notOK.Close()
	_, notOKPort, _ := net.SplitHostPort(strings.TrimPrefix(notOK.URL, "http://"))
	if err := healthcheck(":"+notOKPort, c); err == nil {
		t.Error(`"ok":false: want unhealthy`)
	}
	if err := healthcheck("nonsense", c); err == nil {
		t.Error("bad address: want an error")
	}
	if err := healthcheck("127.0.0.1:1", c); err == nil {
		t.Error("nothing listening: want unhealthy")
	}
}

func TestListenAddr(t *testing.T) {
	if listenAddr(env(nil)) != ":8080" || listenAddr(env(map[string]string{"LISTEN_ADDR": ":9000"})) != ":9000" {
		t.Error("unexpected listen address")
	}
}
