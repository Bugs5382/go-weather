// Package space carries geomagnetic activity.
//
// Here rather than in an astronomy library because it is the same kind of
// thing as weather and the opposite of astronomy: where the Sun *is* can be
// computed from a date and a coordinate, but what the Sun *did* two days ago
// can only be observed. One repository for the sky conditions that have to be
// measured, and a clean boundary against the ones that can be derived.
package space

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

// stormKp is the conventional threshold for a geomagnetic storm: NOAA's G1.
const stormKp = 5

// Activity is the planetary K index at an instant.
//
// Kp alone, and deliberately nothing derived from it. Whether an aurora is
// visible from anywhere in particular depends on the observer's geomagnetic
// latitude, which is the consumer's to know -- this package reports the
// number and does not guess at what anybody can see.
type Activity struct {
	// At is the instant the reading covers. Kp is published in three-hour
	// windows, so this is the start of one rather than a continuous sample.
	At time.Time
	// Kp is the planetary K index, 0 to 9.
	Kp float64
}

// Storm reports whether the activity is at or above the conventional storm
// threshold.
func (a Activity) Storm() bool { return a.Kp >= stormKp }
