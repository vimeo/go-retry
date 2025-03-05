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

//go:build go1.20

package retry

// Unwrap returns the errors that occured during retrying.
func (e *Errors) Unwrap() []error {
	if len(e.Errs) == 0 {
		return nil
	}
	out := make([]error, len(e.Errs))
	for i, err := range e.Errs {
		out[i] = err
	}
	return out
}
