// Copyright 2022 The Go Authors.
// Copyright 2022 Joseph Cumines.
//
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package backport

import (
	"errors"
)

// ErrorIs performs a normal errors.Is then, if false, checks target.Backport_Is against every layer of err
func ErrorIs(err error, target error) bool {
	if errors.Is(err, target) {
		return true
	}
	t, ok := target.(interface{ Backport_Is(err error) bool })
	if !ok {
		return false
	}
	for {
		if t.Backport_Is(err) {
			return true
		}
		if err = errors.Unwrap(err); err == nil {
			return false
		}
	}
}
