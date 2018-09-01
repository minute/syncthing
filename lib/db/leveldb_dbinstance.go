// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/sync"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type deletionHandler func(t readWriteTransaction, folder, device, name []byte, dbi iterator.Iterator)

// The reader interface is satisifed by the *leveldb.DB (for inconsistent
// reads), the *leveldb.Snapshot and the *leveldb.Transaction. The
// Transaction is not by itself a snapshot, though.
type reader interface {
	Get(key []byte, ro *opt.ReadOptions) ([]byte, error)
	NewIterator(slice *util.Range, ro *opt.ReadOptions) iterator.Iterator
}

// The writer interface is satsifed by the *leveldb.DB (for inconsistent
// writes), and the *leveldb.Transaction.
type writer interface {
	reader
	Delete(key []byte, wo *opt.WriteOptions) error
	Put(key, value []byte, wo *opt.WriteOptions) error
	Write(b *leveldb.Batch, wo *opt.WriteOptions) error
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

type Instance struct {
	committed int64 // this must be the first attribute in the struct to ensure 64 bit alignment on 32 bit plaforms
	leveldb   *leveldb.DB
	location  string
	folderIdx *smallIndex
	deviceIdx *smallIndex
}

const (
	keyPrefixLen   = 1
	keyFolderLen   = 4 // indexed
	keyDeviceLen   = 4 // indexed
	keySequenceLen = 8
	keyHashLen     = 32

	maxInt64 int64 = 1<<63 - 1
)

func Open(file string) (*Instance, error) {
	opts := &opt.Options{
		OpenFilesCacheCapacity: 100,
		WriteBuffer:            4 << 20,
	}

	db, err := leveldb.OpenFile(file, opts)
	if leveldbIsCorrupted(err) {
		db, err = leveldb.RecoverFile(file, opts)
	}
	if leveldbIsCorrupted(err) {
		// The database is corrupted, and we've tried to recover it but it
		// didn't work. At this point there isn't much to do beyond dropping
		// the database and reindexing...
		l.Infoln("Database corruption detected, unable to recover. Reinitializing...")
		if err := os.RemoveAll(file); err != nil {
			return nil, errorSuggestion{err, "failed to delete corrupted database"}
		}
		db, err = leveldb.OpenFile(file, opts)
	}
	if err != nil {
		return nil, errorSuggestion{err, "is another instance of Syncthing running?"}
	}

	return newDBInstance(db, file)
}

func OpenMemory() *Instance {
	db, _ := leveldb.Open(storage.NewMemStorage(), nil)
	ldb, _ := newDBInstance(db, "<memory>")
	return ldb
}

func newDBInstance(db *leveldb.DB, location string) (*Instance, error) {
	i := &Instance{
		leveldb:  db,
		location: location,
	}
	snap, err := db.GetSnapshot()
	if err != nil {
		return nil, err
	}
	i.folderIdx = newSmallIndex(snap, []byte{KeyTypeFolderIdx})
	i.deviceIdx = newSmallIndex(snap, []byte{KeyTypeDeviceIdx})
	snap.Release()

	tran, err := db.OpenTransaction()
	if err != nil {
		return nil, err
	}
	if err := updateSchema(tran); err != nil {
		tran.Discard()
		return nil, err
	}
	return i, tran.Commit()
}

// Committed returns the number of items committed to the database since startup
func (db *Instance) Committed() int64 {
	return atomic.LoadInt64(&db.committed)
}

// Location returns the filesystem path where the database is stored
func (db *Instance) Location() string {
	return db.location
}

func (db *Instance) updateFiles(w writer, folder, device []byte, fs []protocol.FileInfo, meta *metadataTracker) {
	var fk []byte
	var gk []byte
	for _, f := range fs {
		name := []byte(f.Name)
		fk = db.deviceKeyInto(w, fk, folder, device, name)

		// Get and unmarshal the file entry. If it doesn't exist or can't be
		// unmarshalled we'll add it as a new entry.
		bs, err := w.Get(fk, nil)
		var ef FileInfoTruncated
		if err == nil {
			err = ef.Unmarshal(bs)
		}

		// Local flags or the invalid bit might change without the version
		// being bumped. The IsInvalid() method handles both.
		if err == nil && ef.Version.Equal(f.Version) && ef.IsInvalid() == f.IsInvalid() {
			continue
		}

		devID := protocol.DeviceIDFromBytes(device)
		if err == nil {
			meta.removeFile(devID, ef)
		}
		meta.addFile(devID, f)

		insertFile(w, fk, folder, device, f)

		gk = db.globalKeyInto(w, gk, folder, name)
		updateGlobal(w, gk, folder, device, f, meta)
	}
}

func (db *Instance) addSequences(w writer, folder []byte, fs []protocol.FileInfo) {
	var sk []byte
	var dk []byte
	for _, f := range fs {
		sk = db.sequenceKeyInto(w, sk, folder, f.Sequence)
		dk = db.deviceKeyInto(w, dk, folder, protocol.LocalDeviceID[:], []byte(f.Name))
		w.Put(sk, dk, nil)
	}
}

func (db *Instance) removeSequences(w writer, folder []byte, fs []protocol.FileInfo) {
	var sk []byte
	for _, f := range fs {
		w.Delete(db.sequenceKeyInto(w, sk, folder, f.Sequence), nil)
	}
}

func (db *Instance) withHave(r reader, folder, device, prefix []byte, truncate bool, fn Iterator) {
	if len(prefix) > 0 {
		unslashedPrefix := prefix
		if bytes.HasSuffix(prefix, []byte{'/'}) {
			unslashedPrefix = unslashedPrefix[:len(unslashedPrefix)-1]
		} else {
			prefix = append(prefix, '/')
		}

		if f, ok := db.getFileTrunc(db.deviceKey(noWriter{r}, folder, device, unslashedPrefix), true); ok && !fn(f) {
			return
		}
	}

	dbi := r.NewIterator(util.BytesPrefix(db.deviceKey(noWriter{r}, folder, device, prefix)[:keyPrefixLen+keyFolderLen+keyDeviceLen+len(prefix)]), nil)
	defer dbi.Release()

	for dbi.Next() {
		name := db.deviceKeyName(dbi.Key())
		if len(prefix) > 0 && !bytes.HasPrefix(name, prefix) {
			return
		}

		// The iterator function may keep a reference to the unmarshalled
		// struct, which in turn references the buffer it was unmarshalled
		// from. dbi.Value() just returns an internal slice that it reuses, so
		// we need to copy it.
		f, err := unmarshalTrunc(append([]byte{}, dbi.Value()...), truncate)
		if err != nil {
			l.Debugln("unmarshal error:", err)
			continue
		}
		if !fn(f) {
			return
		}
	}
}

func (db *Instance) withHaveSequence(r reader, folder []byte, startSeq int64, fn Iterator) {
	dbi := r.NewIterator(&util.Range{Start: db.sequenceKey(noWriter{r}, folder, startSeq), Limit: db.sequenceKey(noWriter{r}, folder, maxInt64)}, nil)
	defer dbi.Release()

	for dbi.Next() {
		f, ok := db.getFile(dbi.Value())
		if !ok {
			l.Debugln("missing file for sequence number", db.sequenceKeySequence(dbi.Key()))
			continue
		}
		if !fn(f) {
			return
		}
	}
}

func (db *Instance) withAllFolderTruncated(r reader, folder []byte, fn func(device []byte, f FileInfoTruncated) bool) {
	dbi := r.NewIterator(util.BytesPrefix(db.deviceKey(noWriter{r}, folder, nil, nil)[:keyPrefixLen+keyFolderLen]), nil)
	defer dbi.Release()

	var gk []byte

	for dbi.Next() {
		device := db.deviceKeyDevice(dbi.Key())
		var f FileInfoTruncated
		// The iterator function may keep a reference to the unmarshalled
		// struct, which in turn references the buffer it was unmarshalled
		// from. dbi.Value() just returns an internal slice that it reuses, so
		// we need to copy it.
		err := f.Unmarshal(append([]byte{}, dbi.Value()...))
		if err != nil {
			l.Debugln("unmarshal error:", err)
			continue
		}

		if !fn(device, f) {
			return
		}
	}
}

func getFile(r reader, key []byte) (protocol.FileInfo, bool) {
	if f, ok := getFileTrunc(r, key, false); ok {
		return f.(protocol.FileInfo), true
	}
	return protocol.FileInfo{}, false
}

func getFileTrunc(r reader, key []byte, trunc bool) (FileIntf, bool) {
	bs, err := r.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return nil, false
	}
	if err != nil {
		l.Debugln("surprise error:", err)
		return nil, false
	}

	f, err := unmarshalTrunc(bs, trunc)
	if err != nil {
		l.Debugln("unmarshal error:", err)
		return nil, false
	}
	return f, true
}

