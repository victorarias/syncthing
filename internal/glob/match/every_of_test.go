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

func TestEveryOfIndex(t *testing.T) {
	for id, test := range []struct {
		matchers Matchers
		fixture  string
		index    int
		segments []int
	}{
		{
			Matchers{
				NewAny(nil),
				NewText("b"),
				NewText("c"),
			},
			"dbc",
			-1,
			nil,
		},
		{
			Matchers{
				NewAny(nil),
				NewPrefix("b"),
				NewSuffix("c"),
			},
			"abc",
			1,
			[]int{2},
		},
	} {
		everyOf := NewEveryOf(test.matchers...)
		index, segments := everyOf.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}
