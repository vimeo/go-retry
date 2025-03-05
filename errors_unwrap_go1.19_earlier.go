//   Copyright 2025 Vimeo
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

//go:build !go1.20

package retry

import "errors"

// Unwrap returns the most recent error that occured during retrying.
func (e *Errors) Unwrap() error {
	if len(e.Errs) == 0 {
		return nil
	}
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
