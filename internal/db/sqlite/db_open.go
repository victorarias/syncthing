// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package sqlite

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/victorarias/syncthing/internal/db"
	"github.com/victorarias/syncthing/internal/slogutil"
)

const (
	// maxIdleConns is set so that most db operations will usually fit
	// within one of those connections and hence use their persistent page
	// cache. Additional connections on top of these will allocate their own
	// page cache (same as any other), but it will be deallocated on
	// connection close.
	maxIdleConns = 4

	minDeleteRetention = 24 * time.Hour
)

// maxOpenConns is sized for handling spikes. The primary driver is the
// Copiers folder option which may result in up to 2*NumCPU concurrent
// iterations. We reserve additional space on top of this to serve
// additional operations, some of which may be reentrant (queries within
// iterators) without deadlock.
var maxOpenConns = max(16, 4*runtime.NumCPU())

type DB struct {
	*baseDB

	pathBase        string
	deleteRetention time.Duration

	sl             *slog.Logger
	folderDBsMut   sync.RWMutex
	folderDBs      map[string]*folderDB
	folderDBOpener func(folder, path string, deleteRetention time.Duration) (*folderDB, error)
}

var _ db.DB = (*DB)(nil)

type Option func(*DB)

func WithLogger(logger *slog.Logger) Option { return func(s *DB) { s.sl = slogutil.Logger(logger) } }

func WithDeleteRetention(d time.Duration) Option {
	return func(s *DB) {
		if d <= 0 {
			s.deleteRetention = 0
		} else {
			s.deleteRetention = max(d, minDeleteRetention)
		}
	}
}

func Open(path string, opts ...Option) (*DB, error) {
	settings := &DB{}
	for _, opt := range opts {
		opt(settings)
	}
	pragmas := []string{
		"journal_mode = WAL",
		"optimize = 0x10002",
		"auto_vacuum = INCREMENTAL",
		fmt.Sprintf("application_id = %d", applicationIDMain),
	}
	schemas := []string{
		"sql/schema/common/*",
		"sql/schema/main/*",
	}
	migrations := []string{
		"sql/migrations/common/*",
		"sql/migrations/main/*",
	}

	_ = os.MkdirAll(path, 0o700)

	mainPath := filepath.Join(path, "main.db")
	mainBase, err := openBase(mainPath, maxOpenConns, maxIdleConns, pragmas, schemas, migrations, settings.sl)
	if err != nil {
		return nil, err
	}

	db := &DB{
		sl:        settings.sl,
		pathBase:  path,
		baseDB:    mainBase,
		folderDBs: make(map[string]*folderDB),
		folderDBOpener: func(folder, path string, retention time.Duration) (*folderDB, error) {
			return openFolderDBWithLogger(folder, path, retention, settings.sl)
		},
	}

	for _, opt := range opts {
		opt(db)
	}

	if err := db.cleanDroppedFolders(); err != nil {
		slogutil.Logger(db.sl).Warn("Failed to clean dropped folders", slogutil.Error(err))
	}

	if err := db.startFolderDatabases(); err != nil {
		return nil, wrap(err)
	}

	return db, nil
}

// Open the database with options suitable for the migration inserts. This
// is not a safe mode of operation for normal processing, use only for bulk
// inserts with a close afterwards.
func OpenForMigration(path string) (*DB, error) {
	pragmas := []string{
		"journal_mode = OFF",
		"foreign_keys = 0",
		"synchronous = 0",
		"locking_mode = EXCLUSIVE",
		fmt.Sprintf("application_id = %d", applicationIDMain),
	}
	schemas := []string{
		"sql/schema/common/*",
		"sql/schema/main/*",
	}
	migrations := []string{
		"sql/migrations/common/*",
		"sql/migrations/main/*",
	}

	_ = os.MkdirAll(path, 0o700)

	mainPath := filepath.Join(path, "main.db")
	mainBase, err := openBase(mainPath, 1, 1, pragmas, schemas, migrations)
	if err != nil {
		return nil, err
	}

	db := &DB{
		pathBase:       path,
		baseDB:         mainBase,
		folderDBs:      make(map[string]*folderDB),
		folderDBOpener: openFolderDBForMigration,
	}

	if err := db.cleanDroppedFolders(); err != nil {
		slogutil.Logger(db.sl).Warn("Failed to clean dropped folders", slogutil.Error(err))
	}

	return db, nil
}

func (s *DB) Close() error {
	s.folderDBsMut.Lock()
	defer s.folderDBsMut.Unlock()
	for folder, fdb := range s.folderDBs {
		fdb.Close()
		delete(s.folderDBs, folder)
	}
	return wrap(s.baseDB.Close())
}
