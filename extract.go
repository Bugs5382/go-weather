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

import apperr "github.com/Bugs5382/go-apperr"

// fogVisibilityMetres is the visibility below which the sky is fog whatever
// the code says.
//
// One kilometre, the conventional meteorological threshold for fog as against
// mist. The override exists because the code and the visibility come from
// different grids: a code can say overcast while the visibility figure says
// three hundred metres, and a person standing there is in fog.
const fogVisibilityMetres = 1000

// wmoConditions maps WMO 4677 interpretation codes onto the vocabulary.
//
// This table is the judgement of the library, and the reason the core is
// worth having at all: a consumer given raw codes would make these choices
// itself and the next consumer would make them differently.
//
// It covers the set Open-Meteo documents. Codes absent from it return an
// error rather than a guess, because providers add codes and answering
// "unknown" is honest where inventing an answer is not.
//
// Four judgements worth naming, because a switch statement hides them:
//
//   - Drizzle has no condition of its own. It is light rain, and a ninth
//     value would be a distinction nothing downstream could draw.
//   - "Moderate" sits with the light value rather than the heavy one. Heavy
//     is the exceptional case and should look like it; putting moderate there
//     would make most rainy days heavy.
//   - Freezing rain maps to its unfrozen equivalent. Freezing is a hazard
//     rather than a different sky, and this vocabulary describes how a sky
//     looks.
//   - Hail has no value of its own. A thunderstorm with hail is a
//     thunderstorm; the storm is the sky, and the hail is an event within it.
var wmoConditions = map[int]Condition{
	0: Clear, 1: Clear,
	2: Cloudy, 3: Cloudy,
	45: Fog, 48: Fog,
	51: Rain, 53: Rain, 55: Rain,
	56: Rain, 57: Rain,
	61: Rain, 63: Rain, 65: RainHeavy,
	66: Rain, 67: RainHeavy,
	71: Snow, 73: Snow, 75: SnowHeavy,
	77: Snow,
	80: Rain, 81: Rain, 82: RainHeavy,
	85: Snow, 86: SnowHeavy,
	95: Thunderstorm, 96: Thunderstorm, 99: Thunderstorm,
}

// precipitating reports whether a condition is falling weather, which the
// visibility override must never reach: heavy rain is heavy rain however
// little can be seen through it.
func precipitating(c Condition) bool {
	switch c {
	case Rain, RainHeavy, Snow, SnowHeavy, Thunderstorm:
		return true
	default:
		return false
	}
}

// ConditionFromWMO maps a provider's WMO interpretation code onto the
// vocabulary, letting visibility overrule it where the two disagree.
//
// Returns a CodeUnknownProviderCode error for a code with no mapping.
//
// A visibility flagged in Missing never overrules the code. A zero
// VisibilityMetres is also still read as "not reported" rather than as zero
// visibility, because that is what it meant before Missing existed: a
// Quantities built by hand without the flag would otherwise fog every sky
// whose author left the field out, which is the more damaging reading of an
// absent number.
func ConditionFromWMO(code int, q Quantities) (Condition, error) {
	c, ok := wmoConditions[code]
	if !ok {
		return "", apperr.Coded(CodeUnknownProviderCode, ErrUnknownProviderCode)
	}
	visibility, reported := q.VisibilityReading()
	if !precipitating(c) && reported && visibility > 0 && visibility < fogVisibilityMetres {
		return Fog, nil
	}
	return c, nil
}
