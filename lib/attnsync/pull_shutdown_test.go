// Copyright (C) 2026 The attn Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package attnsync_test

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"testing/synctest"

	"github.com/victorarias/syncthing/lib/attnsync"
	"github.com/victorarias/syncthing/lib/events"
	"github.com/victorarias/syncthing/lib/protocol"
)

func TestShutdownDuringAPullLeavesNoBackgroundWork(t *testing.T) {
	const files, finishedBeforeShutdown = 400, 200
	aID, bID := protocol.NewDeviceID([]byte("pull-source")), protocol.NewDeviceID([]byte("pull-target"))
	aCfg, aRoot := configuration(t, aID, bID)
	bCfg, _ := configuration(t, bID, aID)
	aCfg.Folders[0].FSWatcherEnabled, bCfg.Folders[0].FSWatcherEnabled = false, false
	for i := range files {
		write(t, filepath.Join(aRoot, fmt.Sprintf("note-%03d.md", i)), "note")
	}
	synctest.Test(t, func(t *testing.T) {
		a := start(t, aCfg, aID)
		b, err := attnsync.New(filepath.Join(t.TempDir(), "index"), bCfg, bID, nil)
		if err != nil {
			t.Fatal(err)
		}
		serving, shutdown := context.WithCancel(context.Background())
		stopped := make(chan error, 1)
		go func() { stopped <- b.Serve(serving) }()
		<-b.Ready()
		finished := b.Subscribe(events.ItemFinished)
		pulling := make(chan struct{})
		go func() {
			count := 0
			for range finished.C() {
				if count++; count == finishedBeforeShutdown {
					close(pulling)
				}
			}
		}()
		left, right := net.Pipe()
		connected, disconnect := context.WithCancel(context.Background())
		ended := make(chan error, 2)
		go func() { ended <- a.ServeConn(connected, left, bID) }()
		go func() { ended <- b.ServeConn(connected, right, aID) }()
		<-pulling
		shutdown()
		<-stopped
		disconnect()
		<-ended
		<-ended
	})
}
