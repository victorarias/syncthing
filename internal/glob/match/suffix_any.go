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

	sutil "github.com/victorarias/syncthing/internal/glob/util/strings"
)

type SuffixAny struct {
	Suffix     string
	Separators []rune
}

func NewSuffixAny(s string, sep []rune) SuffixAny {
	return SuffixAny{s, sep}
}

func (self SuffixAny) Index(s string) (int, []int) {
	idx := strings.Index(s, self.Suffix)
	if idx == -1 {
		return -1, nil
	}

	i := sutil.LastIndexAnyRunes(s[:idx], self.Separators) + 1

	return i, []int{idx + len(self.Suffix) - i}
}

func (self SuffixAny) Len() int {
	return lenNo
}

func (self SuffixAny) Match(s string) bool {
	if !strings.HasSuffix(s, self.Suffix) {
		return false
	}
	return sutil.IndexAnyRunes(s[:len(s)-len(self.Suffix)], self.Separators) == -1
}

func (self SuffixAny) String() string {
	return fmt.Sprintf("<suffix_any:![%s]%s>", string(self.Separators), self.Suffix)
}
