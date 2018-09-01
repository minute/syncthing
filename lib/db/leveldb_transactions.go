// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package db

import (
	"bytes"

	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// A readOnlyTransaction represents a database snapshot.
type readOnlyTransaction struct {
	*leveldb.Snapshot
}

func (db *Instance) newReadOnlyTransaction() readOnlyTransaction {
	snap, err := db.GetSnapshot()
	if err != nil {
		panic(err)
	}
	return readOnlyTransaction{
		Snapshot: snap,
		db:       db,
	}
}

func (t readOnlyTransaction) close() {
	t.Release()
}

func (t readOnlyTransaction) getFile(folder, device, file []byte) (protocol.FileInfo, bool) {
	return t.db.getFile(t.db.deviceKey(folder, device, file))
}

// A readWriteTransaction is a readOnlyTransaction plus a Transaction for writes.
type readWriteTransaction struct {
	readOnlyTransaction
	*leveldb.Transaction
}

func (db *Instance) newReadWriteTransaction() readWriteTransaction {
	rt := db.newReadOnlyTransaction()
	wt, err := db.OpenTransaction()
	if err != nil {
		panic(err)
	}
	return readWriteTransaction{
		readOnlyTransaction: rt,
		Transaction:         wt,
	}
}

func (t readWriteTransaction) close() {
	t.readOnlyTransaction.close()
	if err := t.Commit(); err != nil {
		panic(err)
	}
}

// A batchedWriter is a wrapper around a writer that does automatic batching.
type batchedWriter struct {
	batch *leveldb.Batch
	next  writer
}

func newBatchedWriter(w writer) *batchedWriter {
	return &batchedWriter{
		batch: new(leveldb.Batch),
		next:  w,
	}
}

func (w *batchedWriter) Delete(key []byte, wo *opt.WriteOptions) error {
	w.batch.Delete(key)
	w.checkFlush()
}

func (w *batchedWriter) Put(key, value []byte, wo *opt.WriteOptions) error {
	w.batch.Put(key, value)
	w.checkFlush()
}

func (w *batchedWriter) Write(b *leveldb.Batch, wo *opt.WriteOptions) error {
	w.flush()
	w.next.Write(b)
}

func (w *batchedWriter) checkFlush() {
	if w.batch.Len() > batchFlushSize {
		w.flush()
	}
}

func (w *batchedWriter) flush() {
	if err := w.next.Write(b.batch, nil); err != nil {
		panic(err)
	}
	w.batch.Reset()
}

// updateGlobal adds this device+version to the version list for the given
// file. If the device is already present in the list, the version is updated.
// If the file does not have an entry in the global list, it is created.
func updateGlobal(w writer, gk, folder, device []byte, file protocol.FileInfo, meta *metadataTracker) bool {
	var fl VersionList
	if svl, err := r.Get(gk, nil); err == nil {
		fl.Unmarshal(svl) // Ignore error, continue with empty fl
	}
	fl, removedFV, removedAt, insertedAt := fl.update(folder, device, file, t.db)
	if insertedAt == -1 {
		l.Debugln("update global; same version, global unchanged")
		return false
	}

	name := []byte(file.Name)

	var newGlobal protocol.FileInfo
	if insertedAt == 0 {
		// Inserted a new newest version
		newGlobal = file
	} else if new, ok := t.getFile(folder, fl.Versions[0].Device, name); ok {
		// The previous second version is now the first
		newGlobal = new
	} else {
		panic("This file must exist in the db")
	}

	// Fixup the list of files we need.
	nk := t.db.needKey(folder, name)
	hasNeeded, _ := t.db.Has(nk, nil)
	if localFV, haveLocalFV := fl.Get(protocol.LocalDeviceID[:]); need(newGlobal, haveLocalFV, localFV.Version) {
		if !hasNeeded {
			l.Debugf("local need insert; folder=%q, name=%q", folder, name)
			t.Put(nk, nil)
		}
	} else if hasNeeded {
		l.Debugf("local need delete; folder=%q, name=%q", folder, name)
		t.Delete(nk)
	}

	if removedAt != 0 && insertedAt != 0 {
		l.Debugf(`new global for "%v" after update: %v`, file.Name, fl)
		t.Put(gk, mustMarshal(&fl))
		return true
	}

	// Remove the old global from the global size counter
	var oldGlobalFV FileVersion
	if removedAt == 0 {
		oldGlobalFV = removedFV
	} else if len(fl.Versions) > 1 {
		// The previous newest version is now at index 1
		oldGlobalFV = fl.Versions[1]
	}
	if oldFile, ok := t.getFile(folder, oldGlobalFV.Device, name); ok {
		// A failure to get the file here is surprising and our
		// global size data will be incorrect until a restart...
		meta.removeFile(protocol.GlobalDeviceID, oldFile)
	}

	// Add the new global to the global size counter
	meta.addFile(protocol.GlobalDeviceID, newGlobal)

	l.Debugf(`new global for "%v" after update: %v`, file.Name, fl)
	t.Put(gk, mustMarshal(&fl))

	return true
}

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

// removeFromGlobal removes the device from the global version list for the
// given file. If the version list is empty after this, the file entry is
// removed entirely.
func (t readWriteTransaction) removeFromGlobal(w writer, gk, folder, device, file []byte, meta *metadataTracker) {
	svl, err := w.Get(gk, nil)
	if err != nil {
		// We might be called to "remove" a global version that doesn't exist
		// if the first update for the file is already marked invalid.
		return
	}

	var fl VersionList
	err = fl.Unmarshal(svl)
	if err != nil {
		l.Debugln("unmarshal error:", err)
		return
	}

	removed := false
	for i := range fl.Versions {
		if bytes.Equal(fl.Versions[i].Device, device) {
			if i == 0 && meta != nil {
				f, ok := getFile(w, folder, device, file)
				if !ok {
					// didn't exist anyway, apparently
					continue
				}
				meta.removeFile(protocol.GlobalDeviceID, f)
				removed = true
			}
			fl.Versions = append(fl.Versions[:i], fl.Versions[i+1:]...)
			break
		}
	}

	if len(fl.Versions) == 0 {
		w.Delete(gk)
		return
	}
	w.Put(gk, mustMarshal(&fl))
	if removed {
		if f, ok := getFile(w, folder, fl.Versions[0].Device, file); ok {
			// A failure to get the file here is surprising and our
			// global size data will be incorrect until a restart...
			meta.addFile(protocol.GlobalDeviceID, f)
		}
	}
}

func deleteKeyPrefix(w writer, prefix []byte) {
	dbi := w.NewIterator(util.BytesPrefix(prefix), nil)
	for dbi.Next() {
		w.Delete(dbi.Key(), nil)
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
