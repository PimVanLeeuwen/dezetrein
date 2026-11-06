package api

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

// The recordings' train 2450 runs carriages 3304 and 3108; matchFeed puts it near Schiphol.

func postCarriage(t *testing.T, url, body string) *http.Response {
	t.Helper()
	res, err := http.Post(url+"/api/carriage", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func TestCarriageFromIndexAfterTripView(t *testing.T) {
	fake := newFakeNS(t)
	srv := newTestServer(t, &Server{NS: fake})
	get(t, srv.URL+"/api/trains/2450")
	res := postCarriage(t, srv.URL, `{"carriage":"3108"}`)
	body := decode[carriageResponse](t, res)
	if res.StatusCode != http.StatusOK || body.Train == nil || body.Train.Train != "2450" || body.Source != "index" || body.Located {
		t.Fatalf("status %d, %+v", res.StatusCode, body)
	}
	if body.Train.Destination != "Rotterdam Centraal" || body.Train.Category != "ICD" || body.Train.DistanceM != nil || len(body.Train.Carriages) != 2 {
		t.Errorf("unexpected train details: %+v", body.Train)
	}
	if body.AsOf.IsZero() {
		t.Error("want as_of for the index hit")
	}
}

func TestCarriageIndexFedByJourneyAlone(t *testing.T) {
	fake := newFakeNS(t)
	fake.setFail("composition", true)
	srv := newTestServer(t, &Server{NS: fake})
	get(t, srv.URL+"/api/trains/2450") // the journey lists the stock per stop
	if b := decode[carriageResponse](t, postCarriage(t, srv.URL, `{"carriage":"3304"}`)); b.Train == nil || b.Train.Train != "2450" {
		t.Errorf("want the carriage found through the journey's stock: %+v", b)
	}
}

func TestCarriageNearby(t *testing.T) {
	fake := newFakeNS(t)
	srv := newTestServer(t, &Server{NS: fake, Vehicles: matchFeed})
	body := `{"carriage":"3304","fixes":[` + fixJSON(52.3092, 4.7622, 1414) + `]}`
	b := decode[carriageResponse](t, postCarriage(t, srv.URL, body))
	if b.Train == nil || b.Train.Train != "2450" || b.Source != "nearby" || !b.Located || b.Searched != 2 || b.Train.DistanceM == nil {
		t.Fatalf("want 2450 found among the 2 nearby trains: %+v %+v", b, b.Train)
	}
	if !b.AsOf.Equal(testNow) {
		t.Errorf("as_of %v, want the vehicle snapshot's", b.AsOf)
	}
	calls := fake.count("composition")
	if b := decode[carriageResponse](t, postCarriage(t, srv.URL, body)); b.Source != "index" || fake.count("composition") != calls {
		t.Errorf("second lookup: want an index hit without NS calls, got %q and %d new calls", b.Source, fake.count("composition")-calls)
	}
}

func TestCarriageNotFound(t *testing.T) {
	fake := newFakeNS(t)
	srv := newTestServer(t, &Server{NS: fake, Vehicles: matchFeed})
	b := decode[carriageResponse](t, postCarriage(t, srv.URL, `{"carriage":"9999"}`))
	if b.Train != nil || b.Located || b.Searched != 0 || fake.count("composition") != 0 {
		t.Errorf("without location: want no train, located false and no NS calls: %+v, %d calls", b, fake.count("composition"))
	}
	b = decode[carriageResponse](t, postCarriage(t, srv.URL, `{"carriage":"9999","fixes":[`+fixJSON(52.3092, 4.7622, 30)+`]}`))
	if b.Train != nil || !b.Located || b.Searched != 2 {
		t.Errorf("with location: want no train after checking 2 nearby: %+v", b)
	}
}

func TestCarriageValidation(t *testing.T) {
	fake := newFakeNS(t)
	srv := newTestServer(t, &Server{NS: fake, Vehicles: matchFeed})
	ok := fixJSON(52.3, 4.76, 30)
	for name, body := range map[string]string{
		"letters":       `{"carriage":"94a2"}`,
		"empty":         `{"carriage":""}`,
		"too long":      `{"carriage":"1234567"}`,
		"number type":   `{"carriage":9402}`,
		"unknown field": `{"carriage":"9402","x":1}`,
		"six fixes":     `{"carriage":"9402","fixes":[` + strings.Repeat(ok+",", 5) + ok + `]}`,
		"bad fix":       `{"carriage":"9402","fixes":[` + fixJSON(40, -3, 30) + `]}`,
		"not json":      `carriage=9402`,
	} {
		if res := postCarriage(t, srv.URL, body); res.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, res.StatusCode)
		}
	}
	if n := fake.count("composition") + fake.count("journey"); n != 0 {
		t.Errorf("invalid requests reached NS %d times", n)
	}
	off := newTestServer(t, &Server{})
	if res := postCarriage(t, off.URL, `{"carriage":"9402"}`); res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("no NS key: status %d, want 503", res.StatusCode)
	}
}

func TestCarriageSharesLookupLimit(t *testing.T) {
	srv := newTestServer(t, &Server{NS: newFakeNS(t), Vehicles: matchFeed})
	for range matchBurst {
		postMatch(t, srv.URL, `{"fixes":[`+fixJSON(52.3092, 4.7622, 30)+`]}`)
	}
	if res := postCarriage(t, srv.URL, `{"carriage":"3304"}`); res.StatusCode != http.StatusTooManyRequests {
		t.Errorf("after the lookup burst on match: status %d, want 429", res.StatusCode)
	}
}

func TestCarriageNeverLogsLocation(t *testing.T) {
	var logs bytes.Buffer
	fake := newFakeNS(t)
	fake.setFail("composition", true)
	srv := newTestServer(t, &Server{NS: fake, Vehicles: matchFeed, Log: slog.New(slog.NewJSONHandler(&logs, nil))})
	postCarriage(t, srv.URL, `{"carriage":"3304","fixes":[`+fixJSON(52.309123, 4.762234, 1414)+`]}`)
	out := logs.String()
	if !strings.Contains(out, "ns fetch failed") {
		t.Fatalf("expected the NS failures to be logged, got %q", out)
	}
	for _, leak := range []string{"52.309", "4.762", "1414"} {
		if strings.Contains(out, leak) {
			t.Errorf("log contains %q from the request: %s", leak, out)
		}
	}
}

func TestCarriageIndexExpiresAndCaps(t *testing.T) {
	now := testNow
	x := &carriageIndex{now: func() time.Time { return now }}
	x.record("2450", []string{"3304", "3108"})
	x.record("", []string{"1"})
	x.record("1", nil)
	if train, _, ok := x.lookup("3108"); !ok || train != "2450" {
		t.Fatalf("lookup: %q %v", train, ok)
	}
	x.record("3522", []string{"3108"})
	if train, _, _ := x.lookup("3108"); train != "3522" {
		t.Errorf("a later sighting must win: %q", train)
	}
	now = now.Add(carriageTTL)
	if _, _, ok := x.lookup("3304"); ok {
		t.Error("an entry older than the TTL must not be used")
	}
	for i := range maxIndexed {
		x.record("t", []string{fmt.Sprint(100000 + i)})
	}
	if len(x.m) > maxIndexed {
		t.Errorf("index grew to %d, cap %d", len(x.m), maxIndexed)
	}
}
