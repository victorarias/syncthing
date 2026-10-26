// Copyright (C) 2026 The attn Authors.
// Derived from github.com/calmh/glob at 1d823af5017b.
// Original copyright and MIT terms are in LICENSE.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package match

import (
	"reflect"
	"testing"
)

func TestSuffixAnyIndex(t *testing.T) {
	for id, test := range []struct {
		suffix     string
		separators []rune
		fixture    string
		index      int
		segments   []int
	}{
		{
			"ab",
			[]rune{'.'},
			"ab",
			0,
			[]int{2},
		},
		{
			"ab",
			[]rune{'.'},
			"cab",
			0,
			[]int{3},
		},
		{
			"ab",
			[]rune{'.'},
			"qw.cdab.efg",
			3,
			[]int{4},
		},
	} {
		p := NewSuffixAny(test.suffix, test.separators)
		index, segments := p.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}
