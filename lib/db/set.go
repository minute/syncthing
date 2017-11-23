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
	stdsync "sync"
	"sync/atomic"

	fs "github.com/syncthing/syncthing/lib/fs"
	"github.com/syncthing/syncthing/lib/osutil"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/sync"
)

type FileSet struct {
	sequence   int64 // Our local sequence number
	folder     string
	fs         fs.Filesystem
	db         *Instance
	blockmap   *BlockMap
	localSize  sizeTracker
	globalSize sizeTracker
	foldCase   bool

	remoteSequence map[protocol.DeviceID]int64 // Highest seen sequence numbers for other devices
	updateMutex    sync.Mutex                  // protects remoteSequence and database updates
}

// FileIntf is the set of methods implemented by both protocol.FileInfo and
// FileInfoTruncated.
type FileIntf interface {
	FileSize() int64
	FileName() string
	IsDeleted() bool
	IsInvalid() bool
	IsDirectory() bool
	IsSymlink() bool
	HasPermissionBits() bool
}

// The Iterator is called with either a protocol.FileInfo or a
// FileInfoTruncated (depending on the method) and returns true to
// continue iteration, false to stop.
type Iterator func(f FileIntf) bool

type Counts struct {
	Files       int
	Directories int
	Symlinks    int
	Deleted     int
	Bytes       int64
}

type sizeTracker struct {
	Counts
	mut stdsync.Mutex
}

func (s *sizeTracker) addFile(f FileIntf) {
	if f.IsInvalid() {
		return
	}

	s.mut.Lock()
	switch {
	case f.IsDeleted():
		s.Deleted++
	case f.IsDirectory() && !f.IsSymlink():
		s.Directories++
	case f.IsSymlink():
		s.Symlinks++
	default:
		s.Files++
	}
	s.Bytes += f.FileSize()
	s.mut.Unlock()
}

func (s *sizeTracker) removeFile(f FileIntf) {
	if f.IsInvalid() {
		return
	}

	s.mut.Lock()
	switch {
	case f.IsDeleted():
		s.Deleted--
	case f.IsDirectory() && !f.IsSymlink():
		s.Directories--
	case f.IsSymlink():
		s.Symlinks--
	default:
		s.Files--
	}
	s.Bytes -= f.FileSize()
	if s.Deleted < 0 || s.Files < 0 || s.Directories < 0 || s.Symlinks < 0 {
		panic("bug: removed more than added")
	}
	s.mut.Unlock()
}

func (s *sizeTracker) reset() {
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Counts = Counts{}
}

func (s *sizeTracker) Size() Counts {
	s.mut.Lock()
	defer s.mut.Unlock()
	return s.Counts
}

type Option func(*FileSet)

func WithCaseInsensitive(v bool) Option {
	return func(s *FileSet) {
		s.foldCase = v
	}
}

func NewFileSet(folder string, fs fs.Filesystem, db *Instance, options ...Option) *FileSet {
	var s = &FileSet{
		remoteSequence: make(map[protocol.DeviceID]int64),
		folder:         folder,
		fs:             fs,
		db:             db,
		blockmap:       NewBlockMap(db, db.folderIdx.ID([]byte(folder))),
		updateMutex:    sync.NewMutex(),
	}

	for _, fn := range options {
		fn(s)
	}

	s.db.checkGlobals([]byte(folder), &s.globalSize)

	var deviceID protocol.DeviceID
	s.db.withAllFolderTruncated([]byte(folder), func(device []byte, f FileInfoTruncated) bool {
		copy(deviceID[:], device)
		if deviceID == protocol.LocalDeviceID {
			if f.Sequence > s.sequence {
				s.sequence = f.Sequence
			}
			s.localSize.addFile(f)
		} else if f.Sequence > s.remoteSequence[deviceID] {
			s.remoteSequence[deviceID] = f.Sequence
		}
		return true
	})
	l.Debugf("loaded sequence for %q: %#v", folder, s.sequence)

	return s
}

func (s *FileSet) Drop(device protocol.DeviceID) {
	l.Debugf("%s Drop(%v)", s.folder, device)

	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	s.db.dropDeviceFolder(device[:], []byte(s.folder), &s.globalSize)

	if device == protocol.LocalDeviceID {
		s.blockmap.Drop()
		s.localSize.reset()
		// We deliberately do not reset s.sequence here. Dropping all files
		// for the local device ID only happens in testing - which expects
		// the sequence to be retained, like an old Replace() of all files
		// would do. However, if we ever did it "in production" we would
		// anyway want to retain the sequence for delta indexes to be happy.
	} else {
		// Here, on the other hand, we want to make sure that any file
		// announced from the remote is newer than our current sequence
		// number.
		s.remoteSequence[device] = 0
	}
}

