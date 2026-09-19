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
	"context"
	"errors"
	"fmt"
	"sync"

	apperr "github.com/Bugs5382/go-apperr"
	log "github.com/Bugs5382/go-log"
)

// ErrorServiceDigit is the leading digit that every go-weather error code
// shares, so a code is attributable to this module at a glance. It is the
// go-apperr service prefix for the module's code namespace. go-astronomy owns
// 7; this module owns 8.
const ErrorServiceDigit = 8

// Stable numeric error codes for the module. Each error-returning function
// wraps its cause with one of these via go-apperr, so a caller can recover the
// code with apperr.Code and branch on it without matching error strings. The
// values are part of the public contract: never renumber a code, only add new
// ones.
const (
	// CodeInvalidCoordinate marks a latitude outside [-90, 90] or a longitude
	// outside [-180, 180].
	CodeInvalidCoordinate = 8001
	// CodeUnknownProviderCode marks a provider vocabulary value this library
	// has no mapping for. It is a real condition rather than a bug: providers
	// add codes, and answering "unknown" is honest where guessing is not.
	CodeUnknownProviderCode = 8002
	// CodeProviderUnavailable marks a provider that could not be reached or
	// that answered with a non-success status.
	CodeProviderUnavailable = 8003
	// CodeMalformedResponse marks a provider response that parsed as transport
	// but not as the shape this library expects.
	CodeMalformedResponse = 8004
)

// Sentinel causes, so a caller can use errors.Is as well as read the code.
var (
	// ErrInvalidCoordinate is the cause behind CodeInvalidCoordinate.
	ErrInvalidCoordinate = errors.New("weather: coordinate out of range")
	// ErrUnknownProviderCode is the cause behind CodeUnknownProviderCode.
	ErrUnknownProviderCode = errors.New("weather: unmapped provider code")
	// ErrProviderUnavailable is the cause behind CodeProviderUnavailable.
	ErrProviderUnavailable = errors.New("weather: provider unavailable")
	// ErrMalformedResponse is the cause behind CodeMalformedResponse.
	ErrMalformedResponse = errors.New("weather: malformed provider response")
)

var errorEntries = []apperr.Entry{
	{Code: CodeInvalidCoordinate, Title: "coordinate", Cause: "latitude or longitude out of range"},
	{Code: CodeUnknownProviderCode, Title: "vocabulary", Cause: "provider code has no mapping"},
	{Code: CodeProviderUnavailable, Title: "provider", Cause: "provider unreachable or refused"},
	{Code: CodeMalformedResponse, Title: "provider", Cause: "response did not match the expected shape"},
}

// logSink adapts a go-log Logger to the go-apperr Logger interface. It is the
// pluggable logging sink go-apperr calls from Registry.PresentContext; the
// library never invokes that path itself, so it stays silent by default. When a
// consumer does present a coded error with a context, the sink derives a logger
// correlated with any OpenTelemetry span already on the context. It never starts
// a tracer or exporter: OpenTelemetry is a transitive dependency of go-log that
// remains dormant unless the surrounding service turns it on.
type logSink struct{ l log.Logger }

// LogCoded emits one structured error line for the code, using go-log's
// context-aware logger so an active trace is correlated when the service has set
// one up.
func (s logSink) LogCoded(ctx context.Context, code int, err error) {
	s.l.Ctx(ctx).Error(err, "coded error", log.F("code", code))
}

// errorRegistry builds the module's go-apperr registry exactly once, on first
// use. Constructing the go-log logger lazily keeps a plain import of this
// package free of any logging setup, and building it here rather than at init
// time keeps that cost off programs that never present a coded error.
var errorRegistry = sync.OnceValue(func() *apperr.Registry {
	reg, err := apperr.NewRegistry(
		errorEntries,
		apperr.WithService(ErrorServiceDigit),
		apperr.WithLogger(logSink{l: log.NewLogger("go-weather")}),
	)
	if err != nil {
		// The entries and service prefix are compile-time constants, so a
		// failure here is a programming error in this file, not a runtime input.
		panic(fmt.Sprintf("weather: building error registry: %v", err))
	}
	return reg
})

// Errors returns the module's go-apperr registry: the stable code table plus the
// presentation and logging policy around it. A consumer renders a coded error at
// its edge with Registry.Present (or PresentContext, which also drives the wired
// go-log logger) to obtain a sanitized, quotable message, looks a code up with
// Registry.Describe, or emits the whole table as Markdown with Registry.Markdown.
// The registry is built once and is safe for concurrent use.
func Errors() *apperr.Registry { return errorRegistry() }
