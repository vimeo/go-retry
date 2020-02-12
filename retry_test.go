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
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryCancel(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := context.WithCancel(context.Background())
	c := make(chan bool)
	backoff := DefaultBackoff
	backoff.MinBackoff = time.Microsecond

	go func() {
		err := Retry(ctx, backoff, 18, func(ctx context.Context) error {
			c <- true
			return fmt.Errorf("foo")
		})
		assert.Regexp(t, regexp.MustCompile("context expired while retrying: context canceled. retried \\d times"), err)
		close(c)
	}()
	<-c
	cancelFunc()
	calls := 1
	for range c {
		calls++
	}
	if calls > 8 {
		t.Errorf("Too many retries: %d", calls)
	}
}

func TestRetry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	c := make(chan bool)
	backoff := DefaultBackoff
	backoff.MinBackoff = time.Microsecond

	go func() {
		q := 0
		err := Retry(ctx, backoff, 18, func(ctx context.Context) error {
			q++
			if q == 2 {
				return nil
			}
			return fmt.Errorf("foo")
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, q)
		close(c)
	}()
	<-c
	for range c {
	}
}

func TestRetryUntilExhausted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	c := make(chan bool)
	backoff := DefaultBackoff
	backoff.MinBackoff = time.Microsecond

	go func() {
		q := 0
		err := Retry(ctx, backoff, 8, func(ctx context.Context) error {
			q++
			return fmt.Errorf("foo")
		})
		assert.EqualError(t, err, "aborting retry. errors: [foo foo foo foo foo foo foo foo]")
		assert.Equal(t, 8, q)
		close(c)
	}()
	<-c
	for range c {
	}
}
