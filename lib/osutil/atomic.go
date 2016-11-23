// Copyright (C) 2015 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package osutil

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var (
	ErrClosed  = errors.New("write to closed writer")
	TempPrefix = ".syncthing.tmp."
)

// An AtomicWriter is an *os.File that writes to a temporary file in the same
// directory as the final path. On successful Close the file is renamed to
// it's final path. Any error on Write or during Close is accumulated and
// returned on Close, so a lazy user can ignore errors until Close.
type AtomicWriter struct {
	path string
	next *os.File
	err  error
}

// CreateAtomic is like os.Create with a FileMode, except a temporary file
// name is used instead of the given name.
func CreateAtomic(path string, mode os.FileMode) (*AtomicWriter, error) {
	fd, err := tempFile(filepath.Dir(path), TempPrefix, mode)
	if err != nil {
		return nil, err
	}

	w := &AtomicWriter{
		path: path,
		next: fd,
	}

	return w, nil
}

// Write is like io.Writer, but is a no-op on an already failed AtomicWriter.
func (w *AtomicWriter) Write(bs []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err := w.next.Write(bs)
	if err != nil {
		w.err = err
		w.next.Close()
	}
	return n, err
}

// Close closes the temporary file and renames it to the final path. It is
// invalid to call Write() or Close() after Close().
func (w *AtomicWriter) Close() error {
	if w.err != nil {
		return w.err
	}

	// Try to not leave temp file around, but ignore error.
	defer os.Remove(w.next.Name())

	if err := w.next.Sync(); err != nil {
		w.err = err
		return err
	}

	if err := w.next.Close(); err != nil {
		w.err = err
		return err
	}

	// Remove the destination file, on Windows only. If it fails, and not due
	// to the file not existing, we won't be able to complete the rename
	// either. Return this error because it may be more informative. On non-
	// Windows we want the atomic rename behavior so we don't attempt remove.
	if runtime.GOOS == "windows" {
		if err := os.Remove(w.path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	if err := os.Rename(w.next.Name(), w.path); err != nil {
		w.err = err
		return err
	}

	SyncDir(filepath.Dir(w.next.Name()))

	// Set w.err to return appropriately for any future operations.
	w.err = ErrClosed

	return nil
}

// The rest of this file is from ioutil in Go 1.7.3 with light modifications.

// tempFile is ioutil.TempFile but with the mode configurable
func tempFile(dir, prefix string, mode os.FileMode) (f *os.File, err error) {
	if dir == "" {
		dir = os.TempDir()
	}

	nconflict := 0
	for i := 0; i < 10000; i++ {
		name := filepath.Join(dir, prefix+nextSuffix())
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, mode)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		break
	}
	return
}

// Random number state.
// We generate random temporary file names so that there's a good
// chance the file doesn't exist yet - keeps the number of tries in
// TempFile to a minimum.
var rand uint32
var randmu sync.Mutex

func reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func nextSuffix() string {
	randmu.Lock()
	r := rand
	if r == 0 {
		r = reseed()
	}
	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	rand = r
	randmu.Unlock()
	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}
