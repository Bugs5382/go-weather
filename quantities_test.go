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
	"testing"

	weather "github.com/Bugs5382/go-weather"
)

// Each accessor must tell a measured zero from a missing reading, and must
// report a value as itself (issue #19).
func TestQuantityReadings(t *testing.T) {
	t.Parallel()

	type reading func(weather.Quantities) (float64, bool)

	cases := []struct {
		name    string
		read    reading
		set     func(*weather.Quantities, float64)
		missing func(*weather.Missing)
	}{
		{
			"cloud cover", weather.Quantities.CloudCoverReading,
			func(q *weather.Quantities, v float64) { q.CloudCover = v },
			func(m *weather.Missing) { m.CloudCover = true },
		},
		{
			"precipitation", weather.Quantities.PrecipitationReading,
			func(q *weather.Quantities, v float64) { q.PrecipitationMMPerHour = v },
			func(m *weather.Missing) { m.Precipitation = true },
		},
		{
			"snowfall", weather.Quantities.SnowfallReading,
			func(q *weather.Quantities, v float64) { q.SnowfallCMPerHour = v },
			func(m *weather.Missing) { m.Snowfall = true },
		},
		{
			"visibility", weather.Quantities.VisibilityReading,
			func(q *weather.Quantities, v float64) { q.VisibilityMetres = v },
			func(m *weather.Missing) { m.Visibility = true },
		},
	}

	for _, c := range cases {
		t.Run(c.name+" missing", func(t *testing.T) {
			t.Parallel()
			var q weather.Quantities
			c.missing(&q.Missing)
			if v, ok := c.read(q); ok || v != 0 {
				t.Errorf("missing %s = (%v, %v), want (0, false)", c.name, v, ok)
			}
		})

		t.Run(c.name+" zero", func(t *testing.T) {
			t.Parallel()
			var q weather.Quantities
			c.set(&q, 0)
			if v, ok := c.read(q); !ok || v != 0 {
				t.Errorf("measured zero %s = (%v, %v), want (0, true)", c.name, v, ok)
			}
		})

		t.Run(c.name+" present", func(t *testing.T) {
			t.Parallel()
			var q weather.Quantities
			c.set(&q, 0.5)
			if v, ok := c.read(q); !ok || v != 0.5 {
				t.Errorf("%s = (%v, %v), want (0.5, true)", c.name, v, ok)
			}
		})
	}
}

// A missing reading reports zero from its accessor even if something was left
// in the field beside the flag, so a caller that trusts the flag never draws
// a stale number.
func TestMissingReadingHidesTheField(t *testing.T) {
	t.Parallel()

	q := weather.Quantities{VisibilityMetres: 300, Missing: weather.Missing{Visibility: true}}
	if v, ok := q.VisibilityReading(); ok || v != 0 {
		t.Errorf("visibility = (%v, %v), want (0, false)", v, ok)
	}
}

// The zero Missing is "everything reported". That is what keeps a v1 literal
// such as Quantities{CloudCover: 0.9} meaning what it always meant.
func TestZeroMissingMeansReported(t *testing.T) {
	t.Parallel()

	q := weather.Quantities{CloudCover: 0.9}
	if _, ok := q.CloudCoverReading(); !ok {
		t.Error("a literal without Missing should read as reported")
	}
	if q.Missing.Any() {
		t.Error("the zero Missing should report nothing missing")
	}
	q.Missing.Snowfall = true
	if !q.Missing.Any() {
		t.Error("Any should see a missing snowfall")
	}
}
