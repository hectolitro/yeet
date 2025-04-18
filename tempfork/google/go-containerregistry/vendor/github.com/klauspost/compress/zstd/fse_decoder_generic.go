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

//go:build !amd64 || appengine || !gc || noasm
// +build !amd64 appengine !gc noasm

package zstd

import (
	"errors"
	"fmt"
)

// buildDtable will build the decoding table.
func (s *fseDecoder) buildDtable() error {
	tableSize := uint32(1 << s.actualTableLog)
	highThreshold := tableSize - 1
	symbolNext := s.stateTable[:256]

	// Init, lay down lowprob symbols
	{
		for i, v := range s.norm[:s.symbolLen] {
			if v == -1 {
				s.dt[highThreshold].setAddBits(uint8(i))
				highThreshold--
				symbolNext[i] = 1
			} else {
				symbolNext[i] = uint16(v)
			}
		}
	}

	// Spread symbols
	{
		tableMask := tableSize - 1
		step := tableStep(tableSize)
		position := uint32(0)
		for ss, v := range s.norm[:s.symbolLen] {
			for i := 0; i < int(v); i++ {
				s.dt[position].setAddBits(uint8(ss))
				position = (position + step) & tableMask
				for position > highThreshold {
					// lowprob area
					position = (position + step) & tableMask
				}
			}
		}
		if position != 0 {
			// position must reach all cells once, otherwise normalizedCounter is incorrect
			return errors.New("corrupted input (position != 0)")
		}
	}

	// Build Decoding table
	{
		tableSize := uint16(1 << s.actualTableLog)
		for u, v := range s.dt[:tableSize] {
			symbol := v.addBits()
			nextState := symbolNext[symbol]
			symbolNext[symbol] = nextState + 1
			nBits := s.actualTableLog - byte(highBits(uint32(nextState)))
			s.dt[u&maxTableMask].setNBits(nBits)
			newState := (nextState << nBits) - tableSize
			if newState > tableSize {
				return fmt.Errorf("newState (%d) outside table size (%d)", newState, tableSize)
			}
			if newState == uint16(u) && nBits == 0 {
				// Seems weird that this is possible with nbits > 0.
				return fmt.Errorf("newState (%d) == oldState (%d) and no bits", newState, u)
			}
			s.dt[u&maxTableMask].setNewState(newState)
		}
	}
	return nil
}
