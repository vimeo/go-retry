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

package retry

import (
	"math"
	"testing"
	"time"
)

func TestBackoffNext(t *testing.T) {
	b := Backoff{
		MaxBackoff: time.Minute,
		MinBackoff: time.Second,
		Jitter:     0.1,
		ExpFactor:  1.2,
	}

	dOld := time.Duration(1) * time.Second
	unjitteredBaseNS := float64(b.MinBackoff.Nanoseconds())

	for i := 0; i < 1000; i++ {
		d := b.Next()
		if d == dOld {
			// Randomized jitter seems to have a problem.
			t.Errorf("d == dOld, (%s). Jitter should prevent this. (i=%d)", d, i)
		}
		dMax := time.Duration(unjitteredBaseNS*(1.+b.Jitter)) * time.Nanosecond
		dMin := time.Duration(unjitteredBaseNS*(1.-b.Jitter)) * time.Nanosecond
		if dMax > b.MaxBackoff {
			dMax = b.MaxBackoff
		}
		if dMin > b.MinBackoff {
			dMin = b.MinBackoff
		}
		if d > dMax {
			t.Errorf("b.next() = %s, which is greater than expected: %s", d, dMax)
		}
		if d < dMin {
			t.Errorf("b.next() = %s, which is less than expected: %s", d, dMin)
		}
		if d > b.MaxBackoff {
			t.Errorf("b.next() = %s, which is greater than maximum: %s", d, b.MaxBackoff)
		}
		if d < b.MinBackoff {
			t.Errorf("b.next() = %s, which is less than maximum: %s", d, b.MinBackoff)
		}
		if i > 17 && d < time.Duration(20)*time.Second {
			t.Errorf("b.next() = %s, which is less than 20s after 18 iterations (i = %d)", d, i)
		}
		dOld = d
		unjitteredBaseNS *= b.ExpFactor
		if unjitteredBaseNS > float64(b.MaxBackoff.Nanoseconds()) {
			unjitteredBaseNS = float64(b.MaxBackoff.Nanoseconds())
		}
	}

}

func TestBackoffN(t *testing.T) {
	b := Backoff{
		MaxBackoff: time.Minute,
		MinBackoff: time.Second,
		Jitter:     0.1,
		ExpFactor:  1.2,
	}

	backoff0 := b.BackoffN(0)
	if backoff0 < b.MinBackoff {
		t.Errorf("initial backoff too small: %s vs min backoff: %s", backoff0, b.MinBackoff)
	}

	jitterMaxInit := time.Duration(float64(b.MinBackoff.Nanoseconds())*(1+b.Jitter)) * time.Nanosecond
	if backoff0 > jitterMaxInit {
		t.Errorf("initial backoff too large: %s vs min backoff * jitter: %s", backoff0, jitterMaxInit)
	}

	factor10 := math.Pow(b.ExpFactor, 10.)
	backoffBase10NS := factor10 * float64(b.MinBackoff.Nanoseconds())
	jitterMin := time.Duration(backoffBase10NS*(1-b.Jitter)) * time.Nanosecond
	jitterMax := time.Duration(backoffBase10NS*(1+b.Jitter)) * time.Nanosecond

	backoff10 := b.BackoffN(10)
	if backoff10 < b.MinBackoff {
		t.Errorf("10th backoff too small: %s vs min backoff: %s", backoff10, b.MinBackoff)
	}
	if backoff10 < jitterMin {
		t.Errorf("10th backoff too small: %s vs min with jitter: %s", backoff10, jitterMin)
	}
	if backoff10 > jitterMax {
		t.Errorf("10th backoff too large: %s vs max with jitter: %s", backoff10, jitterMax)
	}
}
