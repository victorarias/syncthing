// Copyright (C) 2026 The attn Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package attnsync_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/victorarias/syncthing/lib/attnsync"
	"github.com/victorarias/syncthing/lib/config"
	"github.com/victorarias/syncthing/lib/events"
	"github.com/victorarias/syncthing/lib/protocol"
)

func TestCallerOwnedStreamAndMemoryIgnores(t *testing.T) {
	original := slog.Default()
	aID, bID := protocol.NewDeviceID([]byte("a")), protocol.NewDeviceID([]byte("b"))
	aCfg, aRoot := configuration(t, aID, bID)
	bCfg, bRoot := configuration(t, bID, aID)
	// A disk ignore file must not change the embedder's patterns.
	write(t, filepath.Join(aRoot, ".stignore"), "shared.md\n")
	write(t, filepath.Join(aRoot, "shared.md"), "hello")
	write(t, filepath.Join(aRoot, "private.md"), "ignored")
	aCfg.Folders[0].IgnorePatterns = append(aCfg.Folders[0].IgnorePatterns, "/private.md", "/.stignore")
	bCfg.Folders[0].IgnorePatterns = append(bCfg.Folders[0].IgnorePatterns, "/private.md", "/.stignore")
	a := start(t, aCfg, aID)
	b := start(t, bCfg, bID)
	if slog.Default() != original {
		t.Fatal("engine replaced the embedder's default logger")
	}
	aEvents, bEvents := a.Subscribe(events.AllEvents), b.Subscribe(events.AllEvents)
	defer bEvents.Unsubscribe()
	// Drain A, while waiting on B, so event delivery never depends on a timeout.
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		for range aEvents.C() {
		}
	}()
	write(t, filepath.Join(aRoot, "later-private.md"), "configured ignore")
	aCfg.Folders[0].IgnorePatterns = append(aCfg.Folders[0].IgnorePatterns, "/later-private.md")
	if err := a.Configure(aCfg); err != nil {
		t.Fatal(err)
	}
	invalid := aCfg.Copy()
	invalid.Folders[0].IgnorePatterns = []string{"foo\n#include outside"}
	if err := a.Configure(invalid); err == nil {
		t.Fatal("included disk file through Configure")
	}
	left, right := net.Pipe()
	connCtx, cancel := context.WithCancel(context.Background())
	ended := make(chan error, 2)
	go func() { ended <- a.ServeConn(connCtx, left, bID) }()
	go func() { ended <- b.ServeConn(connCtx, right, aID) }()
	for event := range bEvents.C() {
		if event.Type == events.ItemFinished {
			data := event.Data.(map[string]any)
			if data["item"] == "shared.md" {
				break
			}
		}
	}
	content, err := os.ReadFile(filepath.Join(bRoot, "shared.md"))
	if err != nil || string(content) != "hello" {
		t.Fatalf("synced file: %q, %v", content, err)
	}
	if _, err := os.Stat(filepath.Join(bRoot, "private.md")); !os.IsNotExist(err) {
		t.Fatalf("ignored file arrived: %v", err)
	}
	if _, err := os.Stat(filepath.Join(bRoot, "later-private.md")); !os.IsNotExist(err) {
		t.Fatalf("configured ignore arrived: %v", err)
	}
	for _, root := range []string{aRoot, bRoot} {
		if _, err := os.Stat(filepath.Join(root, ".stfolder")); !os.IsNotExist(err) {
			t.Fatalf("default marker created: %v", err)
		}
	}
	if _, err := os.Stat(filepath.Join(bRoot, ".stignore")); !os.IsNotExist(err) {
		t.Fatalf("disk ignores created: %v", err)
	}
	cancel()
	<-ended
	<-ended
	aEvents.Unsubscribe()
	<-drained
}

func TestUnknownPeerAndCancelledHello(t *testing.T) {
	id, peer := protocol.NewDeviceID([]byte("a")), protocol.NewDeviceID([]byte("b"))
	cfg, _ := configuration(t, id, peer)
	engine := start(t, cfg, id)
	a, b := net.Pipe()
	err := engine.ServeConn(context.Background(), a, protocol.NewDeviceID([]byte("unknown")))
	if err == nil || !strings.Contains(err.Error(), "unknown peer") {
		t.Fatalf("unknown peer: %v", err)
	}
	if _, err := b.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("refused connection remains open: %v", err)
	}
	b.Close()
	a, b = net.Pipe()
	defer b.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := engine.ServeConn(ctx, a, peer); err == nil {
		t.Fatal("cancelled hello succeeded")
	}
}

