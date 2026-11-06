package ns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseRecordedComposition(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ns", "composition.json"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseComposition(body)
	if err != nil {
		t.Fatal(err)
	}
	want := &Composition{Train: "2450", Type: "ICNG", Carriages: []string{"3304", "3108"}, LengthM: 273}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("got %+v, want %+v", c, want)
	}
}

func TestParseCompositionEmpty(t *testing.T) {
	c, err := ParseComposition([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Carriages == nil || len(c.Carriages) != 0 {
		t.Errorf("want an empty, non-nil carriage list, got %#v", c.Carriages)
	}
}

func TestClientComposition(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write([]byte(`{"type":"VIRM","materieeldelen":[{"materieelnummer":9419},{"materieelnummer":"9402"}]}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Key: "k", HTTP: srv.Client()}
	comp, err := c.Composition(context.Background(), "3522")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/virtual-train-api/api/v1/trein/3522" || comp.Train != "3522" || !reflect.DeepEqual(comp.Carriages, []string{"9419", "9402"}) {
		t.Errorf("path %q, got %+v", path, comp)
	}
}
