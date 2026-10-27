// Copyright (C) 2026 The attn Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package attnsync embeds the file synchronization engine. The caller owns
// peer authentication, transport, the folder marker and configuration storage.
package attnsync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/thejerf/suture/v4"
	"github.com/victorarias/syncthing/internal/db/sqlite"
	"github.com/victorarias/syncthing/internal/slogutil"
	"github.com/victorarias/syncthing/lib/config"
	"github.com/victorarias/syncthing/lib/events"
	"github.com/victorarias/syncthing/lib/ignore"
	"github.com/victorarias/syncthing/lib/model"
	"github.com/victorarias/syncthing/lib/protocol"
	"github.com/victorarias/syncthing/lib/svcutil"
)

const Version = "v0.1.5"

type Engine struct {
	ID         protocol.DeviceID
	cfg        config.Wrapper
	model      model.Model
	database   *sqlite.DB
	keys       *protocol.KeyGenerator
	events     events.Logger
	logger     *slog.Logger
	supervisor *suture.Supervisor
}

// Configuration returns defaults for attn's externally managed transport.
func Configuration(id protocol.DeviceID) config.Configuration {
	cfg := config.New(id)
	cfg.GUI.Enabled = false
	o := &cfg.Options
	o.SetLowPriority = false
	o.URAccepted = -1
	o.CREnabled = false
	o.AutoUpgradeIntervalH = 0
	o.GlobalAnnEnabled, o.LocalAnnEnabled, o.RelaysEnabled, o.NATEnabled = false, false, false, false
	o.RawListenAddresses, o.RawStunServers = nil, nil
	cfg.Defaults.Device.RawNumConnections = 1
	f := &cfg.Defaults.Folder
	f.MaxConflicts = -1
	f.FSWatcherDelayS = 1
	f.MarkerName = ".attn/notebook-sync"
	f.IgnorePatterns = []string{"/.attn", ".*.tmp.*", ".seed-transfer-*", "(?d).DS_Store"}
	return cfg
}

// New opens an index directory. Serve owns the engine and closes its database
// when it returns. Each engine receives its own logger; nil discards logs.
func New(indexPath string, cfg config.Configuration, id protocol.DeviceID, logger *slog.Logger) (*Engine, error) {
	cfg = cfg.Copy()
	if err := prepare(&cfg, id, logger); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(indexPath) {
		return nil, fmt.Errorf("index path must be absolute: %q", indexPath)
	}
	logger = slogutil.Logger(logger)
	database, err := sqlite.Open(indexPath, sqlite.WithLogger(logger))
	if err != nil {
		return nil, err
	}
	ev := events.NewLogger(logger)
	wrapper := config.Wrap("", cfg, id, ev, logger)
	keys := protocol.NewKeyGenerator()
	m := model.NewModel(wrapper, id, database, nil, ev, keys, logger)
	sup := suture.New("attnsync", svcutil.SpecWithInfoLogger(logger))
	sup.Add(ev)
	sup.Add(wrapper)
	sup.Add(m)
	sup.Add(database.Service(8 * time.Hour))
	return &Engine{ID: id, cfg: wrapper, model: m, database: database, keys: keys, events: ev, logger: logger, supervisor: sup}, nil
}

func prepare(cfg *config.Configuration, id protocol.DeviceID, logger *slog.Logger) error {
	cfg.SetLogger(logger)
	for i := range cfg.Folders {
		f := &cfg.Folders[i]
		if !filepath.IsAbs(f.Path) {
			return fmt.Errorf("folder %s path must be absolute: %q", f.ID, f.Path)
		}
		if f.MarkerName == "" {
			f.MarkerName = ".attn/notebook-sync"
		}
		if !filepath.IsLocal(f.MarkerName) || f.MarkerName == config.DefaultMarkerName {
			return fmt.Errorf("folder %s marker must be a custom path under the folder: %q", f.ID, f.MarkerName)
		}
		if err := ignore.New(f.Filesystem()).ParseMemory(f.IgnorePatterns); err != nil {
			return fmt.Errorf("folder %s ignores: %w", f.ID, err)
		}
	}
	return cfg.Prepare(id)
}

func (e *Engine) Serve(ctx context.Context) error {
	defer e.database.Close()
	return e.supervisor.Serve(ctx)
}

func (e *Engine) Ready() <-chan struct{} { return e.model.Started() }

// Configure replaces the in-memory configuration and waits for application.
func (e *Engine) Configure(cfg config.Configuration) error {
	cfg = cfg.Copy()
	if err := prepare(&cfg, e.ID, e.logger); err != nil {
		return err
	}
	waiter, err := e.cfg.Modify(func(current *config.Configuration) { *current = cfg })
	if err != nil {
		return err
	}
	waiter.Wait()
	return nil
}

func (e *Engine) Subscribe(mask events.EventType) events.Subscription {
	return e.events.Subscribe(mask)
}

