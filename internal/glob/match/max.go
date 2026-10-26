// Copyright (C) 2026 The attn Authors.
// Derived from github.com/calmh/glob at 1d823af5017b.
// Original copyright and MIT terms are in LICENSE.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package match

import (
	"fmt"
	"unicode/utf8"
)

type Max struct {
	Limit int
}

func NewMax(l int) Max {
	return Max{l}
}

func (self Max) Match(s string) bool {
	var l int
	for range s {
		l += 1
		if l > self.Limit {
			return false
		}
	}

	return true
}

func (self Max) Index(s string) (int, []int) {
	segments := acquireSegments(self.Limit + 1)
	segments = append(segments, 0)
	var count int
	for i, r := range s {
		count++
		if count > self.Limit {
			break
		}
		segments = append(segments, i+utf8.RuneLen(r))
	}

	return 0, segments
}

func (self Max) Len() int {
	return lenNo
}

func (self Max) String() string {
	return fmt.Sprintf("<max:%d>", self.Limit)
}
