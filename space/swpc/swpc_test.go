package swpc_test

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
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Bugs5382/go-weather/space/swpc"
)

// Against a recorded feed. The shape was worth recording: the endpoint
// returns objects with a numeric Kp, not the array-of-arrays with a header
// row that several SWPC products use.
func TestDecodeRecordedFeed(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("testdata/kp.json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}

	got, err := swpc.Decode(body)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.At.IsZero() {
		t.Error("no instant")
	}
	if got.Kp < 0 || got.Kp > 9 {
		t.Errorf("Kp %v is outside 0..9", got.Kp)
	}
}

// The feed is a history and the caller wants now, so the last row wins. Taking
// the first would report the sky of several days ago, which would look
// entirely plausible and be wrong.
func TestLatestIsTheLastRow(t *testing.T) {
	t.Parallel()

	got, err := swpc.Decode([]byte(`[
      {"time_tag":"2026-09-18T15:00:00","Kp":1.00,"a_running":4,"station_count":8},
      {"time_tag":"2026-09-18T18:00:00","Kp":7.67,"a_running":9,"station_count":8}
    ]`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Kp != 7.67 {
		t.Errorf("Kp = %v, want the last row's 7.67", got.Kp)
	}
	if got.At.Hour() != 18 {
		t.Errorf("hour = %d, want 18", got.At.Hour())
	}
	if !got.Storm() {
		t.Error("Kp 7.67 is a storm")
	}
}

func TestEmptyFeedIsMalformed(t *testing.T) {
	t.Parallel()

	if _, err := swpc.Decode([]byte(`[]`)); err == nil {
		t.Error("an empty feed has no latest reading and should not answer zero")
	}
	if _, err := swpc.Decode([]byte(`not json`)); err == nil {
		t.Error("expected an error")
	}
}

func TestNonSuccessIsAnError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := swpc.New()
	c.BaseURL = srv.URL
	if _, err := c.Latest(context.Background()); err == nil {
		t.Fatal("expected an error")
	}
}
