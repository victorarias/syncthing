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
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	_ "github.com/victorarias/syncthing/lib/attnsync"
)

func main() {
	result := struct {
		Goroutines int
		Collectors []string
	}{Goroutines: runtime.NumGoroutine()}
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		panic(err)
	}
	for _, family := range families {
		if strings.HasPrefix(family.GetName(), "syncthing_") {
			result.Collectors = append(result.Collectors, family.GetName())
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err)
	}
}
