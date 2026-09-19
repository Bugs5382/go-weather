package weather_test

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
	"errors"
	"testing"
	"time"

	weather "github.com/Bugs5382/go-weather"
)

// Eight values, and the two that are absent matter as much as the eight that
// are. A blizzard is a three-hour classification rather than something anybody
// can see happening, so it is not in the vocabulary and must not creep back in.
func TestConditionVocabulary(t *testing.T) {
	t.Parallel()

	for _, c := range []weather.Condition{
		weather.Clear, weather.Cloudy, weather.Fog, weather.Rain,
		weather.RainHeavy, weather.Snow, weather.SnowHeavy, weather.Thunderstorm,
	} {
		if !c.Valid() {
			t.Errorf("%q should be valid", c)
		}
	}

	for _, c := range []weather.Condition{"BLIZZARD", "WHITEOUT", "", "SLEET"} {
		if c.Valid() {
			t.Errorf("%q should not be valid", c)
		}
	}
}

func TestCoordinateValidation(t *testing.T) {
	t.Parallel()

	if err := (weather.Coordinate{Lat: 40.68, Lng: -73.94}).Validate(); err != nil {
		t.Errorf("Brooklyn should validate: %v", err)
	}

	for _, bad := range []weather.Coordinate{
		{Lat: 91, Lng: 0}, {Lat: -91, Lng: 0}, {Lat: 0, Lng: 181}, {Lat: 0, Lng: -181},
	} {
		if err := bad.Validate(); !errors.Is(err, weather.ErrInvalidCoordinate) {
			t.Errorf("%+v should be rejected, got %v", bad, err)
		}
	}
}

// The provider says when its answer goes stale; the library reports it and
// makes no decision of its own. An observation with no stated expiry is never
// stale, because claiming otherwise would be inventing a TTL.
func TestStaleFollowsTheProvidersExpiry(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	o := weather.Observation{At: at, ExpiresAt: at.Add(5 * time.Minute)}

	if o.Stale(at.Add(4 * time.Minute)) {
		t.Error("should be fresh before the stated expiry")
	}
	if !o.Stale(at.Add(6 * time.Minute)) {
		t.Error("should be stale after the stated expiry")
	}

	noExpiry := weather.Observation{At: at}
	if noExpiry.Stale(at.Add(100 * time.Hour)) {
		t.Error("an observation with no stated expiry must never be called stale")
	}
}
