// Package nws fetches official alerts from the United States National Weather
// Service.
//
// An optional adapter, like every other fetcher here: the alert vocabulary
// does not import it.
//
// One provider among several that publish CAP. NWS covers the United States
// and nothing else, and that limit is the caller's to handle -- a Report from
// this package says Covered because NWS answered, and a coordinate outside
// its area should not be asked in the first place. Silence from an
// unsupported region must never be presented as an all-clear.
package nws

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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	weather "github.com/Bugs5382/go-weather"
	"github.com/Bugs5382/go-weather/alert"
)

// DefaultBaseURL is the active-alerts endpoint.
const DefaultBaseURL = "https://api.weather.gov/alerts/active"

// userAgent identifies this library to NWS, which asks callers to say who
// they are and throttles those that do not.
const userAgent = "go-weather (github.com/Bugs5382/go-weather)"

// requestTimeout bounds a single fetch, so a consumer calling this on a
// request path cannot be held open by somebody else's outage.
const requestTimeout = 10 * time.Second

// Client fetches active alerts.
type Client struct {
	// HTTP is the client used for requests.
	HTTP *http.Client
	// BaseURL is the endpoint. Replaceable for testing.
	BaseURL string
}

// New returns a Client with a bounded timeout and the default endpoint.
func New() *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: requestTimeout},
		BaseURL: DefaultBaseURL,
	}
}

// response is the GeoJSON shape, only as far as this needs it.
type response struct {
	Features []struct {
		// ID is the canonical URL for this alert: the page the issuer
		// publishes, which is where a reader should be sent.
		ID         string `json:"id"`
		Properties struct {
			Event    string `json:"event"`
			Headline string `json:"headline"`
			Severity string `json:"severity"`
			// Status is CAP's Actual, Exercise, System, Test or Draft. Only
			// Actual is a real warning; see actual().
			Status     string `json:"status"`
			SenderName string `json:"senderName"`
			Expires    string `json:"expires"`
		} `json:"properties"`
	} `json:"features"`
}

// actual reports whether a CAP status describes a real warning.
//
// This filter is not hypothetical. NWS publishes test messages on the live
// feed: of 489 alerts active nationwide when this adapter was written, one
// was a Test, and it arrived with the same shape as a severe thunderstorm
// warning and no headline at all. Showing that to a reader as a warning is
// precisely the failure this whole package must not have.
//
// CAP's other statuses are Exercise, System and Draft. None of them is
// something a member of the public should be shown, so only Actual passes.
// An absent status is treated as Actual, because every real alert carries one
// and refusing those would drop genuine warnings over a missing field.
func actual(status string) bool {
	s := strings.ToUpper(strings.TrimSpace(status))
	return s == "" || s == "ACTUAL"
}

// severityOf maps NWS's title-case grade onto CAP's ladder.
//
// An unrecognised grade is Unknown rather than an error. CAP permits an
// ungraded alert, and an ungraded warning is still a warning -- refusing to
// parse one would drop it.
func severityOf(s string) alert.Severity {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "EXTREME":
		return alert.Extreme
	case "SEVERE":
		return alert.Severe
	case "MODERATE":
		return alert.Moderate
	case "MINOR":
		return alert.Minor
	default:
		return alert.Unknown
	}
}

// Decode turns a response body into a Report.
//
// Exported so the mapping can be tested against recorded feeds with nothing
// listening.
//
// Covered is set unconditionally: an answer from NWS is coverage, whatever it
// contains. That is the whole distinction the field exists for -- an empty
// feed means nothing is happening, and only a failed request means we do not
// know.
func Decode(body []byte) (alert.Report, error) {
	var r response
	if err := json.Unmarshal(body, &r); err != nil {
		return alert.Report{}, apperr.Coded(
			weather.CodeMalformedResponse,
			fmt.Errorf("%w: %v", weather.ErrMalformedResponse, err),
		)
	}

	out := alert.Report{Covered: true}
	for _, f := range r.Features {
		if !actual(f.Properties.Status) {
			continue
		}
		a := alert.Alert{
			Severity: severityOf(f.Properties.Severity),
			Event:    f.Properties.Event,
			Headline: f.Properties.Headline,
			URL:      f.ID,
			Issuer:   f.Properties.SenderName,
		}
		// An unparseable expiry costs the expiry, never the alert. Dropping a
		// tornado warning because its timestamp was odd is the worst failure
		// available here.
		if t, err := time.Parse(time.RFC3339, f.Properties.Expires); err == nil {
			a.Until = t
		}
		out.Alerts = append(out.Alerts, a)
	}
	return out, nil
}

// Active fetches the alerts in force at a coordinate.
//
// On any failure the returned Report is the zero value, whose Covered is
// false. That matters more than the error: a consumer that ignored the error
// and rendered the report would show "we do not know" rather than an
// all-clear it was never told.
func (c *Client) Active(ctx context.Context, at weather.Coordinate) (alert.Report, error) {
	if err := at.Validate(); err != nil {
		return alert.Report{}, err
	}

	url := fmt.Sprintf("%s?point=%.4f,%.4f", c.BaseURL, at.Lat, at.Lng)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return alert.Report{}, apperr.Coded(
			weather.CodeProviderUnavailable,
			fmt.Errorf("%w: %v", weather.ErrProviderUnavailable, err),
		)
	}
	req.Header.Set("Accept", "application/geo+json")
	req.Header.Set("User-Agent", userAgent)

	res, err := c.HTTP.Do(req)
	if err != nil {
		return alert.Report{}, apperr.Coded(
			weather.CodeProviderUnavailable,
			fmt.Errorf("%w: %v", weather.ErrProviderUnavailable, err),
		)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return alert.Report{}, apperr.Coded(
			weather.CodeProviderUnavailable,
			fmt.Errorf("%w: status %d", weather.ErrProviderUnavailable, res.StatusCode),
		)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return alert.Report{}, apperr.Coded(
			weather.CodeProviderUnavailable,
			fmt.Errorf("%w: %v", weather.ErrProviderUnavailable, err),
		)
	}
	return Decode(body)
}