func (s *FileSet) Update(device protocol.DeviceID, fs []protocol.FileInfo) {
	l.Debugf("%s Update(%v, [%d])", s.folder, device, len(fs))
	normalizeFilenames(fs)

	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	s.updateLocked(device, fs)
}

func (s *FileSet) updateLocked(device protocol.DeviceID, files []protocol.FileInfo) {
	// names must be normalized and the lock held

	if device == protocol.LocalDeviceID {
		discards := make([]protocol.FileInfo, 0, len(files))
		updates := make([]protocol.FileInfo, 0, len(files))
		// db.UpdateFiles will sort unchanged files out -> save one db lookup
		// filter slice according to https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
		oldFs := files
		files = files[:0]
		for _, nf := range oldFs {
			name := []byte(nf.Name)
			if s.foldCase {
				name = []byte(fs.UnicodeLowercase(string(name)))
			}
			ef, ok := s.db.getFile([]byte(s.folder), device[:], name)
			if ok && ef.Version.Equal(nf.Version) && ef.Invalid == nf.Invalid {
				continue
			}

			nf.Sequence = atomic.AddInt64(&s.sequence, 1)
			files = append(files, nf)

			if ok {
				discards = append(discards, ef)
			}
			updates = append(updates, nf)
		}
		s.blockmap.Discard(discards)
		s.blockmap.Update(updates)
	} else {
		s.remoteSequence[device] = maxSequence(files)
	}
	s.db.updateFiles([]byte(s.folder), device[:], files, &s.localSize, &s.globalSize, s.foldCase)
}

func (s *FileSet) WithNeed(device protocol.DeviceID, fn Iterator) {
	l.Debugf("%s WithNeed(%v)", s.folder, device)
	s.db.withNeed([]byte(s.folder), device[:], false, false, nativeFileIterator(fn))
}

func (s *FileSet) WithNeedTruncated(device protocol.DeviceID, fn Iterator) {
	l.Debugf("%s WithNeedTruncated(%v)", s.folder, device)
	s.db.withNeed([]byte(s.folder), device[:], true, false, nativeFileIterator(fn))
}

// WithNeedOrInvalid considers all invalid files as needed, regardless of their version
// (e.g. for pulling when ignore patterns changed)
func (s *FileSet) WithNeedOrInvalid(device protocol.DeviceID, fn Iterator) {
	l.Debugf("%s WithNeedExcludingInvalid(%v)", s.folder, device)
	s.db.withNeed([]byte(s.folder), device[:], false, true, nativeFileIterator(fn))
}

func (s *FileSet) WithNeedOrInvalidTruncated(device protocol.DeviceID, fn Iterator) {
	l.Debugf("%s WithNeedExcludingInvalidTruncated(%v)", s.folder, device)
	s.db.withNeed([]byte(s.folder), device[:], true, true, nativeFileIterator(fn))
}

func (s *FileSet) WithHave(device protocol.DeviceID, fn Iterator) {
	l.Debugf("%s WithHave(%v)", s.folder, device)
	s.db.withHave([]byte(s.folder), device[:], nil, false, nativeFileIterator(fn))
}

func (s *FileSet) WithHaveTruncated(device protocol.DeviceID, fn Iterator) {
	l.Debugf("%s WithHaveTruncated(%v)", s.folder, device)
	s.db.withHave([]byte(s.folder), device[:], nil, true, nativeFileIterator(fn))
}

func (s *FileSet) WithPrefixedHaveTruncated(device protocol.DeviceID, prefix string, fn Iterator) {
	if s.foldCase {
		prefix = fs.UnicodeLowercase(prefix)
	}
	l.Debugf("%s WithPrefixedHaveTruncated(%v)", s.folder, device)
	s.db.withHave([]byte(s.folder), device[:], []byte(osutil.NormalizedFilename(prefix)), true, nativeFileIterator(fn))
}
func (s *FileSet) WithGlobal(fn Iterator) {
	l.Debugf("%s WithGlobal()", s.folder)
	s.db.withGlobal([]byte(s.folder), nil, false, nativeFileIterator(fn))
}

func (s *FileSet) WithGlobalTruncated(fn Iterator) {
	l.Debugf("%s WithGlobalTruncated()", s.folder)
	s.db.withGlobal([]byte(s.folder), nil, true, nativeFileIterator(fn))
}

