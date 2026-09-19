// Package weather turns third-party weather observations into one stable
// vocabulary.
//
// It is the sibling of go-astronomy and differs from it in one way worth
// stating first: go-astronomy computes and go-weather observes. Where the Sun
// is follows from a date and a coordinate, with no truth outside the
// arithmetic. Whether it is raining is a measurement somebody took, so it can
// only come from a feed.
//
// That difference is the whole shape of the package. The core here is pure --
// the vocabulary, and the judgement that maps a provider's values onto it --
// and every network call lives in an optional adapter subpackage the core
// does not import. A consumer that already has an observation never touches
// the network at all, and the core's tests run with nothing listening.
//
// The package stores nothing and schedules nothing. Every observation carries
// the provider's own expiry where the provider states one, and what a
// consumer does with that is the consumer's business: a library that reaches
// for a cache has opinions about deployment that a library has no business
// having.
//
// It is stateless and concurrency-safe. time.Time is always a parameter and
// never captured at construction.
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
