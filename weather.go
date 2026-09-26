package weather

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
	"time"

	apperr "github.com/Bugs5382/go-apperr"
)

// Condition is what the sky is doing, as one of eight values.
//
// The grade is part of the condition rather than a separate number, because
// the providers grade it: WMO codes 61, 63 and 65 are slight, moderate and
// heavy rain, which is why a phone says "light rain" rather than "rain". Rain
// and RainHeavy are different facts, not different drawings of one fact.
//
// There is deliberately no blizzard and no whiteout. The published definition
// of a blizzard requires the conditions to hold for three hours, which makes
// it a diagnosis applied in retrospect rather than something anybody can see
// happening -- and heavy snow with a high wind, which is what a person
// standing outside actually sees, is already expressible as SnowHeavy with a
// Wind beside it.
type Condition string

// The eight conditions.
const (
	Clear        Condition = "CLEAR"
	Cloudy       Condition = "CLOUDY"
	Fog          Condition = "FOG"
	Rain         Condition = "RAIN"
	RainHeavy    Condition = "RAIN_HEAVY"
	Snow         Condition = "SNOW"
	SnowHeavy    Condition = "SNOW_HEAVY"
	Thunderstorm Condition = "THUNDERSTORM"
)

// Valid reports whether c is one of the eight conditions.
func (c Condition) Valid() bool {
	switch c {
	case Clear, Cloudy, Fog, Rain, RainHeavy, Snow, SnowHeavy, Thunderstorm:
		return true
	default:
		return false
	}
}

// Wind is how hard it is blowing and which way from.
//
// Beside the condition and never inside it: any condition can be calm or
// blowing hard. Clear covers a still morning and a gale under a blue sky, and
// deriving a nominal speed per condition would collapse that distinction
// permanently -- every consumer would grow its own table and disagree with
// the next one.
type Wind struct {
	// SpeedMPH is the sustained speed. Zero is calm.
	SpeedMPH float64
	// GustMPH is the peak gust, which can far exceed the sustained speed and
	// is what actually moves a branch.
	GustMPH float64
	// FromDegrees is the bearing the wind blows *from*, the meteorological
	// convention: 270 comes out of the west and pushes towards the east, so
	// something leaning in it leans away from this bearing.
	FromDegrees float64

	// Missing names the wind readings the provider did not give. A missing
	// reading leaves its field at zero, and zero is a real reading for each:
	// a calm, a still air between gusts, a wind out of the north. Read the
	// field only where Missing says it was reported, or use the Reading
	// accessors, which answer both at once.
	//
	// Its zero value is "everything reported", so a Wind written without it
	// means what it meant before the flag existed.
	Missing WindMissing
}

// WindMissing records, per wind reading, that the provider gave no value.
//
// A flag for absence rather than presence, for the reason Missing gives: only
// that way round is the zero value right.
type WindMissing struct {
	Speed     bool
	Gust      bool
	Direction bool
}

// Any reports whether any wind reading is missing.
func (m WindMissing) Any() bool {
	return m.Speed || m.Gust || m.Direction
}

// SpeedReading returns the sustained speed and whether it was reported. A
// missing reading returns (0, false), whatever the field holds.
func (w Wind) SpeedReading() (float64, bool) {
	return reading(w.SpeedMPH, w.Missing.Speed)
}

// GustReading returns the peak gust and whether it was reported. A missing
// reading returns (0, false), whatever the field holds.
func (w Wind) GustReading() (float64, bool) {
	return reading(w.GustMPH, w.Missing.Gust)
}

// DirectionReading returns the bearing the wind blows from and whether it was
// reported. A missing reading returns (0, false), whatever the field holds.
func (w Wind) DirectionReading() (float64, bool) {
	return reading(w.FromDegrees, w.Missing.Direction)
}

// Quantities are the continuous values a renderer actually draws from.
//
// Beside the condition rather than derived from it, because eight discrete
// states can only be cut between and numbers can be interpolated. That is the
// difference between weather that changes and weather that jumps, and it is
// what lets a consumer stay a renderer: it is handed values to draw rather
// than a category to interpret.
type Quantities struct {
	// CloudCover is 0 to 1, not a percentage.
	CloudCover float64
	// PrecipitationMMPerHour is liquid equivalent, so snow appears here too.
	PrecipitationMMPerHour float64
	// SnowfallCMPerHour is depth rather than liquid equivalent. Both are
	// carried because their ratio is how dry powder is told from wet snow,
	// and the same wind moves the two completely differently.
	SnowfallCMPerHour float64
	// VisibilityMetres is how far can be seen.
	VisibilityMetres float64

	// Missing names the quantities the provider did not report. A missing
	// quantity leaves its field at zero, and zero is a real reading for every
	// one of them: a clear sky, a dry hour, fog. Read the field only where
	// Missing says it was reported, or use the Reading accessors, which
	// answer both at once.
	//
	// Its zero value is "everything reported", so a Quantities written
	// without it means what it meant before the flag existed.
	Missing Missing
}