func (s *FileSet) WithPrefixedGlobalTruncated(prefix string, fn Iterator) {
	if s.foldCase {
		prefix = fs.UnicodeLowercase(prefix)
	}
	l.Debugf("%s WithPrefixedGlobalTruncated()", s.folder, prefix)
	s.db.withGlobal([]byte(s.folder), []byte(osutil.NormalizedFilename(prefix)), true, nativeFileIterator(fn))
}

func (s *FileSet) Get(device protocol.DeviceID, file string) (protocol.FileInfo, bool) {
	if s.foldCase {
		file = fs.UnicodeLowercase(file)
	}
	f, ok := s.db.getFile([]byte(s.folder), device[:], []byte(osutil.NormalizedFilename(file)))
	f.Name = osutil.NativeFilename(f.Name)
	return f, ok
}

func (s *FileSet) GetGlobal(file string) (protocol.FileInfo, bool) {
	if s.foldCase {
		file = fs.UnicodeLowercase(file)
	}
	fi, ok := s.db.getGlobal([]byte(s.folder), []byte(osutil.NormalizedFilename(file)), false)
	if !ok {
		return protocol.FileInfo{}, false
	}
	f := fi.(protocol.FileInfo)
	f.Name = osutil.NativeFilename(f.Name)
	return f, true
}

func (s *FileSet) GetGlobalTruncated(file string) (FileInfoTruncated, bool) {
	if s.foldCase {
		file = fs.UnicodeLowercase(file)
	}
	fi, ok := s.db.getGlobal([]byte(s.folder), []byte(osutil.NormalizedFilename(file)), true)
	if !ok {
		return FileInfoTruncated{}, false
	}
	f := fi.(FileInfoTruncated)
	f.Name = osutil.NativeFilename(f.Name)
	return f, true
}

func (s *FileSet) Availability(file string) []protocol.DeviceID {
	if s.foldCase {
		file = fs.UnicodeLowercase(file)
	}
	return s.db.availability([]byte(s.folder), []byte(osutil.NormalizedFilename(file)))
}

func (s *FileSet) Sequence(device protocol.DeviceID) int64 {
	if device == protocol.LocalDeviceID {
		return atomic.LoadInt64(&s.sequence)
	}

	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	return s.remoteSequence[device]
}

func (s *FileSet) LocalSize() Counts {
	return s.localSize.Size()
}

func (s *FileSet) GlobalSize() Counts {
	return s.globalSize.Size()
}

func (s *FileSet) IndexID(device protocol.DeviceID) protocol.IndexID {
	id := s.db.getIndexID(device[:], []byte(s.folder))
	if id == 0 && device == protocol.LocalDeviceID {
		// No index ID set yet. We create one now.
		id = protocol.NewIndexID()
		s.db.setIndexID(device[:], []byte(s.folder), id)
	}
	return id
}

func (s *FileSet) SetIndexID(device protocol.DeviceID, id protocol.IndexID) {
	if device == protocol.LocalDeviceID {
		panic("do not explicitly set index ID for local device")
	}
	s.db.setIndexID(device[:], []byte(s.folder), id)
}

func (s *FileSet) MtimeFS() *fs.MtimeFS {
	prefix := s.db.mtimesKey([]byte(s.folder))
	kv := NewNamespacedKV(s.db, string(prefix))
	return fs.NewMtimeFS(s.fs, kv)
}

func (s *FileSet) ListDevices() []protocol.DeviceID {
	s.updateMutex.Lock()
	devices := make([]protocol.DeviceID, 0, len(s.remoteSequence))
	for id, seq := range s.remoteSequence {
		if seq > 0 {
			devices = append(devices, id)
		}
	}
	s.updateMutex.Unlock()
	return devices
}

// maxSequence returns the highest of the Sequence numbers found in
// the given slice of FileInfos. This should really be the Sequence of
// the last item, but Syncthing v0.14.0 and other implementations may not
// implement update sorting....
func maxSequence(fs []protocol.FileInfo) int64 {
	var max int64
	for _, f := range fs {
		if f.Sequence > max {
			max = f.Sequence
		}
	}
	return max
}

// DropFolder clears out all information related to the given folder from the
// database.
func DropFolder(db *Instance, folder string) {
	db.dropFolder([]byte(folder))
	db.dropMtimes([]byte(folder))
	bm := &BlockMap{
		db:     db,
		folder: db.folderIdx.ID([]byte(folder)),
	}
	bm.Drop()
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
