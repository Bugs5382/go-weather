// Package swpc fetches geomagnetic activity from NOAA's Space Weather
// Prediction Center.
//
// An optional adapter, like every other fetcher here: the space vocabulary
// does not import it.
package swpc

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
	"github.com/Bugs5382/go-weather/space"
)

// DefaultBaseURL is the planetary K index product.
const DefaultBaseURL = "https://services.swpc.noaa.gov/products/noaa-planetary-k-index.json"

// requestTimeout bounds a single fetch.
const requestTimeout = 10 * time.Second

// timeLayout is SWPC's stamp: no zone, and UTC by convention.
const timeLayout = "2006-01-02T15:04:05"

// Client fetches the planetary K index.
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

// reading is one row of the product.
//
// Objects with a numeric Kp, which is worth noting because several other SWPC
// products are arrays of arrays whose first row is a header of column names.
// This one is not, and decoding it as though it were would silently treat a
// real reading as a header and drop it.
type reading struct {
	TimeTag string  `json:"time_tag"`
	Kp      float64 `json:"Kp"`
}

// Decode turns a response body into the latest Activity.
//
// The feed is a history, several days of three-hourly readings, and a caller
// wants now -- so the last row wins. Taking the first would report the sky of
// several days ago, which would look entirely plausible and be wrong.
func Decode(body []byte) (space.Activity, error) {
	var rows []reading
	if err := json.Unmarshal(body, &rows); err != nil {
		return space.Activity{}, apperr.Coded(
			weather.CodeMalformedResponse,
			fmt.Errorf("%w: %v", weather.ErrMalformedResponse, err),
		)
	}
	if len(rows) == 0 {
		return space.Activity{}, apperr.Coded(
			weather.CodeMalformedResponse,
			fmt.Errorf("%w: no readings", weather.ErrMalformedResponse),
		)
	}

	last := rows[len(rows)-1]
	at, err := time.Parse(timeLayout, last.TimeTag)
	if err != nil {
		return space.Activity{}, apperr.Coded(
			weather.CodeMalformedResponse,
			fmt.Errorf("%w: time_tag %q: %v", weather.ErrMalformedResponse, last.TimeTag, err),
		)
	}

	return space.Activity{At: at.UTC(), Kp: last.Kp}, nil
}

// Latest fetches the most recent reading.
//
// It takes no coordinate: Kp is planetary, one number for the whole Earth.
// What it means for any particular observer is the consumer's to work out
// from their geomagnetic latitude.
func (c *Client) Latest(ctx context.Context) (space.Activity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL, nil)
	if err != nil {
		return space.Activity{}, apperr.Coded(
			weather.CodeProviderUnavailable,
			fmt.Errorf("%w: %v", weather.ErrProviderUnavailable, err),
		)
	}

	res, err := c.HTTP.Do(req)
	if err != nil {
		return space.Activity{}, apperr.Coded(
			weather.CodeProviderUnavailable,
			fmt.Errorf("%w: %v", weather.ErrProviderUnavailable, err),
		)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return space.Activity{}, apperr.Coded(
			weather.CodeProviderUnavailable,
			fmt.Errorf("%w: status %d", weather.ErrProviderUnavailable, res.StatusCode),
		)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return space.Activity{}, apperr.Coded(
			weather.CodeProviderUnavailable,
			fmt.Errorf("%w: %v", weather.ErrProviderUnavailable, err),
		)
	}
	return Decode(body)
}
