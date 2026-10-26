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
)

type Nothing struct{}

func NewNothing() Nothing {
	return Nothing{}
}

func (self Nothing) Match(s string) bool {
	return len(s) == 0
}

func (self Nothing) Index(s string) (int, []int) {
	return 0, segments0
}

func (self Nothing) Len() int {
	return lenZero
}

func (self Nothing) String() string {
	return fmt.Sprintf("<nothing>")
}
