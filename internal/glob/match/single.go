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

	"github.com/victorarias/syncthing/internal/glob/util/runes"
)

// single represents ?
type Single struct {
	Separators []rune
}

func NewSingle(s []rune) Single {
	return Single{s}
}

func (self Single) Match(s string) bool {
	r, w := utf8.DecodeRuneInString(s)
	if len(s) > w {
		return false
	}

	return runes.IndexRune(self.Separators, r) == -1
}

func (self Single) Len() int {
	return lenOne
}

func (self Single) Index(s string) (int, []int) {
	for i, r := range s {
		if runes.IndexRune(self.Separators, r) == -1 {
			return i, segmentsByRuneLength[utf8.RuneLen(r)]
		}
	}

	return -1, nil
}

func (self Single) String() string {
	return fmt.Sprintf("<single:![%s]>", string(self.Separators))
}