func (e *Engine) SubscribeContext(ctx context.Context, mask events.EventType) (events.Subscription, error) {
	return e.events.SubscribeContext(ctx, mask)
}

type FolderState struct {
	State     string
	Changed   time.Time
	NeedFiles int
	NeedBytes int64
	Error     error
}

func (e *Engine) FolderState(folder string) (FolderState, error) {
	state, changed, health := e.model.State(folder)
	need, err := e.model.NeedSize(folder, protocol.LocalDeviceID)
	if err != nil {
		return FolderState{}, err
	}
	return FolderState{State: state, Changed: changed, NeedFiles: need.Files, NeedBytes: need.Bytes, Error: health}, nil
}

func (e *Engine) Completion(peer protocol.DeviceID, folder string) (model.FolderCompletion, error) {
	return e.model.Completion(peer, folder)
}

// ServeConn takes ownership of an authenticated byte stream. It adds no TLS,
// listener, dialer, retry or certificate exchange. Cancellation closes the stream,
// including while the peer has not completed the protocol hello.
func (e *Engine) ServeConn(ctx context.Context, conn net.Conn, peer protocol.DeviceID) error {
	defer conn.Close()
	device, ok := e.cfg.Device(peer)
	if !ok || peer == e.ID {
		return fmt.Errorf("unknown peer %s", peer)
	}
	if device.Paused {
		return fmt.Errorf("peer %s is paused", peer)
	}
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	select {
	case <-e.Ready():
	case <-ctx.Done():
		return ctx.Err()
	}
	hello := protocol.Hello{ClientName: "attn", ClientVersion: Version, Timestamp: time.Now().UnixNano(), NumConnections: 1}
	if me, ok := e.cfg.Device(e.ID); ok {
		hello.DeviceName = me.Name
	}
	theirs, err := protocol.ExchangeHello(conn, hello)
	if err != nil {
		return err
	}
	if err := e.model.OnHello(peer, conn.RemoteAddr(), theirs); err != nil {
		return err
	}
	info := connectionInfo{Conn: conn, id: fmt.Sprintf("%x-%x", hello.Timestamp, theirs.Timestamp), at: time.Now()}
	pc := protocol.NewConnection(peer, conn, conn, conn, e.model, info, device.Compression.ToProtocol(), e.keys, e.logger)
	closed := e.model.AddConnection(pc, theirs)
	defer func() { <-closed }()
	select {
	case <-pc.Closed():
		return errors.New("connection closed")
	case <-ctx.Done():
		conn.Close()
		pc.Close(ctx.Err())
		<-pc.Closed()
		return ctx.Err()
	}
}

type connectionInfo struct {
	net.Conn
	id string
	at time.Time
}

func (c connectionInfo) Type() string             { return "attn-bridge" }
func (c connectionInfo) Transport() string        { return "attn-bridge" }
func (c connectionInfo) IsLocal() bool            { return false }
func (c connectionInfo) Priority() int            { return 0 }
func (c connectionInfo) String() string           { return "attn-bridge " + c.RemoteAddr().String() }
func (c connectionInfo) Crypto() string           { return "authenticated by caller" }
func (c connectionInfo) EstablishedAt() time.Time { return c.at }
func (c connectionInfo) ConnectionID() string     { return c.id }

// IsConflict reports whether a file name is a Syncthing conflict copy.
func IsConflict(path string) bool { return strings.Contains(filepath.Base(path), ".sync-conflict-") }

type FolderSnapshot struct {
	FolderState
	ScanComplete    bool
	LocalSequence   int64
	PeerSequence    int64
	NeedItems       int
	NeedDeletes     int
	PeerNeedItems   int
	PeerNeedDeletes int
	PeerNeedBytes   int64
	Errors          []model.FileError
}

func (e *Engine) Snapshot(folder string, peer protocol.DeviceID) (FolderSnapshot, error) {
	var snapshot FolderSnapshot
	var err error
	if snapshot.ScanComplete, err = e.model.ScanComplete(folder); err != nil {
		return snapshot, err
	}
	if snapshot.LocalSequence, err = e.model.Sequence(folder, protocol.LocalDeviceID); err != nil {
		return snapshot, err
	}
	if snapshot.PeerSequence, err = e.model.Sequence(folder, peer); err != nil {
		return snapshot, err
	}
	local, err := e.model.Completion(e.ID, folder)
	if err != nil {
		return snapshot, err
	}
	remote, err := e.model.Completion(peer, folder)
	if err != nil {
		return snapshot, err
	}
	if snapshot.FolderState, err = e.FolderState(folder); err != nil {
		return snapshot, err
	}
	if snapshot.Errors, err = e.model.FolderErrors(folder); err != nil {
		return snapshot, err
	}
	snapshot.NeedItems, snapshot.NeedDeletes = local.NeedItems, local.NeedDeletes
	snapshot.PeerNeedItems, snapshot.PeerNeedDeletes, snapshot.PeerNeedBytes = remote.NeedItems, remote.NeedDeletes, remote.NeedBytes
	return snapshot, nil
}
