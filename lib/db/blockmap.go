// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package db

import (
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	"github.com/syncthing/syncthing/lib/osutil"
	"github.com/syncthing/syncthing/lib/protocol"

	"github.com/syndtr/goleveldb/leveldb/util"
)

var blockFinder *BlockFinder

const maxBatchSize = 1000

type BlockMap struct {
	folder uint32
}

func NewBlockMap(folder uint32) *BlockMap {
	return &BlockMap{
		folder: folder,
	}
}

// Add files to the block map, ignoring any deleted or invalid files.
func (m *BlockMap) Add(t transaction, files []protocol.FileInfo) error {
	buf := make([]byte, 4)
	var key []byte
	for _, file := range files {
		if file.IsDirectory() || file.IsDeleted() || file.IsInvalid() {
			continue
		}

		for i, block := range file.Blocks {
			binary.BigEndian.PutUint32(buf, uint32(i))
			key = m.blockKeyInto(key, block.Hash, file.Name)
			if err := t.Put(key, buf, nil); err != nil {
				return errors.Wrap(err, "blockmap add")
			}
		}
	}
	return nil
}

// Update block map state, removing any deleted or invalid files.
func (m *BlockMap) Update(t transaction, files []protocol.FileInfo) error {
	buf := make([]byte, 4)
	var key []byte
	for _, file := range files {
		if file.IsDirectory() {
			continue
		}

		if file.IsDeleted() || file.IsInvalid() {
			for _, block := range file.Blocks {
				key = m.blockKeyInto(key, block.Hash, file.Name)
				if err := t.Delete(key, nil); err != nil {
					return errors.Wrap(err, "blockmap update")
				}
			}
			continue
		}

		for i, block := range file.Blocks {
			binary.BigEndian.PutUint32(buf, uint32(i))
			key = m.blockKeyInto(key, block.Hash, file.Name)
			if err := t.Put(key, buf, nil); err != nil {
				return errors.Wrap(err, "blockmap update")
			}
		}
	}
	return nil
}

// Discard block map state, removing the given files
func (m *BlockMap) Discard(t transaction, files []protocol.FileInfo) error {
	var key []byte
	for _, file := range files {
		for _, block := range file.Blocks {
			key = m.blockKeyInto(key, block.Hash, file.Name)
			if err := t.Delete(key, nil); err != nil {
				return errors.Wrap(err, "blockmap discard")
			}
		}
	}
	return nil
}

// Drop block map, removing all entries related to this block map from the db.
func (m *BlockMap) Drop(t transaction) error {
	iter := t.NewIterator(util.BytesPrefix(m.blockKeyInto(nil, nil, "")[:keyPrefixLen+keyFolderLen]), nil)
	defer iter.Release()
	for iter.Next() {
		if err := t.Delete(iter.Key(), nil); err != nil {
			return errors.Wrap(err, "blockmap drop")
		}
	}
	if iter.Error() != nil {
		return errors.Wrap(iter.Error(), "blockmap drop")
	}
	return nil
}

func (m *BlockMap) blockKeyInto(o, hash []byte, file string) []byte {
	return blockKeyInto(o, hash, m.folder, file)
}

type BlockFinder struct {
	db *Instance
}

func NewBlockFinder(db *Instance) *BlockFinder {
	if blockFinder != nil {
		return blockFinder
	}

	f := &BlockFinder{
		db: db,
	}

	return f
}

func (f *BlockFinder) String() string {
	return fmt.Sprintf("BlockFinder@%p", f)
}

// Iterate takes an iterator function which iterates over all matching blocks
// for the given hash. The iterator function has to return either true (if
// they are happy with the block) or false to continue iterating for whatever
// reason. The iterator finally returns the result, whether or not a
// satisfying block was eventually found.
func (f *BlockFinder) Iterate(folders []string, hash []byte, iterFn func(string, string, int32) bool) bool {
	var key []byte
	found := false
	f.db.tm.withoutTransaction(func(t transaction) error {
	outer:
		for _, folder := range folders {
			folderID := f.db.folderIdx.ID(t, []byte(folder))
			key = blockKeyInto(key, hash, folderID, "")
			iter := t.NewIterator(util.BytesPrefix(key), nil)
			defer iter.Release()

			for iter.Next() && iter.Error() == nil {
				file := blockKeyName(iter.Key())
				index := int32(binary.BigEndian.Uint32(iter.Value()))
				if iterFn(folder, osutil.NativeFilename(file), index) {
					found = true
					break outer
				}
			}
		}
		return nil
	})
	return found
}

// m.blockKey returns a byte slice encoding the following information:
//	   keyTypeBlock (1 byte)
//	   folder (4 bytes)
//	   block hash (32 bytes)
//	   file name (variable size)
func blockKeyInto(o, hash []byte, folder uint32, file string) []byte {
	reqLen := keyPrefixLen + keyFolderLen + keyHashLen + len(file)
	if cap(o) < reqLen {
		o = make([]byte, reqLen)
	} else {
		o = o[:reqLen]
	}
	o[0] = KeyTypeBlock
	binary.BigEndian.PutUint32(o[keyPrefixLen:], folder)
	copy(o[keyPrefixLen+keyFolderLen:], hash)
	copy(o[keyPrefixLen+keyFolderLen+keyHashLen:], []byte(file))
	return o
}

// blockKeyName returns the file name from the block key
func blockKeyName(data []byte) string {
	if len(data) < keyPrefixLen+keyFolderLen+keyHashLen+1 {
		panic("Incorrect key length")
	}
	if data[0] != KeyTypeBlock {
		panic("Incorrect key type")
	}

	file := string(data[keyPrefixLen+keyFolderLen+keyHashLen:])
	return file
}
