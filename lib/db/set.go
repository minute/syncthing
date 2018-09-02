// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package db provides a set type to track local/remote files with newness
// checks. We must do a certain amount of normalization in here. We will get
// fed paths with either native or wire-format separators and encodings
// depending on who calls us. We transform paths to wire-format (NFC and
// slashes) on the way to the database, and transform to native format
// (varying separator and encoding) on the way back out.
package db

import (
	"os"
	"time"

	"github.com/syncthing/syncthing/lib/fs"
	"github.com/syncthing/syncthing/lib/osutil"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/sync"
)

type FileSet struct {
	folder   string
	fs       fs.Filesystem
	db       *Instance
	blockmap *BlockMap
	meta     *metadataTracker

	updateMutex sync.Mutex // protects database updates and the corresponding metadata changes
}

// FileIntf is the set of methods implemented by both protocol.FileInfo and
// FileInfoTruncated.
type FileIntf interface {
	FileSize() int64
	FileName() string
	FileLocalFlags() uint32
	IsDeleted() bool
	IsInvalid() bool
	IsIgnored() bool
	IsUnsupported() bool
	MustRescan() bool
	IsDirectory() bool
	IsSymlink() bool
	ShouldConflict() bool
	HasPermissionBits() bool
	SequenceNo() int64
	BlockSize() int
	FileVersion() protocol.Vector
}

// The Iterator is called with either a protocol.FileInfo or a
// FileInfoTruncated (depending on the method) and returns true to
// continue iteration, false to stop.
type Iterator func(f FileIntf) bool

var databaseRecheckInterval = 30 * 24 * time.Hour

func init() {
	if dur, err := time.ParseDuration(os.Getenv("STRECHECKDBEVERY")); err == nil {
		databaseRecheckInterval = dur
	}
}

func NewFileSet(folder string, fs fs.Filesystem, db *Instance) *FileSet {
	var s FileSet
	db.tm.withoutTransaction(func(t transaction) error {
		s = FileSet{
			folder:      folder,
			fs:          fs,
			db:          db,
			blockmap:    NewBlockMap(db.folderIdx.ID(t, []byte(folder))),
			meta:        newMetadataTracker(),
			updateMutex: sync.NewMutex(),
		}

		if err := s.meta.fromDB(t, db, []byte(folder)); err != nil {
			l.Infof("No stored folder metadata for %q: recalculating", folder)
			s.recalcCounts()
		} else if age := time.Since(s.meta.Created()); age > databaseRecheckInterval {
			l.Infof("Stored folder metadata for %q is %v old; recalculating", folder, age)
			s.recalcCounts()
		}
		return nil
	})

	return &s
}

func (s *FileSet) recalcCounts() {
	s.meta = newMetadataTracker()

	s.db.tm.inWriteTransaction(func(t transaction) error {
		s.db.checkGlobals(t, []byte(s.folder), s.meta)

		var deviceID protocol.DeviceID
		s.db.withAllFolderTruncated(t, []byte(s.folder), func(device []byte, f FileInfoTruncated) bool {
			copy(deviceID[:], device)
			s.meta.addFile(deviceID, f)
			return true
		})

		s.meta.SetCreated()
		s.meta.toDB(t, s.db, []byte(s.folder))
		return nil
	})
}

func (s *FileSet) Drop(device protocol.DeviceID) {
	l.Debugf("%s Drop(%v)", s.folder, device)

	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	s.db.tm.inWriteTransaction(func(t transaction) error {
		s.db.dropDeviceFolder(t, device[:], []byte(s.folder), s.meta)

		if device == protocol.LocalDeviceID {
			s.blockmap.Drop(t)
			s.meta.resetCounts(device)
			// We deliberately do not reset the sequence number here. Dropping
			// all files for the local device ID only happens in testing - which
			// expects the sequence to be retained, like an old Replace() of all
			// files would do. However, if we ever did it "in production" we
			// would anyway want to retain the sequence for delta indexes to be
			// happy.
		} else {
			// Here, on the other hand, we want to make sure that any file
			// announced from the remote is newer than our current sequence
			// number.
			s.meta.resetAll(device)
		}

		s.meta.toDB(t, s.db, []byte(s.folder))
		return nil
	})
}