func TestRejectIncludeBeforeOpeningState(t *testing.T) {
	id, peer := protocol.NewDeviceID([]byte("a")), protocol.NewDeviceID([]byte("b"))
	cfg, _ := configuration(t, id, peer)
	cfg.Folders[0].IgnorePatterns = []string{"foo\n#include /outside"}
	state := filepath.Join(t.TempDir(), "index")
	_, err := attnsync.New(state, cfg, id, nil)
	if err == nil || !strings.Contains(err.Error(), "#include") {
		t.Fatalf("include: %v", err)
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("invalid config created state: %v", err)
	}
}

func configuration(t *testing.T, id, peer protocol.DeviceID) (config.Configuration, string) {
	t.Helper()
	root := t.TempDir()
	write(t, filepath.Join(root, ".attn/notebook-sync"), "")
	cfg := attnsync.Configuration(id)
	d := cfg.Defaults.Device.Copy()
	d.DeviceID = peer
	cfg.Devices = append(cfg.Devices, d)
	f := cfg.Defaults.Folder.Copy()
	f.ID = "notebook"
	f.Path = root
	f.Devices = []config.FolderDeviceConfiguration{{DeviceID: id}, {DeviceID: peer}}
	cfg.Folders = []config.FolderConfiguration{f}
	return cfg, root
}
func start(t *testing.T, cfg config.Configuration, id protocol.DeviceID, loggers ...*slog.Logger) *attnsync.Engine {
	t.Helper()
	var logger *slog.Logger
	if len(loggers) > 0 {
		logger = loggers[0]
	}
	engine, err := attnsync.New(filepath.Join(t.TempDir(), "index"), cfg, id, logger)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- engine.Serve(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	<-engine.Ready()
	return engine
}
func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

type capture struct {
	mu    sync.Mutex
	bytes bytes.Buffer
}

func (c *capture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bytes.Write(p)
}
func (c *capture) String() string { c.mu.Lock(); defer c.mu.Unlock(); return c.bytes.String() }

func TestEngineLoggersStayIndependent(t *testing.T) {
	id, peer := protocol.NewDeviceID([]byte("a")), protocol.NewDeviceID([]byte("b"))
	aCfg, _ := configuration(t, id, peer)
	aCfg.Folders[0].ID = "folder-a"
	bCfg, _ := configuration(t, peer, id)
	bCfg.Folders[0].ID = "folder-b"
	var a, b capture
	start(t, aCfg, id, slog.New(slog.NewJSONHandler(&a, nil)))
	start(t, bCfg, peer, slog.New(slog.NewJSONHandler(&b, nil)))
	if !strings.Contains(a.String(), "folder-a") || strings.Contains(a.String(), "folder-b") {
		t.Fatalf("A logs: %s", a.String())
	}
	if !strings.Contains(b.String(), "folder-b") || strings.Contains(b.String(), "folder-a") {
		t.Fatalf("B logs: %s", b.String())
	}
}

func TestConfigureAfterShutdown(t *testing.T) {
	id, peer := protocol.NewDeviceID([]byte("a")), protocol.NewDeviceID([]byte("b"))
	cfg, _ := configuration(t, id, peer)
	engine, err := attnsync.New(filepath.Join(t.TempDir(), "index"), cfg, id, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- engine.Serve(ctx) }()
	<-engine.Ready()
	cancel()
	<-done
	if err := engine.Configure(cfg); err == nil {
		t.Fatal("stopped engine accepted configuration")
	}
}

func TestCancelBeforeProtocolPromotion(t *testing.T) {
	aID, bID := protocol.NewDeviceID([]byte("a")), protocol.NewDeviceID([]byte("b"))
	aCfg, _ := configuration(t, aID, bID)
	bCfg, _ := configuration(t, bID, aID)
	a, b := start(t, aCfg, aID), start(t, bCfg, bID)
	sub := a.Subscribe(events.DeviceConnected)
	defer sub.Unsubscribe()
	left, right := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ended := make(chan error, 2)
	go func() { ended <- a.ServeConn(ctx, left, bID) }()
	go func() { ended <- b.ServeConn(ctx, right, aID) }()
	<-sub.C()
	at := time.Now()
	cancel()
	<-ended
	<-ended
	if elapsed := time.Since(at); elapsed >= 100*time.Millisecond {
		t.Fatalf("cancel before protocol promotion took %v, want <100ms", elapsed)
	}
}

func TestSnapshotDistinguishesAnEmptyScanFromAnUnknownFolderAndKeepsErrorsCurrent(t *testing.T) {
	id, peer := protocol.NewDeviceID([]byte("snapshot-local")), protocol.NewDeviceID([]byte("snapshot-peer"))
	cfg, root := configuration(t, id, peer)
	marker := filepath.Join(root, ".attn/notebook-sync")
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	engine := start(t, cfg, id)
	sub := engine.Subscribe(events.AllEvents)
	defer sub.Unsubscribe()
	await := func(matches func(attnsync.FolderSnapshot) bool) attnsync.FolderSnapshot {
		t.Helper()
		for {
			snapshot, err := engine.Snapshot("notebook", peer)
			if err == nil && matches(snapshot) {
				return snapshot
			}
			if _, ok := <-sub.C(); !ok {
				t.Fatalf("events ended: %+v, %v", snapshot, err)
			}
		}
	}
	failed := await(func(s attnsync.FolderSnapshot) bool { return s.Error != nil })
	if failed.ScanComplete {
		t.Fatal("missing marker counted as a completed scan")
	}
	restartFolder := func() {
		t.Helper()
		cfg.Folders[0].Paused = true
		if err := engine.Configure(cfg); err != nil {
			t.Fatal(err)
		}
		cfg.Folders[0].Paused = false
		if err := engine.Configure(cfg); err != nil {
			t.Fatal(err)
		}
	}
	write(t, marker, "")
	restartFolder()
	empty := await(func(s attnsync.FolderSnapshot) bool { return s.ScanComplete && s.State == "idle" })
	if empty.LocalSequence != 0 || empty.NeedItems != 0 || empty.NeedDeletes != 0 || empty.Error != nil || len(empty.Errors) != 0 {
		t.Fatalf("empty scan: %+v", empty)
	}
	blocked := filepath.Join(root, "blocked.md")
	write(t, blocked, "cannot hash this")
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0600) })
	restartFolder()
	failure := await(func(s attnsync.FolderSnapshot) bool { return len(s.Errors) != 0 })
	if failure.Errors[0].Path != "blocked.md" || !strings.Contains(failure.Errors[0].Err, "permission denied") {
		t.Fatalf("file error missing its reason: %+v", failure)
	}
	if err := os.Chmod(blocked, 0600); err != nil {
		t.Fatal(err)
	}
	restartFolder()
	repaired := await(func(s attnsync.FolderSnapshot) bool { return s.ScanComplete && s.State == "idle" && len(s.Errors) == 0 })
	if repaired.Error != nil || repaired.LocalSequence == 0 {
		t.Fatalf("repaired scan: %+v", repaired)
	}
}

