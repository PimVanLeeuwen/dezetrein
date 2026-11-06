package main

import (
	"path/filepath"
	"testing"

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
