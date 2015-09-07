// Copyright (C) 2015 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package model

import (
	"bytes"

	"github.com/syncthing/protocol"
	"github.com/syncthing/syncthing/lib/sync"
)

type deviceFolderDownloadState struct {
	mut   sync.RWMutex
	files map[string][]protocol.BlockInfo
}

func (p *deviceFolderDownloadState) Has(file string, block protocol.BlockInfo) bool {
	p.mut.RLock()
	blocks, ok := p.files[file]
	p.mut.RUnlock()
	if !ok {
		return false
	}

	// Special case, used to cover /rest/db/file
	if block.Offset == 0 && block.Size == 0 && block.Hash == nil {
		return true
	}
	for _, existingBlock := range blocks {
		if existingBlock.Offset == block.Offset && block.Size == existingBlock.Size && bytes.Equal(existingBlock.Hash, block.Hash) {
			return true
		}
	}
	return false
}

func (p *deviceFolderDownloadState) Update(updates []protocol.FileDownloadProgressUpdate) {
	// Could acquire lock in the loop to reduce contention, but we shouldn't be
	// getting many updates at a time, hence probably not worth it.
	p.mut.Lock()
	for _, update := range updates {
		if update.UpdateType == protocol.UpdateTypeForget {
			delete(p.files, update.Name)
		} else if update.UpdateType == protocol.UpdateTypeAppend {
			blocks, ok := p.files[update.Name]
			if !ok {
				blocks = make([]protocol.BlockInfo, 0, len(update.Blocks))
			}
			p.files[update.Name] = append(blocks, update.Blocks...)
		}
	}
	p.mut.Unlock()
}

type deviceDownloadState struct {
	mut     sync.RWMutex
	folders map[string]deviceFolderDownloadState
}

func (t *deviceDownloadState) Update(folder string, updates []protocol.FileDownloadProgressUpdate) {
	t.mut.RLock()
	f, ok := t.folders[folder]
	t.mut.RUnlock()

	if !ok {
		f = deviceFolderDownloadState{
			mut:   sync.NewRWMutex(),
			files: make(map[string][]protocol.BlockInfo),
		}
		t.mut.Lock()
		t.folders[folder] = f
		t.mut.Unlock()
	}

	f.Update(updates)
}

func (t *deviceDownloadState) Has(folder, file string, block protocol.BlockInfo) bool {
	t.mut.RLock()
	f, ok := t.folders[folder]
	t.mut.RUnlock()

	if !ok {
		return false
	}

	return f.Has(file, block)
}

func newdeviceDownloadState() *deviceDownloadState {
	return &deviceDownloadState{
		mut:     sync.NewRWMutex(),
		folders: make(map[string]deviceFolderDownloadState),
	}
}
