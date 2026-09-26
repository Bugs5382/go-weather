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

// Each wind accessor must tell a measured zero from a missing reading, and
// must report a value as itself (issue #21). Zero is a real reading for all
// three: a calm, a still gust, a wind out of the north.
func TestWindReadings(t *testing.T) {
	t.Parallel()

	type reading func(weather.Wind) (float64, bool)

	cases := []struct {
		name    string
		read    reading
		set     func(*weather.Wind, float64)
		missing func(*weather.WindMissing)
	}{
		{
			"speed", weather.Wind.SpeedReading,
			func(w *weather.Wind, v float64) { w.SpeedMPH = v },
			func(m *weather.WindMissing) { m.Speed = true },
		},
		{
			"gust", weather.Wind.GustReading,
			func(w *weather.Wind, v float64) { w.GustMPH = v },
			func(m *weather.WindMissing) { m.Gust = true },
		},
		{
			"direction", weather.Wind.DirectionReading,
			func(w *weather.Wind, v float64) { w.FromDegrees = v },
			func(m *weather.WindMissing) { m.Direction = true },
		},
	}

	for _, c := range cases {
		t.Run(c.name+" missing", func(t *testing.T) {
			t.Parallel()
			var w weather.Wind
			c.missing(&w.Missing)
			if v, ok := c.read(w); ok || v != 0 {
				t.Errorf("missing %s = (%v, %v), want (0, false)", c.name, v, ok)
			}
		})

		t.Run(c.name+" zero", func(t *testing.T) {
			t.Parallel()
			var w weather.Wind
			c.set(&w, 0)
			if v, ok := c.read(w); !ok || v != 0 {
				t.Errorf("measured zero %s = (%v, %v), want (0, true)", c.name, v, ok)
			}
		})

		t.Run(c.name+" present", func(t *testing.T) {
			t.Parallel()
			var w weather.Wind
			c.set(&w, 270)
			if v, ok := c.read(w); !ok || v != 270 {
				t.Errorf("%s = (%v, %v), want (270, true)", c.name, v, ok)
			}
		})
	}
}

// A missing wind reading reports zero from its accessor even if something was
// left in the field beside the flag.
func TestMissingWindHidesTheField(t *testing.T) {
	t.Parallel()

	w := weather.Wind{GustMPH: 40, Missing: weather.WindMissing{Gust: true}}
	if v, ok := w.GustReading(); ok || v != 0 {
		t.Errorf("gust = (%v, %v), want (0, false)", v, ok)
	}
}

// The zero WindMissing is "everything reported", so a v1 literal such as
// Wind{SpeedMPH: 12} keeps meaning what it always meant.
func TestZeroWindMissingMeansReported(t *testing.T) {
	t.Parallel()

	w := weather.Wind{SpeedMPH: 12}
	if _, ok := w.SpeedReading(); !ok {
		t.Error("a literal without Missing should read as reported")
	}
	if w.Missing.Any() {
		t.Error("the zero WindMissing should report nothing missing")
	}
	w.Missing.Direction = true
	if !w.Missing.Any() {
		t.Error("Any should see a missing direction")
	}
}

// A condition the provider did not give is unknown: the accessor says so and
// hides whatever the field holds, and the zero flag keeps a v1 literal
// reported (issue #21).
func TestConditionReading(t *testing.T) {
	t.Parallel()

	missing := weather.Observation{Condition: weather.Clear, ConditionMissing: true}
	if c, ok := missing.ConditionReading(); ok || c != "" {
		t.Errorf("missing condition = (%q, %v), want (\"\", false)", c, ok)
	}

	// Clear is WMO code 0, the condition a null code used to become. Set on
	// purpose it is still a reading.
	clear := weather.Observation{Condition: weather.Clear}
	if c, ok := clear.ConditionReading(); !ok || c != weather.Clear {
		t.Errorf("clear = (%q, %v), want (CLEAR, true)", c, ok)
	}

	rain := weather.Observation{Condition: weather.Rain}
	if c, ok := rain.ConditionReading(); !ok || c != weather.Rain {
		t.Errorf("rain = (%q, %v), want (RAIN, true)", c, ok)
	}
}
