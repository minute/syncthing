// Copyright (C) 2018 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package db

import (
	"fmt"
	"strings"

	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// List of all dbVersion to dbMinSyncthingVersion pairs for convenience
//   0: v0.14.0
//   1: v0.14.46
//   2: v0.14.48
//   3: v0.14.49
//   4: v0.14.49
//   5: v0.14.49
//   6: v0.14.50
const (
	dbVersion             = 6
	dbMinSyncthingVersion = "v0.14.50"
)

type databaseDowngradeError struct {
	minSyncthingVersion string
}

func (e databaseDowngradeError) Error() string {
	if e.minSyncthingVersion == "" {
		return "newer Syncthing required"
	}
	return fmt.Sprintf("Syncthing %s required", e.minSyncthingVersion)
}

func (db *Instance) updateSchema(t transaction) error {
	miscDB := NewNamespacedKV(db, string(KeyTypeMiscData))
	prevVersion, _ := miscDB.Int64("dbVersion")

	if prevVersion > dbVersion {
		err := databaseDowngradeError{}
		if minSyncthingVersion, ok := miscDB.String("dbMinSyncthingVersion"); ok {
			err.minSyncthingVersion = minSyncthingVersion
		}
		return err
	}

	if prevVersion == dbVersion {
		return nil
	}

	if prevVersion < 1 {
		db.updateSchema0to1(t)
	}
	if prevVersion < 2 {
		db.updateSchema1to2(t)
	}
	if prevVersion < 3 {
		db.updateSchema2to3(t)
	}
	// This update fixes problems existing in versions 3 and 4
	if prevVersion == 3 || prevVersion == 4 {
		db.updateSchemaTo5(t)
	}
	if prevVersion < 6 {
		db.updateSchema5to6(t)
	}

	miscDB.PutInt64("dbVersion", dbVersion)
	miscDB.PutString("dbMinSyncthingVersion", dbMinSyncthingVersion)

	return nil
}

func (db *Instance) updateSchema0to1(t transaction) {
	dbi := t.NewIterator(util.BytesPrefix([]byte{KeyTypeDevice}), nil)
	defer dbi.Release()

	symlinkConv := 0
	changedFolders := make(map[string]struct{})
	ignAdded := 0
	meta := newMetadataTracker() // dummy metadata tracker
	var gk []byte

	for dbi.Next() {
		folder := db.deviceKeyFolder(dbi.Key())
		device := db.deviceKeyDevice(dbi.Key())
		name := db.deviceKeyName(dbi.Key())

		// Remove files with absolute path (see #4799)
		if strings.HasPrefix(string(name), "/") {
			if _, ok := changedFolders[string(folder)]; !ok {
				changedFolders[string(folder)] = struct{}{}
			}
			gk = db.globalKeyInto(t, gk, folder, name)
			removeFromGlobal(t, gk, folder, device, nil, nil)
			t.Delete(dbi.Key(), nil)
			continue
		}

		// Change SYMLINK_FILE and SYMLINK_DIRECTORY types to the current SYMLINK
		// type (previously SYMLINK_UNKNOWN). It does this for all devices, both
		// local and remote, and does not reset delta indexes. It shouldn't really
		// matter what the symlink type is, but this cleans it up for a possible
		// future when SYMLINK_FILE and SYMLINK_DIRECTORY are no longer understood.
		var f protocol.FileInfo
		if err := f.Unmarshal(dbi.Value()); err != nil {
			// probably can't happen
			continue
		}
		if f.Type == protocol.FileInfoTypeDeprecatedSymlinkDirectory || f.Type == protocol.FileInfoTypeDeprecatedSymlinkFile {
			f.Type = protocol.FileInfoTypeSymlink
			bs, err := f.Marshal()
			if err != nil {
				panic("can't happen: " + err.Error())
			}
			t.Put(dbi.Key(), bs, nil)
			symlinkConv++
		}

		// Add invalid files to global list
		if f.IsInvalid() {
			gk = db.globalKeyInto(t, gk, folder, name)
			if updateGlobal(t, gk, folder, device, f, meta) {
				if _, ok := changedFolders[string(folder)]; !ok {
					changedFolders[string(folder)] = struct{}{}
				}
				ignAdded++
			}
		}
	}

	for folder := range changedFolders {
		db.dropFolderMeta(t, []byte(folder))
	}
}

// updateSchema1to2 introduces a sequenceKey->deviceKey bucket for local items
// to allow iteration in sequence order (simplifies sending indexes).
func (db *Instance) updateSchema1to2(t transaction) {
	var sk []byte
	var dk []byte
	for _, folderStr := range db.ListFolders() {
		folder := []byte(folderStr)
		db.withHave(t, folder, protocol.LocalDeviceID[:], nil, true, func(f FileIntf) bool {
			sk = db.sequenceKeyInto(t, sk, folder, f.SequenceNo())
			dk = db.deviceKeyInto(t, dk, folder, protocol.LocalDeviceID[:], []byte(f.FileName()))
			t.Put(sk, dk, nil)
			return true
		})
	}
}

// updateSchema2to3 introduces a needKey->nil bucket for locally needed files.
func (db *Instance) updateSchema2to3(t transaction) {
	var nk []byte
	var dk []byte
	for _, folderStr := range db.ListFolders() {
		folder := []byte(folderStr)
		db.withGlobal(t, folder, nil, true, func(f FileIntf) bool {
			name := []byte(f.FileName())
			dk = db.deviceKeyInto(t, dk, folder, protocol.LocalDeviceID[:], name)
			var v protocol.Vector
			haveFile, ok := db.getFileTrunc(t, dk, true)
			if ok {
				v = haveFile.FileVersion()
			}
			if !need(f, ok, v) {
				return true
			}
			nk = db.needKeyInto(t, nk, folder, []byte(f.FileName()))
			t.Put(nk, nil, nil)
			return true
		})
	}
}

// updateSchemaTo5 resets the need bucket due to bugs existing in the v0.14.49
// release candidates (dbVersion 3 and 4)
// https://github.com/syncthing/syncthing/issues/5007
// https://github.com/syncthing/syncthing/issues/5053
func (db *Instance) updateSchemaTo5(t transaction) {
	var nk []byte
	for _, folderStr := range db.ListFolders() {
		nk = db.needKeyInto(t, nk, []byte(folderStr), nil)
		deleteKeyPrefix(t, nk[:keyPrefixLen+keyFolderLen])
	}

	db.updateSchema2to3(t)
}

func (db *Instance) updateSchema5to6(t transaction) {
	// For every local file with the Invalid bit set, clear the Invalid bit and
	// set LocalFlags = FlagLocalIgnored.

	var dk []byte

	for _, folderStr := range db.ListFolders() {
		folder := []byte(folderStr)
		db.withHave(t, folder, protocol.LocalDeviceID[:], nil, false, func(f FileIntf) bool {
			if !f.IsInvalid() {
				return true
			}

			fi := f.(protocol.FileInfo)
			fi.RawInvalid = false
			fi.LocalFlags = protocol.FlagLocalIgnored
			bs, _ := fi.Marshal()

			dk = db.deviceKeyInto(t, dk, folder, protocol.LocalDeviceID[:], []byte(fi.Name))
			t.Put(dk, bs, nil)

			return true
		})
	}
}