// Missing records, per quantity, that the provider gave no reading.
//
// A flag for absence rather than one for presence, because only that way
// round is the zero value right: an adapter must say what it did not get, and
// a caller that builds a Quantities by hand gets reported values without
// having to say so.
type Missing struct {
	CloudCover    bool
	Precipitation bool
	Snowfall      bool
	Visibility    bool
}

// Any reports whether any quantity is missing.
func (m Missing) Any() bool {
	return m.CloudCover || m.Precipitation || m.Snowfall || m.Visibility
}

// CloudCoverReading returns the cloud cover and whether it was reported. A
// missing reading returns (0, false), whatever the field holds.
func (q Quantities) CloudCoverReading() (float64, bool) {
	return reading(q.CloudCover, q.Missing.CloudCover)
}

// PrecipitationReading returns the precipitation rate and whether it was
// reported. A missing reading returns (0, false), whatever the field holds.
func (q Quantities) PrecipitationReading() (float64, bool) {
	return reading(q.PrecipitationMMPerHour, q.Missing.Precipitation)
}

// SnowfallReading returns the snowfall rate and whether it was reported. A
// missing reading returns (0, false), whatever the field holds.
func (q Quantities) SnowfallReading() (float64, bool) {
	return reading(q.SnowfallCMPerHour, q.Missing.Snowfall)
}

// VisibilityReading returns the visibility and whether it was reported. A
// missing reading returns (0, false), whatever the field holds.
func (q Quantities) VisibilityReading() (float64, bool) {
	return reading(q.VisibilityMetres, q.Missing.Visibility)
}

func reading(v float64, missing bool) (float64, bool) {
	if missing {
		return 0, false
	}
	return v, true
}

// Observation is one answer about one place at one instant.
type Observation struct {
	// At is the instant the observation describes.
	At time.Time
	// ExpiresAt is when the provider says its answer goes stale. Zero means
	// the provider did not say, and this library will not invent a value on
	// its behalf.
	ExpiresAt time.Time
	// Condition is the headline: one of eight, or empty where
	// ConditionMissing says the provider gave none.
	Condition Condition
	// ConditionMissing reports that the provider gave no condition, so the
	// sky is unknown. The adapter then leaves Condition empty, which is not
	// Valid, rather than letting it fall to whatever a zero code maps to: WMO
	// code 0 is a clear sky, and "no answer" is not "clear".
	//
	// Its zero value is "reported", so an Observation written without it
	// means what it meant before the flag existed.
	ConditionMissing bool
	// Quantities are what a renderer draws from.
	Quantities Quantities
	// Wind is beside the condition, never inside it.
	Wind Wind
	// ElevationMetres is the ground height the provider answered for. Carried
	// because it arrives free with the observation and because an astronomy
	// consumer wants it: the horizon dips with height, which moves sunrise by
	// minutes. This library does nothing with it.
	ElevationMetres float64
}

// ConditionReading returns the condition and whether it was reported. A
// missing condition returns ("", false), whatever the field holds.
func (o Observation) ConditionReading() (Condition, bool) {
	if o.ConditionMissing {
		return "", false
	}
	return o.Condition, true
}

// Stale reports whether the provider's own expiry has passed.
//
// An observation whose provider stated no expiry is never stale. Answering
// otherwise would mean choosing a TTL, and that is the consumer's decision:
// it knows how often it polls and how wrong it can afford to be.
func (o Observation) Stale(now time.Time) bool {
	if o.ExpiresAt.IsZero() {
		return false
	}
	return now.After(o.ExpiresAt)
}

// Coordinate is a place to ask about.
type Coordinate struct {
	// Lat is degrees north, in [-90, 90].
	Lat float64
	// Lng is degrees east, in [-180, 180].
	Lng float64
}

// Validate reports whether the coordinate is on the globe.
//
// Checked before any request is made, because a bad coordinate is the
// caller's bug and should not cost a round trip to discover.
func (c Coordinate) Validate() error {
	if c.Lat < -90 || c.Lat > 90 || c.Lng < -180 || c.Lng > 180 {
		return apperr.Coded(CodeInvalidCoordinate, ErrInvalidCoordinate)
	}
	return nil
}
