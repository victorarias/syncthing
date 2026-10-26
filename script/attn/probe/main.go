// Copyright (C) 2026 The attn Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/victorarias/syncthing/lib/attnsync"
	"github.com/victorarias/syncthing/lib/config"
	"github.com/victorarias/syncthing/lib/events"
	"github.com/victorarias/syncthing/lib/protocol"
)

func main() {
	folder := flag.String("folder", "", "folder")
	state := flag.String("state", "", "state")
	name := flag.String("name", "a", "name")
	peer := flag.String("peer", "b", "peer")
	timeout := flag.Float64("watch-timeout", 0, "watch timeout")
	flag.Parse()
	must(os.MkdirAll(*state, 0700))
	must(os.MkdirAll(filepath.Join(*folder, ".attn"), 0700))
	must(os.WriteFile(filepath.Join(*folder, ".attn/notebook-sync"), nil, 0600))
	id := protocol.NewDeviceID([]byte(*name))
	pid := protocol.NewDeviceID([]byte(*peer))
	cfg := attnsync.Configuration(id)
	d := cfg.Defaults.Device.Copy()
	d.DeviceID = pid
	d.Name = *peer
	cfg.Devices = append(cfg.Devices, d)
	f := cfg.Defaults.Folder.Copy()
	f.ID = "notebook"
	f.Path = *folder
	f.Label = "Notebook"
	f.FSWatcherTimeoutS = *timeout
	f.Devices = []config.FolderDeviceConfiguration{{DeviceID: id}, {DeviceID: pid}}
	cfg.Folders = []config.FolderConfiguration{f}
	log, err := os.Create(filepath.Join(*state, "engine.log"))
	must(err)
	defer log.Close()
	eng, err := attnsync.New(filepath.Join(*state, "index"), cfg, id, slog.New(slog.NewTextHandler(log, &slog.HandlerOptions{Level: slog.LevelWarn})))
	must(err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- eng.Serve(ctx) }()
	<-eng.Ready()
	var out sync.Mutex
	send := func(v any) { out.Lock(); defer out.Unlock(); must(json.NewEncoder(os.Stdout).Encode(v)) }
	sub := eng.Subscribe(events.AllEvents)
	go func() {
		for ev := range sub.C() {
			conflict := false
			switch d := ev.Data.(type) {
			case map[string]string:
				conflict = attnsync.IsConflict(d["path"]) || attnsync.IsConflict(d["item"])
			case map[string]any:
				for _, key := range []string{"path", "item"} {
					if path, ok := d[key].(string); ok {
						conflict = conflict || attnsync.IsConflict(path)
					}
				}
			}
			send(map[string]any{"kind": "event", "type": ev.Type.String(), "time": ev.Time.UnixNano(), "data": ev.Data, "conflict": conflict})
		}
	}()
	var conn net.Conn
	var mu sync.Mutex
	serve := func(c net.Conn) {
		mu.Lock()
		conn = c
		mu.Unlock()
		go func() {
			err := eng.ServeConn(ctx, c, pid)
			send(map[string]any{"kind": "conn_done", "error": fmt.Sprint(err), "time": time.Now().UnixNano()})
		}()
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	must(err)
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			serve(c)
		}
	}()
	send(map[string]any{"kind": "ready", "address": ln.Addr().String(), "id": id.String()})
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var cmd struct {
			Op      string  `json:"op"`
			Address string  `json:"address"`
			Timeout float64 `json:"timeout"`
		}
		must(json.Unmarshal(scanner.Bytes(), &cmd))
		switch cmd.Op {
		case "dial":
			c, err := net.Dial("tcp", cmd.Address)
			must(err)
			serve(c)
			send(map[string]any{"kind": "dialed"})
		case "cut":
			mu.Lock()
			at := time.Now().UnixNano()
			if conn != nil {
				conn.Close()
			}
			mu.Unlock()
			send(map[string]any{"kind": "cut", "time": at})
		case "state":
			s, err := eng.FolderState("notebook")
			c, ce := eng.Completion(pid, "notebook")
			send(map[string]any{"kind": "state", "state": s, "completion": c, "error": fmt.Sprint(err), "completion_error": fmt.Sprint(ce)})
		case "configure":
			cfg.Folders[0].FSWatcherTimeoutS = cmd.Timeout
			err := eng.Configure(cfg)
			send(map[string]any{"kind": "configured", "error": fmt.Sprint(err)})
		case "stop":
			cancel()
			ln.Close()
			err := <-done
			send(map[string]any{"kind": "stopped", "error": fmt.Sprint(err)})
			return
		}
	}
	cancel()
	ln.Close()
	<-done
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
