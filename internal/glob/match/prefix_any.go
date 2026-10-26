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

	sutil "github.com/victorarias/syncthing/internal/glob/util/strings"
)

type PrefixAny struct {
	Prefix     string
	Separators []rune
}

func NewPrefixAny(s string, sep []rune) PrefixAny {
	return PrefixAny{s, sep}
}

func (self PrefixAny) Index(s string) (int, []int) {
	idx := strings.Index(s, self.Prefix)
	if idx == -1 {
		return -1, nil
	}

	n := len(self.Prefix)
	sub := s[idx+n:]
	i := sutil.IndexAnyRunes(sub, self.Separators)
	if i > -1 {
		sub = sub[:i]
	}

	seg := acquireSegments(len(sub) + 1)
	seg = append(seg, n)
	for i, r := range sub {
		seg = append(seg, n+i+utf8.RuneLen(r))
	}

	return idx, seg
}

func (self PrefixAny) Len() int {
	return lenNo
}

func (self PrefixAny) Match(s string) bool {
	if !strings.HasPrefix(s, self.Prefix) {
		return false
	}
	return sutil.IndexAnyRunes(s[len(self.Prefix):], self.Separators) == -1
}

func (self PrefixAny) String() string {
	return fmt.Sprintf("<prefix_any:%s![%s]>", self.Prefix, string(self.Separators))
}
