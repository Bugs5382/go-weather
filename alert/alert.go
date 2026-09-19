// Package alert carries official weather warnings, unedited.
//
// Deliberately not part of the weather vocabulary, and the separation is the
// point. An alert is something a public authority has *said*; a condition is
// something the sky is *doing*. Keeping them apart means a severity can never
// end up driving a sprite, and a tornado warning under a clear sky is drawn
// as a clear sky with a warning beside it rather than as bad weather.
//
// Nothing here is rephrased or summarised. We do not have the standing to
// reword a tornado warning, and a paraphrase is how a warning gets softened
// by accident. The issuer's own event name, headline and link travel through
// untouched.
package alert

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

import "time"

// Severity is CAP's own ladder.
type Severity string

// The CAP severities.
const (
	Extreme  Severity = "EXTREME"
	Severe   Severity = "SEVERE"
	Moderate Severity = "MODERATE"
	Minor    Severity = "MINOR"
	// Unknown is CAP's own value for an alert whose issuer did not grade it.
	// It is a real answer rather than a parse failure.
	Unknown Severity = "UNKNOWN"
)

// Rank orders severities so a consumer can ask which of two matters more.
//
// Unknown ranks lowest rather than highest: an ungraded alert should not
// outrank one somebody took the trouble to grade.
func (s Severity) Rank() int {
	switch s {
	case Extreme:
		return 4
	case Severe:
		return 3
	case Moderate:
		return 2
	case Minor:
		return 1
	default:
		return 0
	}
}

// Alert is one warning, in the issuer's own words.
type Alert struct {
	Severity Severity
	// Event is the issuer's own name for it: "Tornado Warning".
	Event string
	// Headline is the issuer's own summary, unedited.
	Headline string
	// URL is the official page for this alert, so a reader is sent to the
	// people who issued it rather than to a guess at their local station.
	URL string
	// Issuer is the office that put it out: "NWS New York NY".
	Issuer string
	// Until is when it expires, or zero where the issuer did not say.
	Until time.Time
}

// Report is the answer for one place.
//
// Covered says whether this location has an alert feed at all, and it is the
// most important field here. Absence is not safety: a region with no feed and
// a region with nothing happening are different answers, and a consumer that
// cannot tell them apart will show an all-clear it was never told. The zero
// value is therefore "we do not know", not "all clear".
type Report struct {
	Covered bool
	Alerts  []Alert
}

// Worst returns the highest-ranked alert, and false when there are none.
func (r Report) Worst() (Alert, bool) {
	var worst Alert
	found := false
	for _, a := range r.Alerts {
		if !found || a.Severity.Rank() > worst.Severity.Rank() {
			worst, found = a, true
		}
	}
	return worst, found
}