func (db *Instance) getGlobal(folder, file []byte, truncate bool) (FileIntf, bool) {
	t := db.newReadOnlyTransaction()
	defer t.close()

	_, _, f, ok := db.getGlobalInto(t, nil, nil, folder, file, truncate)
	return f, ok
}

func (db *Instance) getGlobalInto(r reader, gk, dk, folder, file []byte, truncate bool) ([]byte, []byte, FileIntf, bool) {
	gk = db.globalKeyInto(noWriter{r}, gk, folder, file)

	bs, err := r.Get(gk, nil)
	if err != nil {
		return gk, dk, nil, false
	}

	vl, ok := unmarshalVersionList(bs)
	if !ok {
		return gk, dk, nil, false
	}

	dk = db.deviceKeyInto(noWriter{r}, dk, folder, vl.Versions[0].Device, file)
	if fi, ok := getFileTrunc(r, dk, truncate); ok {
		return gk, dk, fi, true
	}

	return gk, dk, nil, false
}

func (db *Instance) withGlobal(r reader, folder, prefix []byte, truncate bool, fn Iterator) {
	if len(prefix) > 0 {
		unslashedPrefix := prefix
		if bytes.HasSuffix(prefix, []byte{'/'}) {
			unslashedPrefix = unslashedPrefix[:len(unslashedPrefix)-1]
		} else {
			prefix = append(prefix, '/')
		}

		if f, ok := db.getGlobal(folder, unslashedPrefix, truncate); ok && !fn(f) {
			return
		}
	}

	dbi := r.NewIterator(util.BytesPrefix(db.globalKey(noWriter{r}, folder, prefix)), nil)
	defer dbi.Release()

	var fk []byte
	for dbi.Next() {
		name := db.globalKeyName(dbi.Key())
		if len(prefix) > 0 && !bytes.HasPrefix(name, prefix) {
			return
		}

		vl, ok := unmarshalVersionList(dbi.Value())
		if !ok {
			continue
		}

		fk = db.deviceKeyInto(noWriter{r}, fk, folder, vl.Versions[0].Device, name)

		f, ok := getFileTrunc(r, fk, truncate)
		if !ok {
			continue
		}

		if !fn(f) {
			return
		}
	}
}

