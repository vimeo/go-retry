//   Copyright 2020 Vimeo
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

// Package retry implements a state struct and methods providing exponential
// backoff as well as a wrapper function that does exponential backoff.
package retry

import (
	"log"
	"math"
	"math/rand"
	"time"
)

// Backoff contains the state implementing a generator returning a sequence of
// intervals to wait.
type Backoff struct {
	step int
	// If MaxBackoff == MinBackoff the backoff is constant.
	// If MinBackoff > MaxBackoff, the implementation may generate a runtime panic.
	MaxBackoff time.Duration
	MinBackoff time.Duration
	// Jitter is the maximum value that may be added or substracted based on
	// the output of a prng.
	// Jitter should be < 1, and may produce /interesting/ results if it is > 1 or < 0.
	Jitter float64
	// ExpFactor should be > 1, otherwise it will converge to 0.
	ExpFactor float64
}

// DefaultBackoff returns a resonable default backoff instance.
func DefaultBackoff() Backoff {
	return Backoff{
		MaxBackoff: time.Minute,
		MinBackoff: time.Millisecond,
		Jitter:     .1,
		ExpFactor:  1.2,
	}
}

// Clone returns a cloned copy of a Backoff struct.
func (b Backoff) Clone() Backoff {
	return Backoff{
		step:       b.step,
		MaxBackoff: b.MaxBackoff,
		MinBackoff: b.MinBackoff,
		Jitter:     b.Jitter,
		ExpFactor:  b.ExpFactor,
	}
}

// Get a random float64 between -b.Jitter and +b.Jitter.
func (b Backoff) jitter() float64 {

	// jitter is a random value on [0.0, 1.0), so subtract 0.5 and multiply by
	// 2 to move it to the interval [-1.0, 1.0), which is more suitable for
	// jitter, as we want equal probabilities on either side of the basic
	// exponential backoff.
	return b.Jitter * (rand.Float64() - 0.5) * 2
}

// When the exponential backoff has hit its cap, we need to jitter down, rather
// than both high and low.
func (b Backoff) jitterLow() float64 {
	return -b.Jitter * rand.Float64()
}

// When the exponential backoff has hit its lower bound, we need to jitter up,
// rather than both high and low.
func (b Backoff) jitterHigh() float64 {
	return b.Jitter * rand.Float64()
}

// Reset resets the step-count on its receiver. It is *not* thread-safe.
func (b *Backoff) Reset() {
	b.step = 0
}

// BackoffN is a stateless method that uses the parameters in the receiver to
// return a backoff interval appropriate for the Nth retry.
func (b *Backoff) BackoffN(n int) time.Duration {
	if b.MinBackoff > b.MaxBackoff {
		log.Panicf("MinBackoff (%s) > MaxBackoff(%s)",
			b.MinBackoff, b.MaxBackoff)
	}

	backoff := b.MinBackoff
	// This involves some casting about to get the duration into a float64
	// and back to do the necessary multiplication.
	// this would otherwise be:
	// backoff *= math.Pow(expFactor, n)
	expMul := math.Pow(b.ExpFactor, float64(n))
	backoffNS := math.Min(
		math.Max(float64(backoff.Nanoseconds())*expMul, 0),
		float64(b.MaxBackoff.Nanoseconds()))
	backoff = time.Duration(backoffNS) * time.Nanosecond

	jitter := b.jitter()
	if backoff >= b.MaxBackoff {
		backoff = b.MaxBackoff
		jitter = b.jitterLow()
	} else if backoff <= b.MinBackoff {
		backoff = b.MinBackoff
		jitter = b.jitterHigh()
	}

	// Increase (or decrease) backoff by a factor of jitter.
	// e.g. if jitter == 0.2, and backoff is 100 seconds, backoff becomes
	// 120 seconds.
	backoff += time.Duration(
		jitter*float64(backoff.Nanoseconds())) * time.Nanosecond

	if backoff > b.MaxBackoff {
		return b.MaxBackoff
	} else if backoff < b.MinBackoff {
		return b.MinBackoff
	}
	return backoff
}

// Next returns the next time interval to wait in the sequence.
func (b *Backoff) Next() time.Duration {
	backoff := b.BackoffN(b.step)
	b.step++
	return backoff
}
