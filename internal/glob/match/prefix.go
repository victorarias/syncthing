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
	"strings"
	"unicode/utf8"
)

type Prefix struct {
	Prefix string
}

func NewPrefix(p string) Prefix {
	return Prefix{p}
}

func (self Prefix) Index(s string) (int, []int) {
	idx := strings.Index(s, self.Prefix)
	if idx == -1 {
		return -1, nil
	}

	length := len(self.Prefix)
	var sub string
	if len(s) > idx+length {
		sub = s[idx+length:]
	} else {
		sub = ""
	}

	segments := acquireSegments(len(sub) + 1)
	segments = append(segments, length)
	for i, r := range sub {
		segments = append(segments, length+i+utf8.RuneLen(r))
	}

	return idx, segments
}

func (self Prefix) Len() int {
	return lenNo
}

func (self Prefix) Match(s string) bool {
	return strings.HasPrefix(s, self.Prefix)
}

func (self Prefix) String() string {
	return fmt.Sprintf("<prefix:%s>", self.Prefix)
}
