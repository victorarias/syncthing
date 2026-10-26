// Copyright (C) 2026 The attn Authors.
// Derived from github.com/calmh/glob at 1d823af5017b.
// Original copyright and MIT terms are in LICENSE.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package lexer

import "fmt"

type TokenType int

const (
	EOF TokenType = iota
	Error
	Text
	Char
	Any
	Super
	Single
	Not
	Separator
	RangeOpen
	RangeClose
	RangeLo
	RangeHi
	RangeBetween
	TermsOpen
	TermsClose
)

func (tt TokenType) String() string {
	switch tt {
	case EOF:
		return "eof"

	case Error:
		return "error"

	case Text:
		return "text"

	case Char:
		return "char"

	case Any:
		return "any"

	case Super:
		return "super"

	case Single:
		return "single"

	case Not:
		return "not"

	case Separator:
		return "separator"

	case RangeOpen:
		return "range_open"

	case RangeClose:
		return "range_close"

	case RangeLo:
		return "range_lo"

	case RangeHi:
		return "range_hi"

	case RangeBetween:
		return "range_between"

	case TermsOpen:
		return "terms_open"

	case TermsClose:
		return "terms_close"

	default:
		return "undef"
	}
}

type Token struct {
	Type TokenType
	Raw  string
}

func (t Token) String() string {
	return fmt.Sprintf("%v<%q>", t.Type, t.Raw)
}
