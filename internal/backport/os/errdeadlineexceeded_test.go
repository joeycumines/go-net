// Copyright 2022 The Go Authors.
// Copyright 2022 Joseph Cumines.
//
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/net/internal/backport"
	"net"
	"testing"
	"time"
)

func TestErrDeadlineExceeded_is(t *testing.T) {
	for _, tc := range [...]struct {
		Name       string
		Err        func(t *testing.T) error
		IsErrors   bool
		IsBackport bool
	}{
		{
			Name:       `context deadline exceeded`,
			Err:        func(t *testing.T) error { return context.DeadlineExceeded },
			IsBackport: true,
		},
		{
			Name: `pipe timeout error`,
			Err: func(t *testing.T) error {
				pipe, _ := net.Pipe()
				defer pipe.Close()
				if err := pipe.SetReadDeadline(time.Now()); err != nil {
					t.Fatal(err)
				}
				_, err := pipe.Read(make([]byte, 1))
				if err == nil {
					panic(`expected error`)
				}
				return err
			},
			IsBackport: true,
		},
		{
			Name: `poll errtimeout go114`,
			Err: func(t *testing.T) error {
				ctx, cancel := context.WithTimeout(context.Background(), 1)
				defer cancel()
				<-ctx.Done()
				_, err := (&net.Dialer{}).DialContext(ctx, `tcp`, `localhost:0`)
				if err == nil {
					t.Fatal(`expected error`)
				}
				return err
			},
			IsBackport: true,
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.Err(t)
			for e := err; e != nil; e = errors.Unwrap(e) {
				t.Logf("%T %s\n", e, e)
			}
			if errors.Is(err, ErrDeadlineExceeded) != tc.IsErrors {
				t.Error(err)
			}
			if backport.ErrorIs(err, ErrDeadlineExceeded) != tc.IsBackport {
				t.Error(err)
			}
			if backport.ErrorIs(fmt.Errorf(`wrapped: %w`, err), ErrDeadlineExceeded) != tc.IsBackport {
				t.Error(err)
			}
		})
	}
}
