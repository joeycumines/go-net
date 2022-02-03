// Copyright 2022 The Go Authors.
// Copyright 2022 Joseph Cumines.
//
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.14
// +build go1.14

package os

import (
	"net"
)

type (
	// deadlineExceededError matches the behavior of internal/poll/fd.DeadlineExceededError
	deadlineExceededError struct{}
)

var (
	// ErrDeadlineExceeded was added in
	// https://github.com/golang/go/commit/d422f54619b5b6e6301eaa3e9f22cfa7b65063c8
	ErrDeadlineExceeded error = &deadlineExceededError{}
)

func (x *deadlineExceededError) Error() string   { return "i/o timeout" }
func (x *deadlineExceededError) Timeout() bool   { return true }
func (x *deadlineExceededError) Temporary() bool { return true }
func (x *deadlineExceededError) Backport_Is(err error) bool {
	if err, ok := err.(net.Error); ok && x.Timeout() == err.Timeout() && x.Temporary() == err.Temporary() {
		// there's not much else we can do here (the reason for the change in the first place)
		return true
	}
	return false
}
