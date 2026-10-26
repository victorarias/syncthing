// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build cgo

package sqlite

import (
	"database/sql"

	"github.com/mattn/go-sqlite3" // register sqlite3 database driver
)

const (
	dbDriver      = "attnsync-sqlite3"
	commonOptions = "_fk=true&_rt=true&_sync=1&_txlock=immediate"
)

func init() {
	sql.Register(dbDriver, &sqlite3.SQLiteDriver{ConnectHook: func(conn *sqlite3.SQLiteConn) error {
		_, err := conn.Exec("PRAGMA temp_store=MEMORY", nil)
		return err
	}})
}
