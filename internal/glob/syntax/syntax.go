// Copyright (C) 2026 The attn Authors.
// Derived from github.com/calmh/glob at 1d823af5017b.
// Original copyright and MIT terms are in LICENSE.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package syntax

import (
	"github.com/victorarias/syncthing/internal/glob/syntax/ast"
	"github.com/victorarias/syncthing/internal/glob/syntax/lexer"
)

func Parse(s string) (*ast.Node, error) {
	return ast.Parse(lexer.NewLexer(s))
}

func Special(b byte) bool {
	return lexer.Special(b)
}