func (s *FileSet) Update(device protocol.DeviceID, fs []protocol.FileInfo) {
	l.Debugf("%s Update(%v, [%d])", s.folder, device, len(fs))

	// do not modify fs in place, it is still used in outer scope
	fs = append([]protocol.FileInfo(nil), fs...)

	normalizeFilenames(fs)

	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	s.db.tm.inWriteTransaction(func(t transaction) error {
		if device == protocol.LocalDeviceID {
			discards := make([]protocol.FileInfo, 0, len(fs))
			updates := make([]protocol.FileInfo, 0, len(fs))
			// db.UpdateFiles will sort unchanged files out -> save one db lookup
			// filter slice according to https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
			oldFs := fs
			fs = fs[:0]
			var dk []byte
			folder := []byte(s.folder)
			for _, nf := range oldFs {
				dk = s.db.deviceKeyInto(t, dk, folder, device[:], []byte(osutil.NormalizedFilename(nf.Name)))
				ef, ok := getFileKeyed(t, dk)
				if ok && ef.Version.Equal(nf.Version) && ef.IsInvalid() == nf.IsInvalid() {
					continue
				}

				nf.Sequence = s.meta.nextSeq(protocol.LocalDeviceID)
				fs = append(fs, nf)

				if ok {
					discards = append(discards, ef)
				}
				updates = append(updates, nf)
			}
			s.blockmap.Discard(t, discards)
			s.blockmap.Update(t, updates)
			s.db.removeSequences(t, folder, discards)
			s.db.addSequences(t, folder, updates)
		}

		s.db.updateFiles(t, []byte(s.folder), device[:], fs, s.meta)
		s.meta.toDB(t, s.db, []byte(s.folder))
		return nil
	})
}

func (s *FileSet) WithNeed(device protocol.DeviceID, fn Iterator) {
	l.Debugf("%s WithNeed(%v)", s.folder, device)
	s.db.tm.inReadTransaction(func(t transaction) error {
		s.db.withNeed(t, []byte(s.folder), device[:], false, nativeFileIterator(fn))
		return nil
	})
}

func (s *FileSet) WithNeedTruncated(device protocol.DeviceID, fn Iterator) {
	l.Debugf("%s WithNeedTruncated(%v)", s.folder, device)
	s.db.tm.inReadTransaction(func(t transaction) error {
		s.db.withNeed(t, []byte(s.folder), device[:], true, nativeFileIterator(fn))
		return nil
	})
}

func (s *FileSet) WithHave(device protocol.DeviceID, fn Iterator) {
	l.Debugf("%s WithHave(%v)", s.folder, device)
	s.db.tm.inReadTransaction(func(t transaction) error {
		s.db.withHave(t, []byte(s.folder), device[:], nil, false, nativeFileIterator(fn))
		return nil
	})
}

func (s *FileSet) WithHaveTruncated(device protocol.DeviceID, fn Iterator) {
	l.Debugf("%s WithHaveTruncated(%v)", s.folder, device)
	s.db.tm.inReadTransaction(func(t transaction) error {
		s.db.withHave(t, []byte(s.folder), device[:], nil, true, nativeFileIterator(fn))
		return nil
	})
}

func (s *FileSet) WithHaveSequence(startSeq int64, fn Iterator) {
	l.Debugf("%s WithHaveSequence(%v)", s.folder, startSeq)
	s.db.tm.inReadTransaction(func(t transaction) error {
		s.db.withHaveSequence(t, []byte(s.folder), startSeq, nativeFileIterator(fn))
		return nil
	})
}

// Except for an item with a path equal to prefix, only children of prefix are iterated.
// E.g. for prefix "dir", "dir/file" is iterated, but "dir.file" is not.
func (s *FileSet) WithPrefixedHaveTruncated(device protocol.DeviceID, prefix string, fn Iterator) {
	l.Debugf(`%s WithPrefixedHaveTruncated(%v, "%v")`, s.folder, device, prefix)
	s.db.tm.inReadTransaction(func(t transaction) error {
		s.db.withHave(t, []byte(s.folder), device[:], []byte(osutil.NormalizedFilename(prefix)), true, nativeFileIterator(fn))
		return nil
	})
}

func (s *FileSet) WithGlobal(fn Iterator) {
	l.Debugf("%s WithGlobal()", s.folder)
	s.db.tm.inReadTransaction(func(t transaction) error {
		s.db.withGlobal(t, []byte(s.folder), nil, false, nativeFileIterator(fn))
		return nil
	})
}

func (s *FileSet) WithGlobalTruncated(fn Iterator) {
	l.Debugf("%s WithGlobalTruncated()", s.folder)
	s.db.tm.inReadTransaction(func(t transaction) error {
		s.db.withGlobal(t, []byte(s.folder), nil, true, nativeFileIterator(fn))
		return nil
	})
}

// Except for an item with a path equal to prefix, only children of prefix are iterated.
// E.g. for prefix "dir", "dir/file" is iterated, but "dir.file" is not.
func (s *FileSet) WithPrefixedGlobalTruncated(prefix string, fn Iterator) {
	l.Debugf(`%s WithPrefixedGlobalTruncated("%v")`, s.folder, prefix)
	s.db.tm.inReadTransaction(func(t transaction) error {
		s.db.withGlobal(t, []byte(s.folder), []byte(osutil.NormalizedFilename(prefix)), true, nativeFileIterator(fn))
		return nil
	})
}

func (s *FileSet) Get(device protocol.DeviceID, file string) (protocol.FileInfo, bool) {
	var f protocol.FileInfo
	var ok bool
	s.db.tm.withoutTransaction(func(t transaction) error {
		f, ok = s.db.getFile(t, []byte(s.folder), device[:], []byte(osutil.NormalizedFilename(file)))
		return nil
	})
	f.Name = osutil.NativeFilename(f.Name)
	return f, ok
}