func (db *Instance) availability(r reader, folder, file []byte) []protocol.DeviceID {
	k := db.globalKey(noWriter{r}, folder, file)
	bs, err := r.Get(k, nil)
	if err == leveldb.ErrNotFound {
		return nil
	}
	if err != nil {
		l.Debugln("surprise error:", err)
		return nil
	}

	vl, ok := unmarshalVersionList(bs)
	if !ok {
		return nil
	}

	var devices []protocol.DeviceID
	for _, v := range vl.Versions {
		if !v.Version.Equal(vl.Versions[0].Version) {
			break
		}
		if v.Invalid {
			continue
		}
		n := protocol.DeviceIDFromBytes(v.Device)
		devices = append(devices, n)
	}

	return devices
}

func (db *Instance) withNeed(folder, device []byte, truncate bool, fn Iterator) {
	if bytes.Equal(device, protocol.LocalDeviceID[:]) {
		db.withNeedLocal(folder, truncate, fn)
		return
	}

	t := db.newReadOnlyTransaction()
	defer t.close()

	dbi := t.NewIterator(util.BytesPrefix(db.globalKey(folder, nil)[:keyPrefixLen+keyFolderLen]), nil)
	defer dbi.Release()

	var fk []byte
	for dbi.Next() {
		vl, ok := unmarshalVersionList(dbi.Value())
		if !ok {
			continue
		}

		haveFV, have := vl.Get(device)
		// XXX: This marks Concurrent (i.e. conflicting) changes as
		// needs. Maybe we should do that, but it needs special
		// handling in the puller.
		if have && haveFV.Version.GreaterEqual(vl.Versions[0].Version) {
			continue
		}

		name := db.globalKeyName(dbi.Key())
		needVersion := vl.Versions[0].Version
		needDevice := protocol.DeviceIDFromBytes(vl.Versions[0].Device)

		for i := range vl.Versions {
			if !vl.Versions[i].Version.Equal(needVersion) {
				// We haven't found a valid copy of the file with the needed version.
				break
			}

			if vl.Versions[i].Invalid {
				// The file is marked invalid, don't use it.
				continue
			}

			fk = db.deviceKeyInto(fk, folder, vl.Versions[i].Device, name)
			bs, err := t.Get(fk, nil)
			if err != nil {
				l.Debugln("surprise error:", err)
				continue
			}

			gf, err := unmarshalTrunc(bs, truncate)
			if err != nil {
				l.Debugln("unmarshal error:", err)
				continue
			}

			if gf.IsDeleted() && !have {
				// We don't need deleted files that we don't have
				break
			}

			l.Debugf("need folder=%q device=%v name=%q have=%v invalid=%v haveV=%v globalV=%v globalDev=%v", folder, protocol.DeviceIDFromBytes(device), name, have, haveFV.Invalid, haveFV.Version, needVersion, needDevice)

			if !fn(gf) {
				return
			}

			// This file is handled, no need to look further in the version list
			break
		}
	}
}