func TestSubscriptionCancellationBeforeServeAndAfterShutdown(t *testing.T) {
	id := protocol.NewDeviceID([]byte("subscription-cancellation"))
	cfg, _ := configuration(t, id, protocol.NewDeviceID([]byte("peer")))
	engine, err := attnsync.New(t.TempDir(), cfg, id, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if sub, err := engine.SubscribeContext(ctx, events.AllEvents); sub != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("before Serve: %v, %v", sub, err)
	}
	serving, stop := context.WithCancel(context.Background())
	ended := make(chan error, 1)
	go func() { ended <- engine.Serve(serving) }()
	<-engine.Ready()
	sub, err := engine.SubscribeContext(serving, events.AllEvents)
	if err != nil {
		t.Fatal(err)
	}
	stop()
	if err := <-ended; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	sub.Unsubscribe()
	if sub, err := engine.SubscribeContext(serving, events.AllEvents); sub != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("after shutdown: %v, %v", sub, err)
	}
}

func TestEngineShutdownLeavesNoBackgroundWork(t *testing.T) {
	id := protocol.NewDeviceID([]byte("shutdown"))
	cfg, _ := configuration(t, id, protocol.NewDeviceID([]byte("peer")))
	synctest.Test(t, func(t *testing.T) {
		engine, err := attnsync.New(t.TempDir(), cfg, id, nil)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- engine.Serve(ctx) }()
		<-engine.Ready()
		cancel()
		<-done
	})
}