func (s *FileSet) GetGlobal(file string) (protocol.FileInfo, bool) {
	var fi FileIntf
	var ok bool
	s.db.tm.withoutTransaction(func(t transaction) error {
		fi, ok = s.db.getGlobal(t, []byte(s.folder), []byte(osutil.NormalizedFilename(file)), false)
		return nil
	})
	if !ok {
		return protocol.FileInfo{}, false
	}
	f := fi.(protocol.FileInfo)
	f.Name = osutil.NativeFilename(f.Name)
	return f, true
}

func (s *FileSet) GetGlobalTruncated(file string) (FileInfoTruncated, bool) {
	var fi FileIntf
	var ok bool
	s.db.tm.withoutTransaction(func(t transaction) error {
		fi, ok = s.db.getGlobal(t, []byte(s.folder), []byte(osutil.NormalizedFilename(file)), true)
		return nil
	})
	if !ok {
		return FileInfoTruncated{}, false
	}
	f := fi.(FileInfoTruncated)
	f.Name = osutil.NativeFilename(f.Name)
	return f, true
}

func (s *FileSet) Availability(file string) []protocol.DeviceID {
	var res []protocol.DeviceID
	s.db.tm.withoutTransaction(func(t transaction) error {
		res = s.db.availability(t, []byte(s.folder), []byte(osutil.NormalizedFilename(file)))
		return nil
	})
	return res
}

func (s *FileSet) Sequence(device protocol.DeviceID) int64 {
	return s.meta.Counts(device, 0).Sequence
}

func (s *FileSet) LocalSize() Counts {
	local := s.meta.Counts(protocol.LocalDeviceID, 0)
	recvOnlyChanged := s.meta.Counts(protocol.LocalDeviceID, protocol.FlagLocalReceiveOnly)
	return local.Add(recvOnlyChanged)
}

func (s *FileSet) ReceiveOnlyChangedSize() Counts {
	return s.meta.Counts(protocol.LocalDeviceID, protocol.FlagLocalReceiveOnly)
}

func (s *FileSet) GlobalSize() Counts {
	global := s.meta.Counts(protocol.GlobalDeviceID, 0)
	recvOnlyChanged := s.meta.Counts(protocol.GlobalDeviceID, protocol.FlagLocalReceiveOnly)
	return global.Add(recvOnlyChanged)
}

func (s *FileSet) IndexID(device protocol.DeviceID) protocol.IndexID {
	var id protocol.IndexID
	s.db.tm.inWriteTransaction(func(t transaction) error {
		id := s.db.getIndexID(t, device[:], []byte(s.folder))
		if id == 0 && device == protocol.LocalDeviceID {
			// No index ID set yet. We create one now.
			id = protocol.NewIndexID()
			s.db.setIndexID(t, device[:], []byte(s.folder), id)
		}
		return nil
	})
	return id
}

func (s *FileSet) SetIndexID(device protocol.DeviceID, id protocol.IndexID) {
	if device == protocol.LocalDeviceID {
		panic("do not explicitly set index ID for local device")
	}
	s.db.tm.withoutTransaction(func(t transaction) error {
		s.db.setIndexID(t, device[:], []byte(s.folder), id)
		return nil
	})
}

func (s *FileSet) MtimeFS() *fs.MtimeFS {
	var prefix []byte
	var kv *NamespacedKV
	s.db.tm.withoutTransaction(func(t transaction) error {
		prefix = s.db.mtimesKey(t, []byte(s.folder))
		kv = NewNamespacedKV(t, string(prefix))
		return nil
	})
	return fs.NewMtimeFS(s.fs, kv)
}

func (s *FileSet) ListDevices() []protocol.DeviceID {
	return s.meta.devices()
}

// DropFolder clears out all information related to the given folder from the
// database.
func DropFolder(db *Instance, folder string) {
	db.tm.inWriteTransaction(func(t transaction) error {
		db.dropFolder(t, []byte(folder))
		db.dropMtimes(t, []byte(folder))
		db.dropFolderMeta(t, []byte(folder))
		bm := &BlockMap{
			folder: db.folderIdx.ID(t, []byte(folder)),
		}
		bm.Drop(t)
		return nil
	})
}

func normalizeFilenames(fs []protocol.FileInfo) {
	for i := range fs {
		fs[i].Name = osutil.NormalizedFilename(fs[i].Name)
	}
}

func nativeFileIterator(fn Iterator) Iterator {
	return func(fi FileIntf) bool {
		switch f := fi.(type) {
		case protocol.FileInfo:
			f.Name = osutil.NativeFilename(f.Name)
			return fn(f)
		case FileInfoTruncated:
			f.Name = osutil.NativeFilename(f.Name)
			return fn(f)
		default:
			panic("unknown interface type")
		}
	}
}
