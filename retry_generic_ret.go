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

//go:build go1.18
// +build go1.18

package retry

import (
	"context"
)

// Typed provides a wrapper around the Retryable type that handles
// arbitrary callback return-types in addition to an error.
func Typed[T any](ctx context.Context, r *Retryable, f func(context.Context) (T, error)) (T, error) {
	var ret T

	err := r.Retry(ctx, func(ctx context.Context) error {
		rv, callErr := f(ctx)
		if callErr != nil {
			return callErr
		}
		ret = rv
		return nil
	})
	return ret, err
}
