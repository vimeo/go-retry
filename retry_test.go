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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vimeo/go-clocks/fake"
)

func TestRetryCancel(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := context.WithCancel(context.Background())
	c := make(chan bool)
	backoff := DefaultBackoff()
	backoff.MinBackoff = time.Microsecond

	go func() {
		err := Retry(ctx, backoff, 18, func(ctx context.Context) error {
			c <- true
			return fmt.Errorf("foo")
		})
		theErr := &CtxErrors{}
		require.True(t, errors.As(err, &theErr))
		assert.True(t, len(theErr.Errs) > 0)
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
	c := make(chan struct{})
	backoff := DefaultBackoff()
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
}

func TestRetryUntilDeadlineLooms(t *testing.T) {
	t.Parallel()
	fc := fake.NewClock(time.Now())

	ctx, cancel := context.WithDeadline(context.Background(), fc.Now().Add(time.Second))
	defer cancel()
	c := make(chan struct{})
	backoff := DefaultBackoff()
	backoff.MinBackoff = time.Millisecond * 3
	backoff.MaxBackoff = time.Second
	backoff.Jitter = 0.01
	backoff.ExpFactor = 10

	r := NewRetryable(80)
	r.Clock = fc
	r.B = backoff

	go func() {
		q := 0
		err := r.Retry(ctx, func(ctx context.Context) error {
			q++
			return fmt.Errorf("foo")
		})

		theErr := &CtxErrors{}
		require.True(t, errors.As(err, &theErr))
		assert.EqualValues(t, context.DeadlineExceeded, theErr.CtxErr)
		assert.Len(t, theErr.Errs, 4)
		close(c)
	}()

	fc.AwaitSleepers(1)
	fc.Advance(time.Millisecond * 4)
	fc.AwaitSleepers(1)
	fc.Advance(time.Millisecond * 40)
	fc.AwaitSleepers(1)
	fc.Advance(time.Millisecond * 400)
	// We exit here because the next step would be 1s (mod jitter)
	// which brings us to 1.444s after the initial start-time, beyond the
	// 1s timeout.

	<-c
}

func TestRetryUntilExhausted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	c := make(chan struct{})
	backoff := DefaultBackoff()
	backoff.MinBackoff = time.Microsecond

	go func() {
		q := 0
		err := Retry(ctx, backoff, 8, func(ctx context.Context) error {
			q++
			return fmt.Errorf("foo")
		})

		theErr := &Errors{}
		require.True(t, errors.As(err, &theErr))
		assert.Len(t, theErr.Errs, 8)
		close(c)
	}()
	<-c
}

func TestRetriableWithFakeClock(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	c := make(chan struct{})
	backoff := Backoff{
		// Use large time-intervals since we're using a fake clock
		MaxBackoff: time.Hour * 20,
		MinBackoff: time.Hour,
		Jitter:     0.1,
		ExpFactor:  1.1,
	}

	fc := fake.NewClock(time.Now())

	go func() {
		q := 0
		r := NewRetryable(18)
		r.Clock = fc
		r.B = backoff
		err := r.Retry(ctx, func(ctx context.Context) error {
			q++
			if q == 3 {
				return nil
			}
			return fmt.Errorf("foo")
		})
		assert.NoError(t, err)
		assert.Equal(t, 3, q)
		close(c)
	}()
	// Wait for the goroutine to go to sleep
	fc.AwaitSleepers(1)
	// Advance the clock by 10 hrs so we're guaranteed to wake up
	assert.EqualValues(t, 1, fc.Advance(time.Hour*10))
	// wait for it to back to sleep again since the error fails the first two times.
	fc.AwaitSleepers(1)
	// Advance the clock by 10 hrs so we're guaranteed to wake up
	assert.EqualValues(t, 1, fc.Advance(time.Hour*10))
	// this time, we should succeed; await goroutine exit.

	<-c
}

func TestErrorsWrapping(t *testing.T) {
	last := errors.New("this should get unwrapped")
	errs := &Errors{
		Errs: []*Error{
			{Err: errors.New("error")},
			{Err: errors.New("error")},
			{Err: last},
		},
	}

	assert.True(t, errors.Is(errs, last))
}

func TestErrorsIs(t *testing.T) {
	timeout := errors.New("timeout")
	authFailure := errors.New("auth failed")
	random := errors.New("foo")

	errs := &Errors{
		Errs: []*Error{
			{Err: timeout},
			{Err: authFailure},
		},
	}

	assert.True(t, errors.Is(errs, timeout))
	assert.True(t, errors.Is(errs, authFailure))
	assert.False(t, errors.Is(errs, random))
}

type fooError struct {
	magicNum int
	err      error
}

func (re *fooError) Error() string {
	return re.err.Error()
}

func TestErrorsAs(t *testing.T) {
	timeout := errors.New("timeout")
	authFailure := errors.New("auth failed")

	errs := &Errors{
		Errs: []*Error{
			{Err: &fooError{err: timeout, magicNum: 42}},
			{Err: authFailure},
		},
	}

	var err *fooError
	require.True(t, errors.As(errs, &err))
	assert.Equal(t, 42, err.magicNum)
}
