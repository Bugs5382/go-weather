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

	weather "github.com/Bugs5382/go-weather"
)

// The mapping is the library's judgement, so every debatable case is written
// down with its reasoning rather than left to the reader of a switch.
func TestConditionFromWMO(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		code int
		want weather.Condition
		why  string
	}{
		{"clear sky", 0, weather.Clear, "WMO 0 is clear"},
		{"mainly clear", 1, weather.Clear, "a few clouds is still a clear sky to look at"},
		{"partly cloudy", 2, weather.Cloudy, "visibly cloud"},
		{"overcast", 3, weather.Cloudy, ""},
		{"fog", 45, weather.Fog, ""},
		{"depositing rime fog", 48, weather.Fog, "still fog to look at"},

		{"light drizzle", 51, weather.Rain, "drizzle has no value of its own; it is light rain"},
		{"dense drizzle", 55, weather.Rain, "dense drizzle is still not heavy rain by rate"},
		{"freezing drizzle", 56, weather.Rain, "freezing is a hazard, not a different sky"},

		{"slight rain", 61, weather.Rain, ""},
		{"moderate rain", 63, weather.Rain, "moderate sits with rain; heavy is the exception"},
		{"heavy rain", 65, weather.RainHeavy, ""},
		{"light freezing rain", 66, weather.Rain, ""},
		{"heavy freezing rain", 67, weather.RainHeavy, ""},

		{"slight snow", 71, weather.Snow, ""},
		{"moderate snow", 73, weather.Snow, ""},
		{"heavy snow", 75, weather.SnowHeavy, ""},
		{"snow grains", 77, weather.Snow, "grains are light snow"},

		{"slight rain showers", 80, weather.Rain, ""},
		{"moderate rain showers", 81, weather.Rain, ""},
		{"violent rain showers", 82, weather.RainHeavy, ""},
		{"slight snow showers", 85, weather.Snow, ""},
		{"heavy snow showers", 86, weather.SnowHeavy, ""},

		{"thunderstorm", 95, weather.Thunderstorm, ""},
		{"thunderstorm with slight hail", 96, weather.Thunderstorm, "hail has no value; the storm is the sky"},
		{"thunderstorm with heavy hail", 99, weather.Thunderstorm, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := weather.ConditionFromWMO(c.code, weather.Quantities{})
			if err != nil {
				t.Fatalf("code %d: unexpected error %v", c.code, err)
			}
			if got != c.want {
				t.Errorf("code %d = %q, want %q (%s)", c.code, got, c.want, c.why)
			}
		})
	}
}

// Every mapped code must land on a value the vocabulary admits. A typo in the
// table would otherwise produce a condition nothing downstream can render.
func TestEveryMappedCodeIsValid(t *testing.T) {
	t.Parallel()

	for code := 0; code <= 99; code++ {
		got, err := weather.ConditionFromWMO(code, weather.Quantities{})
		if err != nil {
			continue // unmapped, which is a tested behaviour of its own
		}
		if !got.Valid() {
			t.Errorf("code %d maps to %q, which is not in the vocabulary", code, got)
		}
	}
}

// Providers add codes. Answering "unknown" is honest where guessing is not,
// and a caller can decide whether that is fatal.
func TestUnmappedCodeIsAnError(t *testing.T) {
	t.Parallel()

	if _, err := weather.ConditionFromWMO(4, weather.Quantities{}); !errors.Is(err, weather.ErrUnknownProviderCode) {
		t.Errorf("code 4 should be unmapped, got %v", err)
	}
}

// Fog is the one condition where a quantity overrules the code, because the
// two come from different grids: a code can say overcast while the visibility
// figure says three hundred metres, and a person standing there is in fog.
func TestLowVisibilityBecomesFog(t *testing.T) {
	t.Parallel()

	got, err := weather.ConditionFromWMO(3, weather.Quantities{VisibilityMetres: 300})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if got != weather.Fog {
		t.Errorf("overcast at 300m = %q, want FOG", got)
	}

	// Falling weather is never overruled, however little can be seen through
	// it: heavy rain is heavy rain, not fog.
	got, err = weather.ConditionFromWMO(65, weather.Quantities{VisibilityMetres: 300})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if got != weather.RainHeavy {
		t.Errorf("heavy rain at 300m = %q, want RAIN_HEAVY", got)
	}

	// And an absent visibility figure must not read as zero and fog the sky.
	got, err = weather.ConditionFromWMO(0, weather.Quantities{VisibilityMetres: 0})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if got != weather.Clear {
		t.Errorf("clear with no visibility figure = %q, want CLEAR", got)
	}
}
