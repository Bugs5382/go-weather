package nws_test

/*
MIT License

Copyright (c) 2026 Shane

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	weather "github.com/Bugs5382/go-weather"
	"github.com/Bugs5382/go-weather/alert"
	"github.com/Bugs5382/go-weather/alert/nws"
)

// An empty feed is coverage with nothing happening, and must never read as no
// coverage. This is the case the whole Covered flag exists for, and the
// fixture is a real answer for Brooklyn on a quiet evening.
func TestEmptyFeedIsCoveredAndQuiet(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("testdata/quiet.json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}

	got, err := nws.Decode(body)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Covered {
		t.Error("an answer from NWS is coverage, whatever it contains")
	}
	if len(got.Alerts) != 0 {
		t.Errorf("quiet feed should hold no alerts, got %d", len(got.Alerts))
	}
	if _, ok := got.Worst(); ok {
		t.Error("a quiet report has no worst alert")
	}
}

// Against real recorded alerts, spanning the severities NWS was actually
// issuing. A hand-written fixture would assert only what the author believed
// the field names to be.
func TestAlertsCarryTheIssuersOwnWords(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("testdata/busy.json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}

	got, err := nws.Decode(body)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Covered {
		t.Fatal("should be covered")
	}
	if len(got.Alerts) == 0 {
		t.Fatal("the busy fixture should hold alerts")
	}

	for _, a := range got.Alerts {
		if a.Event == "" {
			t.Error("every alert carries the issuer's own event name")
		}
		if a.Headline == "" {
			t.Error("every alert carries the issuer's own headline")
		}
		if a.URL == "" {
			t.Error("every alert links to the official page for it")
		}
		if a.Issuer == "" {
			t.Error("every alert names the office that issued it")
		}
		if a.Until.IsZero() {
			t.Errorf("%q has no expiry", a.Event)
		}
	}

	// The recorded feed contains a live NWS test message. It must not be
	// here: a reader shown a "Test Message" as a warning is the failure this
	// package exists to avoid.
	for _, a := range got.Alerts {
		if a.Event == "Test Message" {
			t.Error("a CAP Test status reached the report")
		}
	}

	worst, ok := got.Worst()
	if !ok {
		t.Fatal("a populated report has a worst")
	}
	for _, a := range got.Alerts {
		if a.Severity.Rank() > worst.Severity.Rank() {
			t.Errorf("%q outranks the reported worst %q", a.Event, worst.Event)
		}
	}
}

// Only CAP's Actual reaches a reader. Exercise, System, Test and Draft are
// all real values on the live feed and none of them is a warning.
func TestOnlyActualAlertsAreReported(t *testing.T) {
	t.Parallel()

	got, err := nws.Decode([]byte(`{"features":[
      {"id":"https://example/1","properties":{"event":"Real Warning",
       "headline":"h","severity":"Extreme","status":"Actual",
       "senderName":"NWS Test","expires":"2026-09-18T20:45:00-04:00"}},
      {"id":"https://example/2","properties":{"event":"Test Message",
       "severity":"Unknown","status":"Test","senderName":"NWS Test",
       "expires":"2026-09-18T20:45:00-04:00"}},
      {"id":"https://example/3","properties":{"event":"Drill",
       "headline":"h","severity":"Severe","status":"Exercise",
       "senderName":"NWS Test","expires":"2026-09-18T20:45:00-04:00"}},
      {"id":"https://example/4","properties":{"event":"No Status",
       "headline":"h","severity":"Minor","senderName":"NWS Test",
       "expires":"2026-09-18T20:45:00-04:00"}}
    ]}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	var events []string
	for _, a := range got.Alerts {
		events = append(events, a.Event)
	}
	// The real one, and the one with no status -- every genuine alert carries
	// one, so an absent field must not drop a warning.
	if len(events) != 2 {
		t.Fatalf("reported %v, want the real warning and the one with no status", events)
	}
	for _, e := range events {
		if e == "Test Message" || e == "Drill" {
			t.Errorf("%q should have been filtered", e)
		}
	}
}

// NWS grades in title case; the vocabulary is upper. An unrecognised grade is
// Unknown rather than an error: CAP allows it and an ungraded warning is
// still a warning.
func TestSeverityMapping(t *testing.T) {
	t.Parallel()

	got, err := nws.Decode([]byte(`{"features":[
      {"id":"https://example/1","properties":{"event":"Tornado Warning",
       "headline":"h","severity":"Extreme","senderName":"NWS Test",
       "expires":"2026-09-18T20:45:00-04:00"}},
      {"id":"https://example/2","properties":{"event":"Odd Warning",
       "headline":"h","severity":"Fictional","senderName":"NWS Test",
       "expires":"2026-09-18T20:45:00-04:00"}}
    ]}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Alerts[0].Severity != alert.Extreme {
		t.Errorf("Extreme = %q", got.Alerts[0].Severity)
	}
	if got.Alerts[1].Severity != alert.Unknown {
		t.Errorf("an ungraded alert should be Unknown, got %q", got.Alerts[1].Severity)
	}
}

// An unparseable expiry loses the expiry, not the alert. Dropping a tornado
// warning because its timestamp was odd is the worst possible failure here.
func TestBadExpiryKeepsTheAlert(t *testing.T) {
	t.Parallel()

	got, err := nws.Decode([]byte(`{"features":[
      {"id":"https://example/1","properties":{"event":"Tornado Warning",
       "headline":"h","severity":"Extreme","senderName":"NWS Test",
       "expires":"not a time"}}
    ]}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Alerts) != 1 {
		t.Fatalf("the alert should survive, got %d", len(got.Alerts))
	}
	if !got.Alerts[0].Until.IsZero() {
		t.Error("an unparseable expiry should be left unset")
	}
}

// A non-200 is not "no alerts". It must not come back as a quiet report,
// because a consumer would render an all-clear from an outage.
func TestNonSuccessIsNotCoverage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := nws.New()
	c.BaseURL = srv.URL
	got, err := c.Active(context.Background(), weather.Coordinate{Lat: 40.68, Lng: -73.94})
	if err == nil {
		t.Fatal("expected an error")
	}
	if got.Covered {
		t.Error("a failed request must never report coverage")
	}
}

// A point outside the United States is not a provider failure. NWS answers a
// foreign coordinate with 400 and "out of bounds", which is a complete and
// correct answer to the question asked: nobody is watching there.
//
// Reporting it as an error would be wrong twice. It would collapse "we do not
// know" into "nothing is happening" at the consumer, which is the one
// distinction Covered exists to preserve, and it would log an error for every
// reader outside the US on every poll for ever.
func TestOutOfBoundsIsNotCoverageAndNotAnError(t *testing.T) {
	t.Parallel()

	// The live body, recorded from api.weather.gov for 48.85,2.35.
	const body = `{
		"correlationId": "2710a4",
		"title": "Invalid Parameter",
		"type": "https://api.weather.gov/problems/InvalidParameter",
		"status": 400,
		"detail": "Parameter \"point\" is invalid: out of bounds"
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := nws.New()
	c.BaseURL = srv.URL
	got, err := c.Active(context.Background(), weather.Coordinate{Lat: 48.85, Lng: 2.35})
	if err != nil {
		t.Fatalf("out of bounds should be an answer, not an error: %v", err)
	}
	if got.Covered {
		t.Error("a place outside the feed must not report coverage")
	}
	if len(got.Alerts) != 0 {
		t.Error("no alerts where there is no feed")
	}
}

// A 400 that is not about the point staying an error, so a genuine bad
// request is not quietly read as a foreign coordinate.
func TestOtherBadRequestsStayErrors(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"title":"Invalid Parameter","detail":"Parameter \"limit\" is invalid"}`))
	}))
	defer srv.Close()

	c := nws.New()
	c.BaseURL = srv.URL
	got, err := c.Active(context.Background(), weather.Coordinate{Lat: 40.68, Lng: -73.94})
	if err == nil {
		t.Fatal("a bad request that is not about the point is still a failure")
	}
	if got.Covered {
		t.Error("a failed request must never report coverage")
	}
}

// NWS asks callers to identify themselves, and a library that does not is a
// library that gets blocked.
func TestSendsAnIdentifyingUserAgent(t *testing.T) {
	t.Parallel()

	seen := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{"features":[]}`))
	}))
	defer srv.Close()

	c := nws.New()
	c.BaseURL = srv.URL
	if _, err := c.Active(context.Background(), weather.Coordinate{Lat: 40.68, Lng: -73.94}); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if seen == "" {
		t.Error("no User-Agent sent")
	}
}