func (db *Instance) withNeedLocal(folder []byte, truncate bool, fn Iterator) {
	t := db.newReadOnlyTransaction()
	defer t.close()

	dbi := t.NewIterator(util.BytesPrefix(db.needKey(folder, nil)[:keyPrefixLen+keyFolderLen]), nil)
	defer dbi.Release()

	var dk []byte
	var gk []byte
	var f FileIntf
	var ok bool
	for dbi.Next() {
		gk, dk, f, ok = db.getGlobalInto(t, gk, dk, folder, db.globalKeyName(dbi.Key()), truncate)
		if !ok {
			continue
		}
		if !fn(f) {
			return
		}
	}
}

func (db *Instance) ListFolders() []string {
	t := db.newReadOnlyTransaction()
	defer t.close()

	dbi := t.NewIterator(util.BytesPrefix([]byte{KeyTypeGlobal}), nil)
	defer dbi.Release()

	folderExists := make(map[string]bool)
	for dbi.Next() {
		folder, ok := db.globalKeyFolder(dbi.Key())
		if ok && !folderExists[string(folder)] {
			folderExists[string(folder)] = true
		}
	}

	folders := make([]string, 0, len(folderExists))
	for k := range folderExists {
		folders = append(folders, k)
	}

	sort.Strings(folders)
	return folders
}

func (db *Instance) dropFolder(folder []byte) {
	t := db.newReadWriteTransaction()
	defer t.close()

	for _, key := range [][]byte{
		// Remove all items related to the given folder from the device->file bucket
		db.deviceKey(folder, nil, nil)[:keyPrefixLen+keyFolderLen],
		// Remove all sequences related to the folder
		db.sequenceKey([]byte(folder), 0)[:keyPrefixLen+keyFolderLen],
		// Remove all items related to the given folder from the global bucket
		db.globalKey(folder, nil)[:keyPrefixLen+keyFolderLen],
		// Remove all needs related to the folder
		db.needKey(folder, nil)[:keyPrefixLen+keyFolderLen],
	} {
		t.deleteKeyPrefix(key)
	}
}

func (db *Instance) dropDeviceFolder(device, folder []byte, meta *metadataTracker) {
	t := db.newReadWriteTransaction()
	defer t.close()

	dbi := t.NewIterator(util.BytesPrefix(db.deviceKey(folder, device, nil)), nil)
	defer dbi.Release()

	var gk []byte

	for dbi.Next() {
		key := dbi.Key()
		name := db.deviceKeyName(key)
		gk = db.globalKeyInto(gk, folder, name)
		t.removeFromGlobal(gk, folder, device, name, meta)
		t.Delete(key)
		t.checkFlush()
	}
}

