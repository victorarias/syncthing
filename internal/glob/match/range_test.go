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

func TestRangeIndex(t *testing.T) {
	for id, test := range []struct {
		lo, hi   rune
		not      bool
		fixture  string
		index    int
		segments []int
	}{
		{
			'a', 'z',
			false,
			"abc",
			0,
			[]int{1},
		},
		{
			'a', 'c',
			false,
			"abcd",
			0,
			[]int{1},
		},
		{
			'a', 'c',
			true,
			"abcd",
			3,
			[]int{1},
		},
	} {
		m := NewRange(test.lo, test.hi, test.not)
		index, segments := m.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}

func BenchmarkIndexRange(b *testing.B) {
	m := NewRange('0', '9', false)

	for i := 0; i < b.N; i++ {
		_, s := m.Index(bench_pattern)
		releaseSegments(s)
	}
}

func BenchmarkIndexRangeParallel(b *testing.B) {
	m := NewRange('0', '9', false)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, s := m.Index(bench_pattern)
			releaseSegments(s)
		}
	})
}
