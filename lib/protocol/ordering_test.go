// Copyright (C) 2026 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package protocol

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/victorarias/syncthing/lib/testutil"
)

// gatedWriter blocks every write until release is closed, and closes
// writing when the first write arrives.
type gatedWriter struct {
	w       io.Writer
	once    sync.Once
	writing chan struct{}
	release chan struct{}
}

func newGatedWriter(w io.Writer) *gatedWriter {
	return &gatedWriter{w: w, writing: make(chan struct{}), release: make(chan struct{})}
}

func (g *gatedWriter) Write(p []byte) (int, error) {
	g.once.Do(func() { close(g.writing) })
	<-g.release
	return g.w.Write(p)
}

// orderingPair connects a sender to a receiver whose model reports each
// ClusterConfig and Index it receives, in order, on events.
func orderingPair(t *testing.T, senderWriter func(io.Writer) io.Writer) (*rawConnection, <-chan string) {
	t.Helper()
	events := make(chan string, 16)
	receiver := newTestModel()
	receiver.ccFn = func(cc *ClusterConfig) { events <- describeClusterConfig(cc) }
	receiver.indexFn = func(folder string, _ []FileInfo) { events <- "index " + folder }

	ar, aw := io.Pipe()
	br, bw := io.Pipe()
	sender := getRawConnection(NewConnection(c0ID, ar, senderWriter(bw), testutil.NoopCloser{}, newTestModel(), new(mockedConnectionInfo), CompressionNever, testKeyGen))
	sender.Start()
	t.Cleanup(func() { closeAndWait(sender, ar, bw) })
	recv := NewConnection(c1ID, br, aw, testutil.NoopCloser{}, receiver, new(mockedConnectionInfo), CompressionNever, testKeyGen)
	recv.Start()
	t.Cleanup(func() { closeAndWait(recv, ar, bw) })
	return sender, events
}

// describeClusterConfig names each folder with its state, for example
// "cc F:running G:paused".
func describeClusterConfig(cc *ClusterConfig) string {
	parts := []string{"cc"}
	for _, f := range cc.Folders {
		state := "running"
		if f.StopReason != FolderStopReasonRunning {
			state = "paused"
		}
		parts = append(parts, f.ID+":"+state)
	}
	return strings.Join(parts, " ")
}

func clusterConfigWith(folder string, stop FolderStopReason) *ClusterConfig {
	return &ClusterConfig{Folders: []Folder{{ID: folder, StopReason: stop}}}
}

func TestClusterConfigsReachTheWireInOrder(t *testing.T) {
	var gate *gatedWriter
	sender, events := orderingPair(t, func(w io.Writer) io.Writer {
		gate = newGatedWriter(w)
		return gate
	})

	// v0 occupies the writer; v1 to v3 queue behind it. ClusterConfig must
	// not block while the writer is stuck.
	sender.ClusterConfig(clusterConfigWith("v0", FolderStopReasonRunning), nil)
	<-gate.writing
	for v := 1; v <= 3; v++ {
		sender.ClusterConfig(clusterConfigWith(fmt.Sprintf("v%d", v), FolderStopReasonRunning), nil)
	}
	close(gate.release)

	var seen []string
	last := -1
	for last != 3 {
		ev := <-events
		seen = append(seen, ev)
		var v int
		if _, err := fmt.Sscanf(ev, "cc v%d:running", &v); err != nil {
			t.Fatalf("unexpected message %q; received so far %v", ev, seen)
		}
		if v <= last {
			t.Fatalf("ClusterConfig v%d written after v%d; received %v", v, last, seen)
		}
		last = v
	}
	t.Logf("received %v", seen)
}

func TestIndexWaitsForTheClusterConfigThatAnnouncesItsFolder(t *testing.T) {
	sender, events := orderingPair(t, func(w io.Writer) io.Writer { return w })

	expect := func(want string) {
		t.Helper()
		if got := <-events; got != want {
			t.Fatalf("received %q, want %q", got, want)
		}
	}

	sender.ClusterConfig(&ClusterConfig{}, nil)
	expect("cc")

	// The index is queued before any ClusterConfig names F.
	indexErr := make(chan error, 1)
	go func() { indexErr <- sender.Index(context.Background(), &Index{Folder: "F"}) }()

	// Announcing F as paused is not enough.
	sender.ClusterConfig(clusterConfigWith("F", FolderStopReasonPaused), nil)
	expect("cc F:paused")
	select {
	case err := <-indexErr:
		t.Fatalf("Index returned (%v) before F was announced running", err)
	default:
	}

	sender.ClusterConfig(clusterConfigWith("F", FolderStopReasonRunning), nil)
	expect("cc F:running")
	expect("index F")
	if err := <-indexErr; err != nil {
		t.Fatal(err)
	}
}

func TestIndexWaitingForItsFolderEndsWithTheConnection(t *testing.T) {
	sender, _ := orderingPair(t, func(w io.Writer) io.Writer { return w })
	sender.ClusterConfig(&ClusterConfig{}, nil)

	indexErr := make(chan error, 1)
	go func() { indexErr <- sender.Index(context.Background(), &Index{Folder: "F"}) }()
	sender.internalClose(errManual)
	if err := <-indexErr; err != ErrClosed {
		t.Fatalf("Index returned %v, want %v", err, ErrClosed)
	}
}