func (db *Instance) checkGlobals(w writer, folder []byte, meta *metadataTracker) {
	dbi := w.NewIterator(util.BytesPrefix(db.globalKey(folder, nil)[:keyPrefixLen+keyFolderLen]), nil)
	defer dbi.Release()

	var fk []byte
	for dbi.Next() {
		vl, ok := unmarshalVersionList(dbi.Value())
		if !ok {
			continue
		}

		// Check the global version list for consistency. An issue in previous
		// versions of goleveldb could result in reordered writes so that
		// there are global entries pointing to no longer existing files. Here
		// we find those and clear them out.

		name := db.globalKeyName(dbi.Key())
		var newVL VersionList
		for i, version := range vl.Versions {
			fk = db.deviceKeyInto(w, fk, folder, version.Device, name)
			_, err := w.Get(fk, nil)
			if err == leveldb.ErrNotFound {
				continue
			}
			if err != nil {
				l.Debugln("surprise error:", err)
				return
			}
			newVL.Versions = append(newVL.Versions, version)

			if i == 0 {
				if fi, ok := db.getFile(fk); ok {
					meta.addFile(protocol.GlobalDeviceID, fi)
				}
			}
		}

		if len(newVL.Versions) != len(vl.Versions) {
			w.Put(dbi.Key(), mustMarshal(&newVL), nil)
		}
	}
	l.Debugf("db check completed for %q", folder)
}

// deviceKey returns a byte slice encoding the following information:
//	   keyTypeDevice (1 byte)
//	   folder (4 bytes)
//	   device (4 bytes)
//	   name (variable size)
func (db *Instance) deviceKey(w writer, folder, device, file []byte) []byte {
	return db.deviceKeyInto(w, nil, folder, device, file)
}

func (db *Instance) deviceKeyInto(w writer, k, folder, device, file []byte) []byte {
	reqLen := keyPrefixLen + keyFolderLen + keyDeviceLen + len(file)
	k = resize(k, reqLen)
	k[0] = KeyTypeDevice
	binary.BigEndian.PutUint32(k[keyPrefixLen:], db.folderIdx.ID(w, folder))
	binary.BigEndian.PutUint32(k[keyPrefixLen+keyFolderLen:], db.deviceIdx.ID(w, device))
	copy(k[keyPrefixLen+keyFolderLen+keyDeviceLen:], file)
	return k
}

// deviceKeyName returns the device ID from the key
func (db *Instance) deviceKeyName(key []byte) []byte {
	return key[keyPrefixLen+keyFolderLen+keyDeviceLen:]
}

// deviceKeyFolder returns the folder name from the key
func (db *Instance) deviceKeyFolder(key []byte) []byte {
	folder, ok := db.folderIdx.Val(binary.BigEndian.Uint32(key[keyPrefixLen:]))
	if !ok {
		panic("bug: lookup of nonexistent folder ID")
	}
	return folder
}

// deviceKeyDevice returns the device ID from the key
func (db *Instance) deviceKeyDevice(key []byte) []byte {
	device, ok := db.deviceIdx.Val(binary.BigEndian.Uint32(key[keyPrefixLen+keyFolderLen:]))
	if !ok {
		panic("bug: lookup of nonexistent device ID")
	}
	return device
}

// globalKey returns a byte slice encoding the following information:
//	   keyTypeGlobal (1 byte)
//	   folder (4 bytes)
//	   name (variable size)
func (db *Instance) globalKey(w writer, folder, file []byte) []byte {
	return db.globalKeyInto(w, nil, folder, file)
}

func (db *Instance) globalKeyInto(w writer, gk, folder, file []byte) []byte {
	reqLen := keyPrefixLen + keyFolderLen + len(file)
	gk = resize(gk, reqLen)
	gk[0] = KeyTypeGlobal
	binary.BigEndian.PutUint32(gk[keyPrefixLen:], db.folderIdx.ID(w, folder))
	copy(gk[keyPrefixLen+keyFolderLen:], file)
	return gk
}

// globalKeyName returns the filename from the key
func (db *Instance) globalKeyName(key []byte) []byte {
	return key[keyPrefixLen+keyFolderLen:]
}

// globalKeyFolder returns the folder name from the key
func (db *Instance) globalKeyFolder(key []byte) ([]byte, bool) {
	return db.folderIdx.Val(binary.BigEndian.Uint32(key[keyPrefixLen:]))
}

// needKey is a globalKey with a different prefix
func (db *Instance) needKey(w writer, folder, file []byte) []byte {
	return db.needKeyInto(w, nil, folder, file)
}

func (db *Instance) needKeyInto(w writer, k, folder, file []byte) []byte {
	k = db.globalKeyInto(w, k, folder, file)
	k[0] = KeyTypeNeed
	return k
}

