// Copyright (C) 2026 The attn Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package attnsync_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportLeavesTheProcessIdle(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "importprobe")
	if output, err := exec.Command("go", "build", "-o", binary, "./testdata/importprobe").CombinedOutput(); err != nil {
		t.Fatalf("build import probe: %v\n%s", err, output)
	}
	output, err := exec.Command(binary).CombinedOutput()
	if err != nil {
		t.Fatalf("run import probe: %v\n%s", err, output)
	}
	var result struct{ Goroutines int }
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode import probe: %v\n%s", err, output)
	}
	if result.Goroutines != 1 {
		t.Fatalf("import started background work: %s", output)
	}
}

func TestImportLinksNoMetricsLibrary(t *testing.T) {
	output, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("list dependencies: %v\n%s", err, output)
	}
	for _, pkg := range strings.Fields(string(output)) {
		if strings.HasPrefix(pkg, "github.com/prometheus/") {
			t.Fatalf("attnsync links %s, whose package init every importing process pays", pkg)
		}
	}
}
