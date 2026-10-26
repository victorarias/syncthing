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

type Min struct {
	Limit int
}

func NewMin(l int) Min {
	return Min{l}
}

func (self Min) Match(s string) bool {
	var l int
	for range s {
		l += 1
		if l >= self.Limit {
			return true
		}
	}

	return false
}

func (self Min) Index(s string) (int, []int) {
	var count int

	c := len(s) - self.Limit + 1
	if c <= 0 {
		return -1, nil
	}

	segments := acquireSegments(c)
	for i, r := range s {
		count++
		if count >= self.Limit {
			segments = append(segments, i+utf8.RuneLen(r))
		}
	}

	if len(segments) == 0 {
		return -1, nil
	}

	return 0, segments
}

func (self Min) Len() int {
	return lenNo
}

func (self Min) String() string {
	return fmt.Sprintf("<min:%d>", self.Limit)
}
