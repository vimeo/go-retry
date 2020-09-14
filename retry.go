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
	"time"

	clocks "github.com/vimeo/go-clocks"
)

// Retryable manages the operations of a retryable operation.
type Retryable struct {
	// Backoff parameters to use for retry
	B Backoff

	// ShouldRetry is a filter function to indicate whether to continue
	// iterating based on the error.
	// An implementation that uniformly returns true is used if nil
	ShouldRetry func(error) bool

	// Maximum retry attempts
	MaxSteps int32

	// Clock provides a clock to use when backing off (if nil, uses
	// github.com/vimeo/go-clocks.DefaultClock())
	Clock clocks.Clock
}

// NewRetryable returns a newly constructed Retryable instance
func NewRetryable(MaxSteps int32) *Retryable {
	return &Retryable{
		B:           DefaultBackoff(),
		ShouldRetry: nil,
		MaxSteps:    MaxSteps,
		Clock:       clocks.DefaultClock(),
	}
}

func (r *Retryable) clock() clocks.Clock {
	if r.Clock == nil {
		return clocks.DefaultClock()
	}
	return r.Clock
}

// Retry calls the function `f` at most `MaxSteps` times using the exponential
// backoff parameters defined in `B`, or until the context expires.
func (r *Retryable) Retry(ctx context.Context, f func(context.Context) error) error {
	b := r.B.Clone()
	b.Reset()
	filter := r.ShouldRetry
	if filter == nil {
		filter = func(err error) bool {
			return true
		}
	}
	errors := &Errors{}
	for n := int32(0); n < r.MaxSteps; n++ {
		err := f(ctx)
		if err == nil {
			return nil
		}
		if !filter(err) {
			return err
		}
		errors.Errs = append(errors.Errs, &Error{
			When: r.clock().Now(),
			Err:  err,
		})
		if !r.clock().SleepFor(ctx, b.Next()) {
			return &CtxErrors{
				Errors: errors,
				CtxErr: ctx.Err(),
			}
		}
	}
	return errors
}

// Retry calls the function `f` at most `steps` times using the exponential
// backoff parameters defined in `b`, or until the context expires.
func Retry(ctx context.Context, b Backoff, steps int, f func(context.Context) error) error {
	// Make sure b is clean (it's passed by value so there are no
	// observable effects of this).
	b.Reset()
	r := Retryable{B: b, MaxSteps: int32(steps), Clock: clocks.DefaultClock()}
	return r.Retry(ctx, f)
}

// Error is an error that occurs at a particular time.
type Error struct {
	// When is when the error occured in the retry cycle.
	When time.Time

	// Err is the underlying error.
	Err error
}

// Unwrap follows go-1.13-style wrapping semantics.
func (e *Error) Unwrap() error {
	return e.Err
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("Error at %s: %s", e.When, e.Err.Error())
}

// Errors is a collection errors that happen across multiple retries.
type Errors struct {
	Errs []*Error
}

// Unwrap returns the most recent error that occured during retrying.
func (e *Errors) Unwrap() error {
	return e.Errs[len(e.Errs)-1]
}

// Is will return true if any of the underlying errors matches the target.  See
// https://golang.org/pkg/errors/#Is
func (e *Errors) Is(target error) bool {
	for _, err := range e.Errs {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// As will return true if any of the underlying errors matches the target and
// sets the argument to that error specifically.  It returns false otherwise,
// leaving the argument unchanged.  See https://golang.org/pkg/errors/#As
func (e *Errors) As(target interface{}) bool {
	for _, err := range e.Errs {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}

// Error implements the error interface.
func (e *Errors) Error() string {
	return fmt.Sprintf("errors retrying: %+v", e.Errs)
}

// CtxErrors bundles together Errors and a Ctx error to differentiate the errors
// that fail due to context expiration errors from errors that exhaust their
// maximum number of retries.
type CtxErrors struct {
	*Errors
	CtxErr error
}
