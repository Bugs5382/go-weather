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

// Open-Meteo answers null for a quantity its model does not carry at a place,
// and a null decoded straight into a float64 is a zero. A missing visibility
// then reads as fog and a missing rain rate as dry, so each quantity is
// checked three ways: null must be missing, a measured zero must stay a
// reported zero, and a real value must come through as itself (issue #19).
func TestNullQuantityIsMissingNotZero(t *testing.T) {
	t.Parallel()

	type reading func(weather.Quantities) (float64, bool)

	fields := []struct {
		name    string
		json    string
		read    reading
		present string
		want    float64
	}{
		{"cloud cover", "cloud_cover", weather.Quantities.CloudCoverReading, "40", 0.4},
		{"precipitation", "precipitation", weather.Quantities.PrecipitationReading, "1.5", 1.5},
		{"snowfall", "snowfall", weather.Quantities.SnowfallReading, "0.7", 0.7},
		{"visibility", "visibility", weather.Quantities.VisibilityReading, "24140", 24140},
	}

	// body is a full response with one field overridden and the rest at
	// ordinary reported values, so a test of one field cannot pass on
	// another's behalf.
	body := func(field, value string) []byte {
		vals := map[string]string{
			"cloud_cover": "10", "precipitation": "0.2", "snowfall": "0.1", "visibility": "30000",
		}
		vals[field] = value
		return []byte(`{"elevation":10,"current":{"time":"2026-09-18T12:00","interval":900,` +
			`"weather_code":3,"cloud_cover":` + vals["cloud_cover"] +
			`,"precipitation":` + vals["precipitation"] +
			`,"snowfall":` + vals["snowfall"] +
			`,"visibility":` + vals["visibility"] +
			`,"wind_speed_10m":5,"wind_gusts_10m":9,"wind_direction_10m":270}}`)
	}

	for _, f := range fields {
		t.Run(f.name+" null", func(t *testing.T) {
			t.Parallel()
			got, err := openmeteo.Decode(body(f.json, "null"))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if v, ok := f.read(got.Quantities); ok {
				t.Errorf("null %s read as reported %v, want missing", f.name, v)
			}
		})

		t.Run(f.name+" zero", func(t *testing.T) {
			t.Parallel()
			got, err := openmeteo.Decode(body(f.json, "0"))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			v, ok := f.read(got.Quantities)
			if !ok {
				t.Fatalf("measured zero %s read as missing", f.name)
			}
			if v != 0 {
				t.Errorf("%s = %v, want 0", f.name, v)
			}
		})

		t.Run(f.name+" present", func(t *testing.T) {
			t.Parallel()
			got, err := openmeteo.Decode(body(f.json, f.present))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			v, ok := f.read(got.Quantities)
			if !ok {
				t.Fatalf("%s read as missing", f.name)
			}
			if v != f.want {
				t.Errorf("%s = %v, want %v", f.name, v, f.want)
			}
		})
	}
}

// A field left out of the response altogether is as missing as a null one.
// Open-Meteo drops nothing it was asked for today, but a decoder that only
// handled the explicit null would turn the day it does into zeros again.
func TestAbsentQuantityIsMissing(t *testing.T) {
	t.Parallel()

	got, err := openmeteo.Decode([]byte(`{"elevation":10,"current":{
      "time":"2026-09-18T12:00","interval":900,"weather_code":3,
      "wind_speed_10m":5,"wind_gusts_10m":9,"wind_direction_10m":270}}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := weather.Missing{CloudCover: true, Precipitation: true, Snowfall: true, Visibility: true}
	if got.Quantities.Missing != want {
		t.Errorf("missing = %+v, want %+v", got.Quantities.Missing, want)
	}
}

// A null visibility must not fog the sky, and a real low one still must: the
// override reads presence, not the zero a null used to become.
func TestNullVisibilityDoesNotFogTheSky(t *testing.T) {
	t.Parallel()

	got, err := openmeteo.Decode([]byte(`{"elevation":10,"current":{
      "time":"2026-09-18T12:00","interval":900,"weather_code":3,"cloud_cover":90,
      "precipitation":0,"snowfall":0,"visibility":null,
      "wind_speed_10m":5,"wind_gusts_10m":9,"wind_direction_10m":270}}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Condition != weather.Cloudy {
		t.Errorf("overcast with no visibility figure = %q, want CLOUDY", got.Condition)
	}

	got, err = openmeteo.Decode([]byte(`{"elevation":10,"current":{
      "time":"2026-09-18T12:00","interval":900,"weather_code":3,"cloud_cover":90,
      "precipitation":0,"snowfall":0,"visibility":300,
      "wind_speed_10m":5,"wind_gusts_10m":9,"wind_direction_10m":270}}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Condition != weather.Fog {
		t.Errorf("overcast at 300m = %q, want FOG", got.Condition)
	}
}
