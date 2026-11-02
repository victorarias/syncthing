// Copyright (C) 2026 The attn Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"os"
	"runtime"

	_ "github.com/victorarias/syncthing/lib/attnsync"
)

func main() {
	result := struct{ Goroutines int }{Goroutines: runtime.NumGoroutine()}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err)
	}
}
