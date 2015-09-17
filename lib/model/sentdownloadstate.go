// Copyright (C) 2015 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package model

import (
	"time"

	"github.com/syncthing/protocol"
	"github.com/syncthing/syncthing/lib/scanner"
)

// sentFolderDownloadState represents a state of what we've announced as available
// to some remote device for a specific folder.
type sentFolderDownloadState struct {
	fileBlocks  map[string][]protocol.BlockInfo
	fileUpdated map[string]time.Time
}

// update takes a set of currently active sharedPullerStates, and returns a list
// of updates which we need to send to the client to become up to date.
// Any pullers that
func (s *sentFolderDownloadState) update(pullers []*sharedPullerState) []protocol.FileDownloadProgressUpdate {
	var name string
	var updates []protocol.FileDownloadProgressUpdate
	seen := make(map[string]struct{}, len(pullers))

	for _, puller := range pullers {
		name = puller.file.Name

		seen[name] = struct{}{}

		pullerBlocks := puller.Available()
		pullerBlocksUpdated := puller.AvailableUpdated()

		localBlocks := s.fileBlocks[name]
		localBlocksUpdated, ok := s.fileUpdated[name]

		// New file we haven't seen before
		if !ok {
			s.fileBlocks[name] = pullerBlocks
			s.fileUpdated[name] = pullerBlocksUpdated

			updates = append(updates, protocol.FileDownloadProgressUpdate{
				Name:       name,
				UpdateType: protocol.UpdateTypeAppend,
				Blocks:     pullerBlocks,
			})
			continue
		}

		// Existing file we've already sent an update for.
		if pullerBlocksUpdated.Equal(localBlocksUpdated) {
			// The file state hasn't changed, go to next.
			continue
		}

		// Relies on the fact that Available() should always append.
		_, need := scanner.BlockDiff(localBlocks, pullerBlocks)

		s.fileBlocks[name] = append(localBlocks, need...)
		s.fileUpdated[name] = pullerBlocksUpdated

		updates = append(updates, protocol.FileDownloadProgressUpdate{
			Name:       name,
			UpdateType: protocol.UpdateTypeAppend,
			Blocks:     need,
		})
	}

	// For each file that we are tracking, see if there still is a puller for it
	// if not, the file completed or errored out.
	for name := range s.fileBlocks {
		_, ok := seen[name]
		if !ok {
			updates = append(updates, protocol.FileDownloadProgressUpdate{
				Name:       name,
				UpdateType: protocol.UpdateTypeForget,
			})
		}
	}

	return updates
}

// destroy removes all stored state, and returns a set of updates we need to
// dispatch to clean up the state on the remote end.
func (s *sentFolderDownloadState) destroy() []protocol.FileDownloadProgressUpdate {
	updates := make([]protocol.FileDownloadProgressUpdate, 0, len(s.fileBlocks))
	for name := range s.fileBlocks {
		updates = append(updates, protocol.FileDownloadProgressUpdate{
			Name:       name,
			UpdateType: protocol.UpdateTypeForget,
		})
		delete(s.fileBlocks, name)
		delete(s.fileUpdated, name)
	}
	return updates
}

// sentDownloadState represents a state of what we've announced as available
// to some remote device.
type sentDownloadState struct {
	folderStates map[string]sentFolderDownloadState
}

// update receives a folder, and a slice of pullers that are currently available
// for the given folder, and according to the state of what we've seen before
// returns a set of updates which we should send to the remote device to make
// it aware of everything that we currently have available.
func (s *sentDownloadState) update(folder string, pullers []*sharedPullerState) []protocol.FileDownloadProgressUpdate {
	fs, ok := s.folderStates[folder]
	if !ok {
		fs = sentFolderDownloadState{
			fileBlocks:  make(map[string][]protocol.BlockInfo),
			fileUpdated: make(map[string]time.Time),
		}
		s.folderStates[folder] = fs
	}
	return fs.update(pullers)
}

// folders returns a set of folders this state is currently aware off.
func (s *sentDownloadState) folders() []string {
	folders := make([]string, 0, len(s.folderStates))
	for key := range s.folderStates {
		folders = append(folders, key)
	}
	return folders
}

// cleanup cleans up all state related to a folder, and returns a set of updates
// which would clean up the state on the remote device.
func (s *sentDownloadState) cleanup(folder string) []protocol.FileDownloadProgressUpdate {
	fs, ok := s.folderStates[folder]
	if ok {
		updates := fs.destroy()
		delete(s.folderStates, folder)
		return updates
	}
	return nil
}
