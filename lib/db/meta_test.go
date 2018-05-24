// Copyright (C) 2018 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package db

import (
	"math/bits"
	"testing"
)

func TestEachFlagBit(t *testing.T) {
	cases := []struct {
		flags      uint32
		iterations int
	}{
		{0, 0},
		{1<<0 | 1<<3, 2},
		{1 << 3, 1},
		{1 << 31, 1},
		{1<<10 | 1<<20 | 1<<30, 3},
	}

	for _, tc := range cases {
		var flags uint32
		iterations := 0

		eachFlagBit(tc.flags, func(f uint32) {
			iterations++
			flags |= f
			if bits.OnesCount32(f) != 1 {
				t.Error("expected exactly one bit to be set in every call")
			}
		})

		if flags != tc.flags {
			t.Errorf("expected 0x%x flags, got 0x%x", tc.flags, flags)
		}
		if iterations != tc.iterations {
			t.Errorf("expected %d iterations, got %d", tc.iterations, iterations)
		}
	}
}
