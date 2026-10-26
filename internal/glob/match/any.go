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

	"github.com/victorarias/syncthing/internal/glob/util/strings"
)

type Any struct {
	Separators []rune
}

func NewAny(s []rune) Any {
	return Any{s}
}

func (self Any) Match(s string) bool {
	return strings.IndexAnyRunes(s, self.Separators) == -1
}

func (self Any) Index(s string) (int, []int) {
	found := strings.IndexAnyRunes(s, self.Separators)
	switch found {
	case -1:
	case 0:
		return 0, segments0
	default:
		s = s[:found]
	}

	segments := acquireSegments(len(s))
	for i := range s {
		segments = append(segments, i)
	}
	segments = append(segments, len(s))

	return 0, segments
}

func (self Any) Len() int {
	return lenNo
}

func (self Any) String() string {
	return fmt.Sprintf("<any:![%s]>", string(self.Separators))
}
