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
	"errors"
	"testing"

	apperr "github.com/Bugs5382/go-apperr"
	weather "github.com/Bugs5382/go-weather"
)

// The codes are a public contract, so their numbers are asserted rather than
// their presence. Renumbering one silently breaks every consumer branching on
// it, and that is exactly the change a test should refuse.
func TestErrorCodesAreStable(t *testing.T) {
	t.Parallel()

	for name, want := range map[string]int{
		"CodeInvalidCoordinate":   8001,
		"CodeUnknownProviderCode": 8002,
		"CodeProviderUnavailable": 8003,
		"CodeMalformedResponse":   8004,
	} {
		got := map[string]int{
			"CodeInvalidCoordinate":   weather.CodeInvalidCoordinate,
			"CodeUnknownProviderCode": weather.CodeUnknownProviderCode,
			"CodeProviderUnavailable": weather.CodeProviderUnavailable,
			"CodeMalformedResponse":   weather.CodeMalformedResponse,
		}[name]
		if got != want {
			t.Errorf("%s = %d, want %d", name, got, want)
		}
	}
}

// A wrapped error must carry its code back out, or a consumer has to match on
// error strings.
func TestCodeSurvivesWrapping(t *testing.T) {
	t.Parallel()

	err := apperr.Coded(weather.CodeInvalidCoordinate, weather.ErrInvalidCoordinate)
	code, ok := apperr.Code(err)
	if !ok {
		t.Fatal("a coded error must report a code")
	}
	if code != weather.CodeInvalidCoordinate {
		t.Errorf("code = %d, want %d", code, weather.CodeInvalidCoordinate)
	}

	// And the cause survives, so a caller can use errors.Is instead.
	if !errors.Is(err, weather.ErrInvalidCoordinate) {
		t.Error("the sentinel cause should survive wrapping")
	}
}