// sequenceKey returns a byte slice encoding the following information:
//	   KeyTypeSequence (1 byte)
//	   folder (4 bytes)
//	   sequence number (8 bytes)
func (db *Instance) sequenceKey(w writer, folder []byte, seq int64) []byte {
	return db.sequenceKeyInto(w, nil, folder, seq)
}

func (db *Instance) sequenceKeyInto(w writer, k []byte, folder []byte, seq int64) []byte {
	reqLen := keyPrefixLen + keyFolderLen + keySequenceLen
	k = resize(k, reqLen)
	k[0] = KeyTypeSequence
	binary.BigEndian.PutUint32(k[keyPrefixLen:], db.folderIdx.ID(w, folder))
	binary.BigEndian.PutUint64(k[keyPrefixLen+keyFolderLen:], uint64(seq))
	return k
}

// sequenceKeySequence returns the sequence number from the key
func (db *Instance) sequenceKeySequence(key []byte) int64 {
	return int64(binary.BigEndian.Uint64(key[keyPrefixLen+keyFolderLen:]))
}

func (db *Instance) getIndexID(w writer, device, folder []byte) protocol.IndexID {
	key := db.indexIDKey(w, device, folder)
	cur, err := w.Get(key, nil)
	if err != nil {
		return 0
	}

	var id protocol.IndexID
	if err := id.Unmarshal(cur); err != nil {
		return 0
	}

	return id
}

func (db *Instance) setIndexID(device, folder []byte, id protocol.IndexID) {
	key := db.indexIDKey(device, folder)
	bs, _ := id.Marshal() // marshalling can't fail
	if err := db.Put(key, bs, nil); err != nil {
		panic("storing index ID: " + err.Error())
	}
}

func (db *Instance) indexIDKey(w writer, device, folder []byte) []byte {
	k := make([]byte, keyPrefixLen+keyDeviceLen+keyFolderLen)
	k[0] = KeyTypeIndexID
	binary.BigEndian.PutUint32(k[keyPrefixLen:], db.deviceIdx.ID(w, device))
	binary.BigEndian.PutUint32(k[keyPrefixLen+keyDeviceLen:], db.folderIdx.ID(w, folder))
	return k
}

func (db *Instance) indexIDDevice(key []byte) []byte {
	device, _ := db.deviceIdx.Val(binary.BigEndian.Uint32(key[keyPrefixLen:]))
	return device
}

func (db *Instance) mtimesKey(w writer, folder []byte) []byte {
	prefix := make([]byte, 5) // key type + 4 bytes folder idx number
	prefix[0] = KeyTypeVirtualMtime
	binary.BigEndian.PutUint32(prefix[1:], db.folderIdx.ID(w, folder))
	return prefix
}

func (db *Instance) folderMetaKey(w writer, folder []byte) []byte {
	prefix := make([]byte, 5) // key type + 4 bytes folder idx number
	prefix[0] = KeyTypeFolderMeta
	binary.BigEndian.PutUint32(prefix[1:], db.folderIdx.ID(w, folder))
	return prefix
}

// DropLocalDeltaIndexIDs removes all index IDs for the local device ID from
// the database. This will cause a full index transmission on the next
// connection.
func (db *Instance) DropLocalDeltaIndexIDs() {
	tran, err := db.OpenTransaction()
	if err != nil {
		panic(err)
	}
	db.dropDeltaIndexIDs(w, true)
	if err := w.Commit(); err != nil {
		panic(err)
	}
}

// DropRemoteDeltaIndexIDs removes all index IDs for the other devices than
// the local one from the database. This will cause them to send us a full
// index on the next connection.
func (db *Instance) DropRemoteDeltaIndexIDs() {
	tran, err := db.OpenTransaction()
	if err != nil {
		panic(err)
	}
	db.dropDeltaIndexIDs(w, false)
	if err := w.Commit(); err != nil {
		panic(err)
	}
}

func dropDeltaIndexIDs(w writer, local bool) {
	dbi := w.NewIterator(util.BytesPrefix([]byte{KeyTypeIndexID}), nil)
	defer dbi.Release()

	for dbi.Next() {
		device := indexIDDevice(w, dbi.Key())
		if bytes.Equal(device, protocol.LocalDeviceID[:]) == local {
			w.Delete(dbi.Key(), nil)
		}
	}
}

