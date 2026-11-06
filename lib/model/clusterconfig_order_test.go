// Copyright (C) 2026 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"testing"

	"github.com/victorarias/syncthing/lib/config"
	"github.com/victorarias/syncthing/lib/protocol"
	"github.com/victorarias/syncthing/lib/rand"
)

func folderStateIn(cc *protocol.ClusterConfig, id string) (protocol.FolderStopReason, bool) {
	for _, f := range cc.Folders {
		if f.ID == id {
			return f.StopReason, true
		}
	}
	return 0, false
}

// A ClusterConfig generated after the configuration gains a folder, but
// before the folder's runner starts, must not announce the folder as
// running: the peer would start sending indexes for it to a folder that
// cannot take them yet.
func TestClusterConfigAnnouncesFolderOnlyOnceRunning(t *testing.T) {
	w, _ := newDefaultCfgWrapper(t)
	m, fc := setupModelWithConnectionFromWrapper(t, w)
	defer cleanupModel(m)

	second := newFolderConfiguration(defaultCfgWrapper, "second", "second", config.FilesystemTypeFake, rand.String(32)+"?content=true")
	second.Devices = append(second.Devices, config.FolderDeviceConfiguration{DeviceID: device1})

	// Holding the read lock keeps the model from starting the folder, so
	// the configuration has it while its runner does not exist yet.
	m.mut.RLock()
	waiter, err := w.Modify(func(cfg *config.Configuration) { cfg.SetFolder(second) })
	if err != nil {
		m.mut.RUnlock()
		t.Fatal(err)
	}
	early, _ := m.generateClusterConfigRLocked(device1)
	m.mut.RUnlock()
	waiter.Wait()

	if state, ok := folderStateIn(early, "second"); !ok || state != protocol.FolderStopReasonPaused {
		t.Fatalf("ClusterConfig before the runner started lists second as %v (present %v), want paused", state, ok)
	}

	sent, _ := fc.ClusterConfigArgsForCall(fc.ClusterConfigCallCount() - 1)
	if state, ok := folderStateIn(sent, "second"); !ok || state != protocol.FolderStopReasonRunning {
		t.Fatalf("ClusterConfig sent after the folder started lists second as %v (present %v), want running", state, ok)
	}
}
