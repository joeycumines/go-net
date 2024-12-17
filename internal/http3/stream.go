// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.24

package http3

import (
	"io"

	"golang.org/x/net/quic"
)

// A stream wraps a QUIC stream, providing methods to read/write various values.
type stream struct {
	stream *quic.Stream

	// lim is the current read limit.
	// Reading a frame header sets the limit to the end of the frame.
	// Reading past the limit or reading less than the limit and ending the frame
	// results in an error.
	// -1 indicates no limit.
	lim int64
}

func newStream(qs *quic.Stream) *stream {
	return &stream{
		stream: qs,
		lim:    -1, // no limit
	}
}

// readFrameHeader reads the type and length fields of an HTTP/3 frame.
// It sets the read limit to the end of the frame.
//
// https://www.rfc-editor.org/rfc/rfc9114.html#section-7.1
func (st *stream) readFrameHeader() (ftype frameType, err error) {
	if st.lim >= 0 {
		// We shoudn't call readFrameHeader before ending the previous frame.
		return 0, errH3FrameError
	}
	ftype, err = readVarint[frameType](st)
	if err != nil {
		return 0, err
	}
	size, err := st.readVarint()
	if err != nil {
		return 0, err
	}
	st.lim = size
	return ftype, nil
}

// endFrame is called after reading a frame to reset the read limit.
// It returns an error if the entire contents of a frame have not been read.
func (st *stream) endFrame() error {
	if st.lim != 0 {
		return errH3FrameError
	}
	st.lim = -1
	return nil
}

// readFrameData returns the remaining data in the current frame.
func (st *stream) readFrameData() ([]byte, error) {
	if st.lim < 0 {
		return nil, errH3FrameError
	}
	// TODO: Pool buffers to avoid allocation here.
	b := make([]byte, st.lim)
	_, err := io.ReadFull(st, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// ReadByte reads one byte from the stream.
func (st *stream) ReadByte() (b byte, err error) {
	if err := st.recordBytesRead(1); err != nil {
		return 0, err
	}
	b, err = st.stream.ReadByte()
	if err != nil {
		if err == io.EOF {
			return 0, io.EOF
		}
		return 0, errH3FrameError
	}
	return b, nil
}

// Read reads from the stream.
func (st *stream) Read(b []byte) (int, error) {
	n, err := st.stream.Read(b)
	if err != nil {
		if err == io.EOF {
			return 0, io.EOF
		}
		return 0, errH3FrameError
	}
	if err := st.recordBytesRead(n); err != nil {
		return 0, err
	}
	return n, nil
}

// Write writes to the stream.
func (st *stream) Write(b []byte) (int, error) { return st.stream.Write(b) }

// Flush commits data written to the stream.
func (st *stream) Flush() error { return st.stream.Flush() }

// readVarint reads a QUIC variable-length integer from the stream.
func (st *stream) readVarint() (v int64, err error) {
	b, err := st.stream.ReadByte()
	if err != nil {
		return 0, err
	}
	v = int64(b & 0x3f)
	n := 1 << (b >> 6)
	for i := 1; i < n; i++ {
		b, err := st.stream.ReadByte()
		if err != nil {
			return 0, errH3FrameError
		}
		v = (v << 8) | int64(b)
	}
	if err := st.recordBytesRead(n); err != nil {
		return 0, err
	}
	return v, nil
}

// readVarint reads a varint of a particular type.
func readVarint[T ~int64 | ~uint64](st *stream) (T, error) {
	v, err := st.readVarint()
	return T(v), err
}

// writeVarint writes a QUIC variable-length integer to the stream.
func (st *stream) writeVarint(v int64) {
	switch {
	case v <= (1<<6)-1:
		st.stream.WriteByte(byte(v))
	case v <= (1<<14)-1:
		st.stream.WriteByte((1 << 6) | byte(v>>8))
		st.stream.WriteByte(byte(v))
	case v <= (1<<30)-1:
		st.stream.WriteByte((2 << 6) | byte(v>>24))
		st.stream.WriteByte(byte(v >> 16))
		st.stream.WriteByte(byte(v >> 8))
		st.stream.WriteByte(byte(v))
	case v <= (1<<62)-1:
		st.stream.WriteByte((3 << 6) | byte(v>>56))
		st.stream.WriteByte(byte(v >> 48))
		st.stream.WriteByte(byte(v >> 40))
		st.stream.WriteByte(byte(v >> 32))
		st.stream.WriteByte(byte(v >> 24))
		st.stream.WriteByte(byte(v >> 16))
		st.stream.WriteByte(byte(v >> 8))
		st.stream.WriteByte(byte(v))
	default:
		panic("varint too large")
	}
}

// recordBytesRead records that n bytes have been read.
// It returns an error if the read passes the current limit.
func (st *stream) recordBytesRead(n int) error {
	if st.lim < 0 {
		return nil
	}
	st.lim -= int64(n)
	if st.lim < 0 {
		st.stream = nil // panic if we try to read again
		return errH3FrameError
	}
	return nil
}
