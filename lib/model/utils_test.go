// Copyright (C) 2019 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/victorarias/syncthing/lib/build"
	"github.com/victorarias/syncthing/lib/fs"
	"github.com/victorarias/syncthing/lib/rand"
)

func TestInWriteableDir(t *testing.T) {
	fs := fs.NewFilesystem(fs.FilesystemTypeFake, rand.String(32))

	fs.Mkdir("testdata", 0o700)
	fs.Mkdir("testdata/rw", 0o700)
	fs.Mkdir("testdata/ro", 0o500)

	create := func(name string) error {
		fd, err := fs.Create(name)
		if err != nil {
			return err
		}
		fd.Close()
		return nil
	}

	// These should succeed

	err := inWritableDir(create, fs, "testdata/file", false)
	if err != nil {
		t.Error("testdata/file:", err)
	}
	err = inWritableDir(create, fs, "testdata/rw/foo", false)
	if err != nil {
		t.Error("testdata/rw/foo:", err)
	}
	err = inWritableDir(fs.Remove, fs, "testdata/rw/foo", false)
	if err != nil {
		t.Error("testdata/rw/foo:", err)
	}

	err = inWritableDir(create, fs, "testdata/ro/foo", false)
	if err != nil {
		t.Error("testdata/ro/foo:", err)
	}
	err = inWritableDir(fs.Remove, fs, "testdata/ro/foo", false)
	if err != nil {
		t.Error("testdata/ro/foo:", err)
	}

	// These should not

	err = inWritableDir(create, fs, "testdata/nonexistent/foo", false)
	if err == nil {
		t.Error("testdata/nonexistent/foo returned nil error")
	}
	err = inWritableDir(create, fs, "testdata/file/foo", false)
	if err == nil {
		t.Error("testdata/file/foo returned nil error")
	}
}

func TestOSWindowsRemove(t *testing.T) {
	if !build.IsWindows {
		t.Skipf("Tests not required")
		return
	}

	fs := fs.NewFilesystem(fs.FilesystemTypeFake, rand.String(32))

	create := func(name string) error {
		fd, err := fs.Create(name)
		if err != nil {
			return err
		}
		fd.Close()
		return nil
	}

	fs.Mkdir("testdata", 0o700)

	fs.Mkdir("testdata/windows", 0o500)
	fs.Mkdir("testdata/windows/ro", 0o500)
	create("testdata/windows/ro/readonly")
	fs.Chmod("testdata/windows/ro/readonly", 0o500)

	for _, path := range []string{"testdata/windows/ro/readonly", "testdata/windows/ro", "testdata/windows"} {
		err := inWritableDir(fs.Remove, fs, path, false)
		if err != nil {
			t.Errorf("Unexpected error %s: %s", path, err)
		}
	}
}

func TestOSWindowsRemoveAll(t *testing.T) {
	if !build.IsWindows {
		t.Skipf("Tests not required")
		return
	}

	fs := fs.NewFilesystem(fs.FilesystemTypeFake, rand.String(32))

	create := func(name string) error {
		fd, err := fs.Create(name)
		if err != nil {
			return err
		}
		fd.Close()
		return nil
	}

	fs.Mkdir("testdata", 0o700)

	fs.Mkdir("testdata/windows", 0o500)
	fs.Mkdir("testdata/windows/ro", 0o500)
	create("testdata/windows/ro/readonly")
	fs.Chmod("testdata/windows/ro/readonly", 0o500)

	if err := fs.RemoveAll("testdata/windows"); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestInWritableDirWindowsRename(t *testing.T) {
	if !build.IsWindows {
		t.Skipf("Tests not required")
		return
	}

	fs := fs.NewFilesystem(fs.FilesystemTypeFake, rand.String(32))

	create := func(name string) error {
		fd, err := fs.Create(name)
		if err != nil {
			return err
		}
		fd.Close()
		return nil
	}

	fs.Mkdir("testdata", 0o700)

	fs.Mkdir("testdata/windows", 0o500)
	fs.Mkdir("testdata/windows/ro", 0o500)
	create("testdata/windows/ro/readonly")
	fs.Chmod("testdata/windows/ro/readonly", 0o500)

	for _, path := range []string{"testdata/windows/ro/readonly", "testdata/windows/ro", "testdata/windows"} {
		err := fs.Rename(path, path+"new")
		if err == nil {
			t.Skipf("seem like this test doesn't work here")
			return
		}
	}

	rename := func(path string) error {
		return fs.Rename(path, path+"new")
	}

	for _, path := range []string{"testdata/windows/ro/readonly", "testdata/windows/ro", "testdata/windows"} {
		err := inWritableDir(rename, fs, path, false)
		if err != nil {
			t.Errorf("Unexpected error %s: %s", path, err)
		}
		_, err = fs.Stat(path + "new")
		if err != nil {
			t.Errorf("Unexpected error %s: %s", path, err)
		}
	}
}

type restoreDeniedFilesystem struct {
	fs.Filesystem
	chmods int
}

func (f *restoreDeniedFilesystem) Chmod(path string, mode fs.FileMode) error {
	f.chmods++
	if f.chmods == 2 {
		return errors.New("restore denied")
	}
	return f.Filesystem.Chmod(path, mode)
}

func TestWritableDirectoryRestoreReportsThroughItsFilesystemLogger(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	filesystem := fs.NewFilesystem(fs.FilesystemTypeBasic, t.TempDir(), &fs.OptionLogger{Logger: logger})
	if err := filesystem.Mkdir("locked", 0o500); err != nil {
		t.Fatal(err)
	}
	failing := &restoreDeniedFilesystem{Filesystem: filesystem}
	if err := inWritableDir(func(string) error { return nil }, failing, "locked/file", false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), "restore denied") {
		t.Fatalf("restore failure missing from embedder log: %s", logs.String())
	}
}
