package api

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/PimVanLeeuwen/dezetrein/backend/internal/ns"
)

// Train 2450 (in the recordings) near Schiphol, another train 150 m away and one far off.
var matchFeed = fakeFeed{AsOf: testNow, Vehicles: []ns.Vehicle{
	{RitID: "2450", Lat: 52.3091, Lng: 4.7622, Type: "ICD"},
	{RitID: "4000", Lat: 52.3105, Lng: 4.7622, Type: "SPR"},
	{RitID: "800", Lat: 51.4400, Lng: 5.4800, Type: "IC"},
}}

func postMatch(t *testing.T, url, body string) *http.Response {
	t.Helper()
	res, err := http.Post(url+"/api/match", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func fixJSON(lat, lng, acc float64) string {
	return fmt.Sprintf(`{"lat":%v,"lng":%v,"accuracy":%v,"speed":null,"heading":null,"t":%d}`, lat, lng, acc, testNow.UnixMilli())
}

func TestMatchReturnsDescribedCandidates(t *testing.T) {
	srv := newTestServer(t, &Server{NS: newFakeNS(t), Vehicles: matchFeed})
	res := postMatch(t, srv.URL, `{"fixes":[`+fixJSON(52.3092, 4.7622, 1414)+`]}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	body := decode[matchResponse](t, res)
	if len(body.Candidates) != 2 || body.Confidence != "low" || body.Match != nil || !body.AsOf.Equal(testNow) || body.Stale {
		t.Fatalf("unexpected: %+v", body)
	}
	first, second := body.Candidates[0], body.Candidates[1]
	if first.Train != "2450" || first.Destination != "Rotterdam Centraal" || len(first.Carriages) == 0 || first.Category != "ICD" || first.DistanceM == nil || *first.DistanceM > 50 {
		t.Errorf("unexpected first candidate: %+v", first)
	}
	if second.Train != "4000" || second.Destination != "" || second.Carriages == nil || len(second.Carriages) != 0 {
		t.Errorf("unknown train: want empty details, got %+v", second)
	}
}

func TestMatchHighConfidenceSetsMatch(t *testing.T) {
	srv := newTestServer(t, &Server{NS: newFakeNS(t), Vehicles: matchFeed})
	body := decode[matchResponse](t, postMatch(t, srv.URL, `{"fixes":[`+fixJSON(51.4401, 5.4800, 15)+`]}`))
	if body.Confidence != "high" || body.Match == nil || *body.Match != "800" {
		t.Errorf("unexpected: %+v", body)
	}
}

func TestMatchValidation(t *testing.T) {
	fake := newFakeNS(t)
	srv := newTestServer(t, &Server{NS: fake, Vehicles: matchFeed})
	ok := fixJSON(52.3, 4.76, 30)
	old := fmt.Sprintf(`{"lat":52.3,"lng":4.76,"accuracy":30,"t":%d}`, testNow.Add(-11*60e9).UnixMilli())
	ahead := fmt.Sprintf(`{"lat":52.3,"lng":4.76,"accuracy":30,"t":%d}`, testNow.Add(2*60e9).UnixMilli())
	for name, body := range map[string]string{
		"not json":         `lat=52`,
		"empty":            ``,
		"no fixes":         `{"fixes":[]}`,
		"six fixes":        `{"fixes":[` + strings.Repeat(ok+",", 5) + ok + `]}`,
		"unknown field":    `{"fixes":[` + ok + `],"name":"x"}`,
		"trailing data":    `{"fixes":[` + ok + `]} {}`,
		"missing lat":      `{"fixes":[{"lng":4.76,"accuracy":30,"t":1}]}`,
		"outside network":  `{"fixes":[` + fixJSON(40.4, -3.7, 30) + `]}`,
		"zero accuracy":    `{"fixes":[` + fixJSON(52.3, 4.76, 0) + `]}`,
		"huge accuracy":    `{"fixes":[` + fixJSON(52.3, 4.76, 50000) + `]}`,
		"negative speed":   `{"fixes":[{"lat":52.3,"lng":4.76,"accuracy":30,"speed":-1,"t":` + fmt.Sprint(testNow.UnixMilli()) + `}]}`,
		"heading over 360": `{"fixes":[{"lat":52.3,"lng":4.76,"accuracy":30,"heading":400,"t":` + fmt.Sprint(testNow.UnixMilli()) + `}]}`,
		"missing time":     `{"fixes":[{"lat":52.3,"lng":4.76,"accuracy":30}]}`,
		"too old":          `{"fixes":[` + old + `]}`,
		"in the future":    `{"fixes":[` + ahead + `]}`,
	} {
		if res := postMatch(t, srv.URL, body); res.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, res.StatusCode)
		}
	}
	big := `{"fixes":[` + ok + `],"pad":"` + strings.Repeat("x", maxMatchBody) + `"}`
	if res := postMatch(t, srv.URL, big); res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body: status %d, want 413", res.StatusCode)
	}
	if n := fake.count("journey") + fake.count("composition"); n != 0 {
		t.Errorf("invalid requests reached NS %d times", n)
	}
	res, err := http.Get(srv.URL + "/api/match")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	// The /api/ catch-all answers any other method, so a GET gets the JSON 404.
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("GET /api/match: status %d, want 404", res.StatusCode)
	}
}

func TestMatchWithoutVehicleData(t *testing.T) {
	body := `{"fixes":[` + fixJSON(52.3, 4.76, 30) + `]}`
	for name, s := range map[string]*Server{
		"no NS key":   {},
		"no poll yet": {Vehicles: fakeFeed{Stale: true}},
	} {
		srv := newTestServer(t, s)
		if res := postMatch(t, srv.URL, body); res.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("%s: status %d, want 503", name, res.StatusCode)
		}
	}
	stale := fakeFeed{AsOf: testNow.Add(-5 * 60e9), Stale: true, Vehicles: matchFeed.Vehicles}
	srv := newTestServer(t, &Server{NS: newFakeNS(t), Vehicles: stale})
	if b := decode[matchResponse](t, postMatch(t, srv.URL, body)); !b.Stale || len(b.Candidates) == 0 {
		t.Errorf("old vehicle data: want candidates marked stale, got %+v", b)
	}
}

// TestMatchNeverLogsLocation runs a match while every NS call fails, so the handler logs, and
// checks that no coordinate from the request reaches the log.
func TestMatchNeverLogsLocation(t *testing.T) {
	var logs bytes.Buffer
	fake := newFakeNS(t)
	fake.setFail("journey", true)
	fake.setFail("composition", true)
	srv := newTestServer(t, &Server{NS: fake, Vehicles: matchFeed, Log: slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))})
	res := postMatch(t, srv.URL, `{"fixes":[`+fixJSON(52.309123, 4.762234, 1414)+`]}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
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
