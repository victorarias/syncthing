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
	var result struct {
		Goroutines int
		Collectors []string
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode import probe: %v\n%s", err, output)
	}
	if result.Goroutines != 1 || len(result.Collectors) != 0 {
		t.Fatalf("import started background work: %s", output)
	}
}
