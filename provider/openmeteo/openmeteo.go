// Package openmeteo fetches observations from Open-Meteo.
//
// An optional adapter: the core does not import it, so a consumer that
// already has an observation never pulls in an HTTP client. It lives out here
// rather than in the core because the core is pure and this is not.
//
// Open-Meteo needs no API key and answers for any coordinate on earth, which
// is why it is the first adapter -- a consumer drawing the sky over wherever
// its reader happens to be cannot use a provider that covers one country.
//
// The package does not cache, retry, rate-limit or schedule. Those are
// deployment decisions and they belong to whoever deploys.
package openmeteo

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
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	weather "github.com/Bugs5382/go-weather"
)

// DefaultBaseURL is Open-Meteo's forecast endpoint.
const DefaultBaseURL = "https://api.open-meteo.com/v1/forecast"

// currentFields are the variables asked for, named once so the request and
// the decode cannot drift apart.
const currentFields = "weather_code,cloud_cover,precipitation,snowfall," +
	"visibility,wind_speed_10m,wind_gusts_10m,wind_direction_10m"

// requestTimeout bounds a single fetch.
//
// A timeout rather than none: a consumer that calls this on a request path
// can otherwise be held open indefinitely by somebody else's outage.
const requestTimeout = 10 * time.Second

// Client fetches current conditions.
type Client struct {
	// HTTP is the client used for requests. Replaceable so a consumer can
	// supply its own transport, timeouts or instrumentation.
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

// response is Open-Meteo's shape, only as far as this needs it.
type response struct {
	Elevation float64 `json:"elevation"`
	Current   struct {
		Time string `json:"time"`
		// Interval is how often the provider updates this value, in seconds.
		// It is the provider stating its own cadence, which is what makes an
		// expiry reportable rather than invented.
		Interval      int     `json:"interval"`
		WeatherCode   int     `json:"weather_code"`
		CloudCover    float64 `json:"cloud_cover"`
		Precipitation float64 `json:"precipitation"`
		Snowfall      float64 `json:"snowfall"`
		Visibility    float64 `json:"visibility"`
		WindSpeed     float64 `json:"wind_speed_10m"`
		WindGusts     float64 `json:"wind_gusts_10m"`
		WindDirection float64 `json:"wind_direction_10m"`
	} `json:"current"`
}

// Decode turns a response body into an Observation.
//
// Exported so the mapping can be tested against a recorded response with
// nothing listening, which is the only honest way to check field names and
// units: a hand-written literal asserts only what the author already believed.
func Decode(body []byte) (weather.Observation, error) {
	var r response
	if err := json.Unmarshal(body, &r); err != nil {
		return weather.Observation{}, apperr.Coded(
			weather.CodeMalformedResponse,
			fmt.Errorf("%w: %v", weather.ErrMalformedResponse, err),
		)
	}

	q := weather.Quantities{
		// Open-Meteo reports cover as a percentage and the vocabulary is a
		// fraction. An off-by-one-hundred here is invisible everywhere else.
		CloudCover:             r.Current.CloudCover / 100,
		PrecipitationMMPerHour: r.Current.Precipitation,
		SnowfallCMPerHour:      r.Current.Snowfall,
		VisibilityMetres:       r.Current.Visibility,
	}

	condition, err := weather.ConditionFromWMO(r.Current.WeatherCode, q)
	if err != nil {
		return weather.Observation{}, err
	}

	// The request asks for no timezone, so Open-Meteo answers in GMT and
	// stamps current.time without an offset.
	at, err := time.Parse("2006-01-02T15:04", r.Current.Time)
	if err != nil {
		return weather.Observation{}, apperr.Coded(
			weather.CodeMalformedResponse,
			fmt.Errorf("%w: current.time %q: %v", weather.ErrMalformedResponse, r.Current.Time, err),
		)
	}
	at = at.UTC()

	// The provider's own cadence becomes the expiry. Where it states none,
	// none is claimed: Observation.Stale then reports never stale, because
	// choosing a TTL is the consumer's decision and not this library's.
	var expires time.Time
	if r.Current.Interval > 0 {
		expires = at.Add(time.Duration(r.Current.Interval) * time.Second)
	}

	return weather.Observation{
		At:         at,
		ExpiresAt:  expires,
		Condition:  condition,
		Quantities: q,
		Wind: weather.Wind{
			SpeedMPH:    r.Current.WindSpeed,
			GustMPH:     r.Current.WindGusts,
			FromDegrees: r.Current.WindDirection,
		},
		ElevationMetres: r.Elevation,
	}, nil
}

// Current fetches the conditions at a coordinate.
//
// The coordinate is validated first, so a caller's bad input costs no round
// trip. A transport failure and a non-success status are both
// CodeProviderUnavailable; a body that will not parse is
// CodeMalformedResponse. A consumer retries the first and not the second.
func (c *Client) Current(ctx context.Context, at weather.Coordinate) (weather.Observation, error) {
	if err := at.Validate(); err != nil {
		return weather.Observation{}, err
	}

	url := fmt.Sprintf("%s?latitude=%.4f&longitude=%.4f&current=%s&wind_speed_unit=mph",
		c.BaseURL, at.Lat, at.Lng, currentFields)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return weather.Observation{}, apperr.Coded(
			weather.CodeProviderUnavailable,
			fmt.Errorf("%w: %v", weather.ErrProviderUnavailable, err),
		)
	}

	res, err := c.HTTP.Do(req)
	if err != nil {
		return weather.Observation{}, apperr.Coded(
			weather.CodeProviderUnavailable,
			fmt.Errorf("%w: %v", weather.ErrProviderUnavailable, err),
		)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return weather.Observation{}, apperr.Coded(
			weather.CodeProviderUnavailable,
			fmt.Errorf("%w: status %d", weather.ErrProviderUnavailable, res.StatusCode),
		)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return weather.Observation{}, apperr.Coded(
			weather.CodeProviderUnavailable,
			fmt.Errorf("%w: %v", weather.ErrProviderUnavailable, err),
		)
	}
	return Decode(body)
}
