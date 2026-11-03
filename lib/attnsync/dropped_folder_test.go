// Copyright (C) 2026 The attn Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package attnsync_test

import (
	"context"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/victorarias/syncthing/lib/events"
	"github.com/victorarias/syncthing/lib/protocol"
)

// gatedConn holds reads while closed, so the test can keep one direction of
// a connection from delivering what the other side wrote.
type gatedConn struct {
	net.Conn
	mu   sync.Mutex
	open chan struct{}
}

func (g *gatedConn) hold() { g.mu.Lock(); g.open = make(chan struct{}); g.mu.Unlock() }
func (g *gatedConn) release() {
	g.mu.Lock()
	if g.open != nil {
		close(g.open)
		g.open = nil
	}
	g.mu.Unlock()
}
func (g *gatedConn) Read(p []byte) (int, error) {
	g.mu.Lock()
	open := g.open
	g.mu.Unlock()
	if open != nil {
		<-open
	}
	return g.Conn.Read(p)
}

// logSignal closes seen when a log record containing want is handled.
type logSignal struct {
	slog.Handler
	want string
	once sync.Once
	seen chan struct{}
}

func (h *logSignal) Enabled(context.Context, slog.Level) bool { return true }
func (h *logSignal) Handle(ctx context.Context, r slog.Record) error {
	if strings.Contains(r.Message, h.want) {
		h.once.Do(func() { close(h.seen) })
	}
	return h.Handler.Handle(ctx, r)
}

// B removes a folder while A, not yet told, sends an index update for it. B
// drops the update and keeps the connection; once B adds the folder again,
// the file A wrote in between reaches B.
func TestIndexForAFolderTheReceiverRemovedKeepsTheConnection(t *testing.T) {
	aID, bID := protocol.NewDeviceID([]byte("a")), protocol.NewDeviceID([]byte("b"))
	aCfg, aRoot := configuration(t, aID, bID)
	bCfg, bRoot := configuration(t, bID, aID)
	aLog, bLog := &capture{}, &capture{}
	a := start(t, aCfg, aID, slog.New(slog.NewTextHandler(aLog, &slog.HandlerOptions{Level: slog.LevelDebug})))
	dropped := &logSignal{
		Handler: slog.NewTextHandler(bLog, &slog.HandlerOptions{Level: slog.LevelDebug}),
		want:    `"notebook": dropped`,
		seen:    make(chan struct{}),
	}
	b := start(t, bCfg, bID, slog.New(dropped))
	defer func() {
		if t.Failed() {
			t.Logf("A log:\n%s\nB log:\n%s", aLog.String(), bLog.String())
		}
	}()
	bEvents := b.Subscribe(events.ItemFinished)
	defer bEvents.Unsubscribe()

	// A reads what B writes through the gate.
	left, right := net.Pipe()
	gate := &gatedConn{Conn: left}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	aEnded, bEnded := make(chan struct{}), make(chan struct{})
	var bErr error
	go func() { _ = a.ServeConn(ctx, gate, bID); close(aEnded) }()
	go func() { bErr = b.ServeConn(ctx, right, aID); close(bEnded) }()
	defer func() { cancel(); gate.release(); <-aEnded; <-bEnded }()

	write(t, filepath.Join(aRoot, "first.md"), "first")
	waitForItem(t, bEvents, bEnded, &bErr, "first.md")

	// Hold B's ClusterConfig without the folder, so A keeps sending for it.
	gate.hold()
	removed := bCfg.Copy()
	removed.Folders = nil
	if err := b.Configure(removed); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(aRoot, "second.md"), "second")
	select {
	case <-dropped.seen:
	case <-bEnded:
		t.Fatalf("B's connection ended on an index update for a folder it removed: %v", bErr)
	}
	gate.release()

	// A learns that B shares the folder again only from B's ClusterConfig,
	// and B learns nothing new from A: A's earlier ClusterConfig still
	// announces the folder.
	if err := b.Configure(bCfg); err != nil {
		t.Fatal(err)
	}
	waitForItem(t, bEvents, bEnded, &bErr, "second.md")
	content, err := os.ReadFile(filepath.Join(bRoot, "second.md"))
	if err != nil || string(content) != "second" {
		t.Fatalf("second.md on B: %q, %v", content, err)
	}
}

func waitForItem(t *testing.T, sub events.Subscription, ended <-chan struct{}, err *error, item string) {
	t.Helper()
	for {
		select {
		case event := <-sub.C():
			if event.Data.(map[string]any)["item"] == item {
				return
			}
		case <-ended:
			t.Fatalf("connection ended while waiting for %s: %v", item, *err)
		}
	}
}
