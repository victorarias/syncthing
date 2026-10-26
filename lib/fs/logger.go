// Copyright (C) 2026 The attn Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package fs

import (
	"log/slog"

	"github.com/victorarias/syncthing/internal/slogutil"
)

type OptionLogger struct{ Logger *slog.Logger }

func (o *OptionLogger) apply(f Filesystem) Filesystem { return f }
func (o *OptionLogger) String() string                { return "logger" }
func Logger(f Filesystem) *slog.Logger {
	for _, option := range f.Options() {
		if o, ok := option.(*OptionLogger); ok {
			return slogutil.Logger(o.Logger)
		}
	}
	return slogutil.Logger()
}