func dropMtimes(w writer, folder []byte) {
	db.dropPrefix(w, db.mtimesKey(folder))
}

func dropFolderMeta(w writer, folder []byte) {
	db.dropPrefix(w, db.folderMetaKey(folder))
}

func dropPrefix(w writer, prefix []byte) {
	dbi := w.NewIterator(util.BytesPrefix(prefix), nil)
	defer dbi.Release()

	for dbi.Next() {
		w.Delete(dbi.Key(), nil)
	}
}

func unmarshalTrunc(bs []byte, truncate bool) (FileIntf, error) {
	if truncate {
		var tf FileInfoTruncated
		err := tf.Unmarshal(bs)
		return tf, err
	}

	var tf protocol.FileInfo
	err := tf.Unmarshal(bs)
	return tf, err
}

func unmarshalVersionList(data []byte) (VersionList, bool) {
	var vl VersionList
	if err := vl.Unmarshal(data); err != nil {
		l.Debugln("unmarshal error:", err)
		return VersionList{}, false
	}
	if len(vl.Versions) == 0 {
		l.Debugln("empty version list")
		return VersionList{}, false
	}
	return vl, true
}

// A "better" version of leveldb's errors.IsCorrupted.
func leveldbIsCorrupted(err error) bool {
	switch {
	case err == nil:
		return false

	case errors.IsCorrupted(err):
		return true

	case strings.Contains(err.Error(), "corrupted"):
		return true
	}

	return false
}

// A smallIndex is an in memory bidirectional []byte to uint32 map. It gives
// fast lookups in both directions and persists to the database. Don't use for
// storing more items than fit comfortably in RAM.
type smallIndex struct {
	prefix []byte
	id2val map[uint32]string
	val2id map[string]uint32
	nextID uint32
	mut    sync.Mutex
}

func newSmallIndex(r reader, prefix []byte) *smallIndex {
	idx := &smallIndex{
		prefix: prefix,
		id2val: make(map[uint32]string),
		val2id: make(map[string]uint32),
		mut:    sync.NewMutex(),
	}
	idx.load(r)
	return idx
}

// load iterates over the prefix space in the database and populates the in
// memory maps.
func (i *smallIndex) load(r reader) {
	it := r.NewIterator(util.BytesPrefix(i.prefix), nil)
	for it.Next() {
		val := string(it.Value())
		id := binary.BigEndian.Uint32(it.Key()[len(i.prefix):])
		i.id2val[id] = val
		i.val2id[val] = id
		if id >= i.nextID {
			i.nextID = id + 1
		}
	}
	it.Release()
}

// ID returns the index number for the given byte slice, allocating a new one
// and persisting this to the database if necessary.
func (i *smallIndex) ID(w writer, val []byte) uint32 {
	i.mut.Lock()
	// intentionally avoiding defer here as we want this call to be as fast as
	// possible in the general case (folder ID already exists). The map lookup
	// with the conversion of []byte to string is compiler optimized to not
	// copy the []byte, which is why we don't assign it to a temp variable
	// here.
	if id, ok := i.val2id[string(val)]; ok {
		i.mut.Unlock()
		return id
	}

	id := i.nextID
	i.nextID++

	valStr := string(val)
	i.val2id[valStr] = id
	i.id2val[id] = valStr

	key := make([]byte, len(i.prefix)+8) // prefix plus uint32 id
	copy(key, i.prefix)
	binary.BigEndian.PutUint32(key[len(i.prefix):], id)
	w.Put(key, val, nil)

	i.mut.Unlock()
	return id
}

// Val returns the value for the given index number, or (nil, false) if there
// is no such index number.
func (i *smallIndex) Val(id uint32) ([]byte, bool) {
	i.mut.Lock()
	val, ok := i.id2val[id]
	i.mut.Unlock()
	if !ok {
		return nil, false
	}

	return []byte(val), true
}

// resize returns a byte array of length reqLen, reusing k if possible
func resize(k []byte, reqLen int) []byte {
	if cap(k) < reqLen {
		return make([]byte, reqLen)
	}
	return k[:reqLen]
}

type errorSuggestion struct {
	inner      error
	suggestion string
}

func (e errorSuggestion) Error() string {
	return fmt.Sprintf("%s (%s)", e.inner.Error(), e.suggestion)
}
