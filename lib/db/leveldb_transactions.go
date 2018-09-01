// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package db

import (
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// The reader interface is satisifed by the *leveldb.DB (for inconsistent
// reads), the *leveldb.Snapshot and the *leveldb.Transaction. The
// Transaction is not by itself a snapshot, though.
type reader interface {
	Get(key []byte, ro *opt.ReadOptions) ([]byte, error)
	Has(key []byte, ro *opt.ReadOptions) (bool, error)
	NewIterator(slice *util.Range, ro *opt.ReadOptions) iterator.Iterator
}

// The writer interface is satsifed by the *leveldb.DB (for inconsistent
// writes), and the *leveldb.Transaction.
type writer interface {
	Delete(key []byte, wo *opt.WriteOptions) error
	Put(key, value []byte, wo *opt.WriteOptions) error
	Write(b *leveldb.Batch, wo *opt.WriteOptions) error
}

type transaction interface {
	reader
	writer
}

type transactionManager struct {
	db *leveldb.DB
}

// withoutTransaction runs fn in a transaction-less environment; reads are
// immediate and writes are dirty.
func (m transactionManager) withoutTransaction(fn func(transaction) error) error {
	return fn(m.db)
}

// inReadTransaction runs fn in a read-only transaction where reads are
// consistent (writes will panic).
func (m transactionManager) inReadTransaction(fn func(transaction) error) error {
	snap, err := m.db.GetSnapshot()
	if err != nil {
		return err
	}
	defer snap.Release()
	return fn(noWriter{snap})
}

// inDirtyTransaction runs fn in a transaction where reads are consistent
// and writes are dirty.
func (m transactionManager) inDirtyTransaction(fn func(transaction) error) error {
	snap, err := m.db.GetSnapshot()
	if err != nil {
		return err
	}
	defer snap.Release()

	tran := struct {
		reader
		writer
	}{snap, m.db}
	return fn(tran)
}

// inWriteTransaction runs fn in a transaction where reads are consistent
// and writes are applied atomically at the end, if fn returns nil. If fn
// returns an error the writes are discarded.
func (m transactionManager) inWriteTransaction(fn func(transaction) error) error {
	return m.inReadTransaction(func(rt transaction) error {
		wt, err := m.db.OpenTransaction()
		if err != nil {
			return err
		}
		defer wt.Discard()

		tran := struct {
			reader
			writer
		}{rt, wt}
		if err := fn(tran); err != nil {
			return err
		}

		return wt.Commit()
	})
}

// noWriter is a writer that cannot write. This can be passed to functions
// that expect a writer, but will not use it (typically, the smallIndex when
// it's known the lookup will succeed).
type noWriter struct {
	reader
}

func (noWriter) Delete(key []byte, wo *opt.WriteOptions) error      { panic("put: Delete disallowed") }
func (noWriter) Put(key, value []byte, wo *opt.WriteOptions) error  { panic("put: Put disallowed") }
func (noWriter) Write(b *leveldb.Batch, wo *opt.WriteOptions) error { panic("bug: Write disallowed") }
func (noWriter) Close() error                                       { panic("bug: Write disallowed") }

func need(global FileIntf, haveLocal bool, localVersion protocol.Vector) bool {
	// We never need an invalid file.
	if global.IsInvalid() {
		return false
	}
	// We don't need a deleted file if we don't have it.
	if global.IsDeleted() && !haveLocal {
		return false
	}
	// We don't need the global file if we already have the same version.
	if haveLocal && localVersion.Equal(global.FileVersion()) {
		return false
	}
	return true
}

func deleteKeyPrefix(t transaction, prefix []byte) {
	dbi := t.NewIterator(util.BytesPrefix(prefix), nil)
	for dbi.Next() {
		t.Delete(dbi.Key(), nil)
	}
	dbi.Release()
}

type marshaller interface {
	Marshal() ([]byte, error)
}

func mustMarshal(f marshaller) []byte {
	bs, err := f.Marshal()
	if err != nil {
		panic(err)
	}
	return bs
}
