// Copyright 2020 The Go Authors.
// Copyright 2022 Joseph Cumines.
//
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !go1.14
// +build !go1.14

package os

var (
	ErrDeadlineExceeded = os.ErrDeadlineExceeded
)
