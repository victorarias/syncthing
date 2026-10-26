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

func TestNothingIndex(t *testing.T) {
	for id, test := range []struct {
		fixture  string
		index    int
		segments []int
	}{
		{
			"abc",
			0,
			[]int{0},
		},
		{
			"",
			0,
			[]int{0},
		},
	} {
		p := NewNothing()
		index, segments := p.Index(test.fixture)
		if index != test.index {
			t.Errorf("#%d unexpected index: exp: %d, act: %d", id, test.index, index)
		}
		if !reflect.DeepEqual(segments, test.segments) {
			t.Errorf("#%d unexpected segments: exp: %v, act: %v", id, test.segments, segments)
		}
	}
}

func BenchmarkIndexNothing(b *testing.B) {
	m := NewNothing()

	for i := 0; i < b.N; i++ {
		_, s := m.Index(bench_pattern)
		releaseSegments(s)
	}
}

func BenchmarkIndexNothingParallel(b *testing.B) {
	m := NewNothing()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, s := m.Index(bench_pattern)
			releaseSegments(s)
		}
	})
}
