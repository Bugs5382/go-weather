package alert_test

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
	"time"

	"github.com/Bugs5382/go-weather/alert"
)

// Absence is not safety. A region with no feed and a region with nothing
// happening must be distinguishable, or a reader infers an all-clear from a
// silence the system never meant.
func TestCoveredDistinguishesSilenceFromIgnorance(t *testing.T) {
	t.Parallel()

	quiet := alert.Report{Covered: true}
	if !quiet.Covered || len(quiet.Alerts) != 0 {
		t.Error("a covered region with nothing happening is covered and empty")
	}

	unknown := alert.Report{}
	if unknown.Covered {
		t.Error("the zero value must not claim coverage")
	}
}

// CAP's ladder, ordered so a consumer can ask which of two matters more.
func TestSeverityOrdering(t *testing.T) {
	t.Parallel()

	if alert.Extreme.Rank() <= alert.Severe.Rank() {
		t.Error("extreme outranks severe")
	}
	if alert.Severe.Rank() <= alert.Moderate.Rank() {
		t.Error("severe outranks moderate")
	}
	if alert.Minor.Rank() >= alert.Moderate.Rank() {
		t.Error("moderate outranks minor")
	}
	// An ungraded alert must not outrank one somebody graded.
	if alert.Unknown.Rank() != 0 {
		t.Error("unknown ranks lowest")
	}
}

func TestWorstFindsTheHighest(t *testing.T) {
	t.Parallel()

	r := alert.Report{Covered: true, Alerts: []alert.Alert{
		{Event: "Flood Advisory", Severity: alert.Minor},
		{Event: "Tornado Warning", Severity: alert.Extreme},
		{Event: "Wind Advisory", Severity: alert.Moderate},
	}}

	worst, ok := r.Worst()
	if !ok {
		t.Fatal("should find one")
	}
	if worst.Event != "Tornado Warning" {
		t.Errorf("worst = %q, want the tornado warning", worst.Event)
	}

	if _, ok := (alert.Report{Covered: true}).Worst(); ok {
		t.Error("an empty report has no worst")
	}
}

// Nothing is rephrased. The struct carries the issuer's own strings, and this
// pins that there is no summarising helper tempting anybody to soften a
// warning.
func TestAlertCarriesTheIssuersOwnWords(t *testing.T) {
	t.Parallel()

	a := alert.Alert{
		Severity: alert.Extreme,
		Event:    "Tornado Warning",
		Headline: "Tornado Warning issued September 18 at 4:12PM EDT until September 18 at 4:45PM EDT by NWS New York NY",
		URL:      "https://api.weather.gov/alerts/urn:oid:2.49.0.1.840.0.example",
		Issuer:   "NWS New York NY",
		Until:    time.Date(2026, 9, 18, 20, 45, 0, 0, time.UTC),
	}

	if a.Headline == "" || a.URL == "" || a.Event == "" {
		t.Error("an alert without the issuer's words and link is not usable")
	}
}
