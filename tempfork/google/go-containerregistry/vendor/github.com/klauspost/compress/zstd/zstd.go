// Copyright 2025 AUTHORS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package zstd provides decompression of zstandard files.
//
// For advanced usage and examples, go to the README: https://github.com/klauspost/compress/tree/master/zstd#zstd
package zstd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"math"
	"math/bits"
)

// enable debug printing
const debug = false

// enable encoding debug printing
const debugEncoder = debug

// enable decoding debug printing
const debugDecoder = debug

// Enable extra assertions.
const debugAsserts = debug || false

// print sequence details
const debugSequences = false

// print detailed matching information
const debugMatches = false

// force encoder to use predefined tables.
const forcePreDef = false

// zstdMinMatch is the minimum zstd match length.
const zstdMinMatch = 3

// fcsUnknown is used for unknown frame content size.
const fcsUnknown = math.MaxUint64

var (
	// ErrReservedBlockType is returned when a reserved block type is found.
	// Typically this indicates wrong or corrupted input.
	ErrReservedBlockType = errors.New("invalid input: reserved block type encountered")

	// ErrCompressedSizeTooBig is returned when a block is bigger than allowed.
	// Typically this indicates wrong or corrupted input.
	ErrCompressedSizeTooBig = errors.New("invalid input: compressed size too big")

	// ErrBlockTooSmall is returned when a block is too small to be decoded.
	// Typically returned on invalid input.
	ErrBlockTooSmall = errors.New("block too small")

	// ErrUnexpectedBlockSize is returned when a block has unexpected size.
	// Typically returned on invalid input.
	ErrUnexpectedBlockSize = errors.New("unexpected block size")

	// ErrMagicMismatch is returned when a "magic" number isn't what is expected.
	// Typically this indicates wrong or corrupted input.
	ErrMagicMismatch = errors.New("invalid input: magic number mismatch")

	// ErrWindowSizeExceeded is returned when a reference exceeds the valid window size.
	// Typically this indicates wrong or corrupted input.
	ErrWindowSizeExceeded = errors.New("window size exceeded")

	// ErrWindowSizeTooSmall is returned when no window size is specified.
	// Typically this indicates wrong or corrupted input.
	ErrWindowSizeTooSmall = errors.New("invalid input: window size was too small")

	// ErrDecoderSizeExceeded is returned if decompressed size exceeds the configured limit.
	ErrDecoderSizeExceeded = errors.New("decompressed size exceeds configured limit")

	// ErrUnknownDictionary is returned if the dictionary ID is unknown.
	ErrUnknownDictionary = errors.New("unknown dictionary")

	// ErrFrameSizeExceeded is returned if the stated frame size is exceeded.
	// This is only returned if SingleSegment is specified on the frame.
	ErrFrameSizeExceeded = errors.New("frame size exceeded")

	// ErrFrameSizeMismatch is returned if the stated frame size does not match the expected size.
	// This is only returned if SingleSegment is specified on the frame.
	ErrFrameSizeMismatch = errors.New("frame size does not match size on stream")

	// ErrCRCMismatch is returned if CRC mismatches.
	ErrCRCMismatch = errors.New("CRC check failed")

	// ErrDecoderClosed will be returned if the Decoder was used after
	// Close has been called.
	ErrDecoderClosed = errors.New("decoder used after Close")

	// ErrDecoderNilInput is returned when a nil Reader was provided
	// and an operation other than Reset/DecodeAll/Close was attempted.
	ErrDecoderNilInput = errors.New("nil input provided as reader")
)

func println(a ...interface{}) {
	if debug || debugDecoder || debugEncoder {
		log.Println(a...)
	}
}

func printf(format string, a ...interface{}) {
	if debug || debugDecoder || debugEncoder {
		log.Printf(format, a...)
	}
}

// matchLen returns the maximum common prefix length of a and b.
// a must be the shortest of the two.
func matchLen(a, b []byte) (n int) {
	for ; len(a) >= 8 && len(b) >= 8; a, b = a[8:], b[8:] {
		diff := binary.LittleEndian.Uint64(a) ^ binary.LittleEndian.Uint64(b)
		if diff != 0 {
			return n + bits.TrailingZeros64(diff)>>3
		}
		n += 8
	}

	for i := range a {
		if a[i] != b[i] {
			break
		}
		n++
	}
	return n

}

func load3232(b []byte, i int32) uint32 {
	return binary.LittleEndian.Uint32(b[:len(b):len(b)][i:])
}

func load6432(b []byte, i int32) uint64 {
	return binary.LittleEndian.Uint64(b[:len(b):len(b)][i:])
}

type byter interface {
	Bytes() []byte
	Len() int
}

var _ byter = &bytes.Buffer{}
