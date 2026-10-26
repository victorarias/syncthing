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
)

type Suffix struct {
	Suffix string
}

func NewSuffix(s string) Suffix {
	return Suffix{s}
}

func (self Suffix) Len() int {
	return lenNo
}

func (self Suffix) Match(s string) bool {
	return strings.HasSuffix(s, self.Suffix)
}

func (self Suffix) Index(s string) (int, []int) {
	idx := strings.Index(s, self.Suffix)
	if idx == -1 {
		return -1, nil
	}

	return 0, []int{idx + len(self.Suffix)}
}

func (self Suffix) String() string {
	return fmt.Sprintf("<suffix:%s>", self.Suffix)
}
