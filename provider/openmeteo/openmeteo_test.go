package openmeteo_test

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
	"github.com/Bugs5382/go-weather/provider/openmeteo"
)

// Against a recorded response, which is the only way a vocabulary mapping can
// be checked at all: the assertion is about this provider's field names and
// units, and a hand-written literal would only assert what we already believe.
func TestDecodeRecordedResponse(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("testdata/brooklyn.json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}

	got, err := openmeteo.Decode(body)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !got.Condition.Valid() {
		t.Errorf("condition %q is not in the vocabulary", got.Condition)
	}
	if got.At.IsZero() {
		t.Error("observation has no instant")
	}
	// Elevation rides along with the forecast, which is why nothing has to
	// ask for it separately.
	if got.ElevationMetres == 0 {
		t.Error("elevation should be carried through from the response")
	}
	if got.Quantities.CloudCover < 0 || got.Quantities.CloudCover > 1 {
		t.Errorf("cloud cover %v should be a fraction", got.Quantities.CloudCover)
	}
	if got.Wind.FromDegrees < 0 || got.Wind.FromDegrees > 360 {
		t.Errorf("wind bearing %v out of range", got.Wind.FromDegrees)
	}
	// The provider states its own update cadence, so the expiry is reported
	// rather than invented.
	if !got.ExpiresAt.After(got.At) {
		t.Error("expiry should follow the instant, from the provider's interval")
	}
}

// Cloud cover arrives as a percentage and the vocabulary is a fraction. An
// off-by-one-hundred here is invisible in every other test.
func TestCloudCoverIsAFraction(t *testing.T) {
	t.Parallel()

	got, err := openmeteo.Decode([]byte(`{
      "elevation": 10.0,
      "current": {"time":"2026-09-18T12:00","interval":900,"weather_code":3,
        "cloud_cover":75,"precipitation":0,"snowfall":0,"visibility":24140,
        "wind_speed_10m":5,"wind_gusts_10m":9,"wind_direction_10m":270}
    }`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Quantities.CloudCover != 0.75 {
		t.Errorf("cloud cover = %v, want 0.75", got.Quantities.CloudCover)
	}
}

// The provider's interval becomes the expiry. An absent interval leaves it
// zero rather than inventing one, which Observation.Stale then reads as
// "never stale" -- the library will not choose a TTL.
func TestExpiryComesFromTheProvidersInterval(t *testing.T) {
	t.Parallel()

	with, err := openmeteo.Decode([]byte(`{"elevation":10,"current":{
      "time":"2026-09-18T12:00","interval":900,"weather_code":0,"cloud_cover":0,
      "precipitation":0,"snowfall":0,"visibility":30000,
      "wind_speed_10m":0,"wind_gusts_10m":0,"wind_direction_10m":0}}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := with.ExpiresAt.Sub(with.At); got.Minutes() != 15 {
		t.Errorf("expiry gap = %v, want 15m", got)
	}

	without, err := openmeteo.Decode([]byte(`{"elevation":10,"current":{
      "time":"2026-09-18T12:00","weather_code":0,"cloud_cover":0,
      "precipitation":0,"snowfall":0,"visibility":30000,
      "wind_speed_10m":0,"wind_gusts_10m":0,"wind_direction_10m":0}}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !without.ExpiresAt.IsZero() {
		t.Error("no stated interval must leave the expiry unset, not guessed")
	}
}

func TestMalformedResponseIsCoded(t *testing.T) {
	t.Parallel()

	if _, err := openmeteo.Decode([]byte(`not json`)); err == nil {
		t.Fatal("expected an error")
	}
}

// A non-200 is a provider problem and must be distinguishable from a parse
// problem, because a consumer retries one and not the other.
func TestNonSuccessIsProviderUnavailable(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := openmeteo.New()
	c.BaseURL = srv.URL
	if _, err := c.Current(context.Background(), weather.Coordinate{Lat: 40.68, Lng: -73.94}); err == nil {
		t.Fatal("expected an error")
	}
}

// The coordinate is validated before any request: a bad coordinate is the
// caller's bug and should not cost a round trip to discover.
func TestBadCoordinateIsRejectedWithoutARequest(t *testing.T) {
	t.Parallel()

	reached := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	}))
	defer srv.Close()

	c := openmeteo.New()
	c.BaseURL = srv.URL
	if _, err := c.Current(context.Background(), weather.Coordinate{Lat: 91}); err == nil {
		t.Fatal("expected an error")
	}
	if reached {
		t.Error("a bad coordinate should not reach the provider")
	}
}
