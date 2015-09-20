// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package model

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/syncthing/protocol"
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/sync"
)

type folderConnectionProvider interface {
	FolderConnections() map[string][]Connection
}

type ProgressEmitter struct {
	registry           map[string]*sharedPullerState
	interval           time.Duration
	lastUpdate         time.Time
	sentDownloadStates map[protocol.DeviceID]sentDownloadState // States representing what we've sent to the other peer via DownloadProgress messages.
	sendTempIndexes    bool
	mut                sync.Mutex

	timer              *time.Timer
	connectionProvider folderConnectionProvider

	stop chan struct{}
}

// NewProgressEmitter creates a new progress emitter which emits
// DownloadProgress events every interval.
func NewProgressEmitter(cfg *config.Wrapper, connectionProvider folderConnectionProvider) *ProgressEmitter {
	t := &ProgressEmitter{
		connectionProvider: connectionProvider,
		stop:               make(chan struct{}),
		registry:           make(map[string]*sharedPullerState),
		timer:              time.NewTimer(time.Millisecond),
		sentDownloadStates: make(map[protocol.DeviceID]sentDownloadState),
		mut:                sync.NewMutex(),
	}

	t.CommitConfiguration(config.Configuration{}, cfg.Raw())
	cfg.Subscribe(t)

	return t
}

// Serve starts the progress emitter which starts emitting DownloadProgress
// events as the progress happens.
func (t *ProgressEmitter) Serve() {
	for {
		select {
		case <-t.stop:
			if debug {
				l.Debugln("progress emitter: stopping")
			}
			return
		case <-t.timer.C:
			t.mut.Lock()
			if debug {
				l.Debugln("progress emitter: timer - looking after", len(t.registry))
			}

			newLastUpdated := t.lastUpdate
			for _, puller := range t.registry {
				updated := puller.Updated()
				if updated.After(newLastUpdated) {
					newLastUpdated = updated
				}
			}

			if !newLastUpdated.Equal(t.lastUpdate) {
				t.lastUpdate = newLastUpdated
				t.sendDownloadProgressEvent()
				if t.sendTempIndexes {
					t.sendDownloadProgressMessages()
				}
			} else if debug {
				l.Debugln("progress emitter: nothing new")
			}

			if len(t.registry) != 0 {
				t.timer.Reset(t.interval)
			}
			t.mut.Unlock()
		}
	}
}

func (t *ProgressEmitter) sendDownloadProgressEvent() {
	// registry lock already held
	output := make(map[string]map[string]*pullerProgress)
	for _, puller := range t.registry {
		if output[puller.folder] == nil {
			output[puller.folder] = make(map[string]*pullerProgress)
		}
		output[puller.folder][puller.file.Name] = puller.Progress()
	}
	events.Default.Log(events.DownloadProgress, output)
	if debug {
		l.Debugf("progress emitter: emitting %#v", output)
	}
}

func (t *ProgressEmitter) sendDownloadProgressMessages() {
	// registry lock already held
	sharedFolders := make(map[protocol.DeviceID][]string)
	deviceConns := make(map[protocol.DeviceID]protocol.Connection)
	for folder, conns := range t.connectionProvider.FolderConnections() {
		for _, conn := range conns {
			id := conn.ID()

			deviceConns[id] = conn
			sharedFolders[id] = append(sharedFolders[id], folder)

			state, ok := t.sentDownloadStates[id]
			if !ok {
				state = sentDownloadState{
					folderStates: make(map[string]sentFolderDownloadState),
				}
				t.sentDownloadStates[id] = state
			}

			var activePullers []*sharedPullerState
			for _, puller := range t.registry {
				if puller.folder != folder || puller.file.IsSymlink() || puller.file.IsDirectory() {
					continue
				}
				activePullers = append(activePullers, puller)
			}

			// For every new puller that hasn't yet been seen, it will send all the blocks the puller has available
			// For every existing puller, it will check for new blocks, and send update for the new blocks only
			// For every puller that we've seen before but is no longer there, we will send a forget message
			updates := state.update(folder, activePullers)
			batchSend(conn, folder, updates)
		}

		// Clean up sentDownloadStates for devices which we are no longer connected to.
		for id := range t.sentDownloadStates {
			_, ok := deviceConns[id]
			if !ok {
				// Null out outstanding entries for device
				delete(t.sentDownloadStates, id)
			}
		}
	}

	// If a folder was unshared from some device, tell it that all temp files
	// are now gone.
	for id, sharedDeviceFolders := range sharedFolders {
		state := t.sentDownloadStates[id]
	nextFolder:
		// For each of the folders that the state is aware of,
		// try to match it with a shared folder we've discovered above,
		for _, folder := range state.folders() {
			for _, existingFolder := range sharedDeviceFolders {
				if existingFolder == folder {
					goto nextFolder
				}
			}

			// If we fail to find that folder, we tell the state to forget about it
			// and return us a list of updates which would clean up the state
			// on the remote end.
			updates := state.cleanup(folder)
			batchSend(deviceConns[id], folder, updates)
		}
	}
}

// VerifyConfiguration implements the config.Committer interface
func (t *ProgressEmitter) VerifyConfiguration(from, to config.Configuration) error {
	return nil
}

// CommitConfiguration implements the config.Committer interface
func (t *ProgressEmitter) CommitConfiguration(from, to config.Configuration) bool {
	t.mut.Lock()
	defer t.mut.Unlock()

	t.sendTempIndexes = to.Options.SendTempIndexes
	t.interval = time.Duration(to.Options.ProgressUpdateIntervalS) * time.Second
	if debug {
		l.Debugln("progress emitter: updated interval", t.interval, "sendTempIndexes", t.sendTempIndexes)
	}

	return true
}

// Stop stops the emitter.
func (t *ProgressEmitter) Stop() {
	t.stop <- struct{}{}
}

// Register a puller with the emitter which will start broadcasting pullers
// progress.
func (t *ProgressEmitter) Register(s *sharedPullerState) {
	t.mut.Lock()
	defer t.mut.Unlock()
	if debug {
		l.Debugln("progress emitter: registering", s.folder, s.file.Name)
	}
	t.lastUpdate = time.Time{}
	if len(t.registry) == 0 {
		t.timer.Reset(t.interval)
	}
	t.registry[filepath.Join(s.folder, s.file.Name)] = s
}

// Deregister a puller which will stop broadcasting pullers state.
func (t *ProgressEmitter) Deregister(s *sharedPullerState) {
	t.mut.Lock()
	defer t.mut.Unlock()
	t.lastUpdate = time.Time{}
	if debug {
		l.Debugln("progress emitter: deregistering", s.folder, s.file.Name)
	}
	delete(t.registry, filepath.Join(s.folder, s.file.Name))
}

// BytesCompleted returns the number of bytes completed in the given folder.
func (t *ProgressEmitter) BytesCompleted(folder string) (bytes int64) {
	t.mut.Lock()
	defer t.mut.Unlock()

	for _, s := range t.registry {
		if s.folder == folder {
			bytes += s.Progress().BytesDone
		}
	}
	if debug {
		l.Debugf("progress emitter: bytes completed for %s: %d", folder, bytes)
	}
	return
}

func (t *ProgressEmitter) String() string {
	return fmt.Sprintf("ProgressEmitter@%p", t)
}

func batchSend(conn protocol.Connection, folder string, updates []protocol.FileDownloadProgressUpdate) {
	blocksLeft := indexTargetSize / indexPerBlockSize
	var currentBatch []protocol.FileDownloadProgressUpdate

	for _, update := range updates {
		blocks := len(update.Blocks)
		if blocks <= blocksLeft {
			currentBatch = append(currentBatch, update)
			blocksLeft -= len(update.Blocks)
		} else {
			currentBatch = append(currentBatch, protocol.FileDownloadProgressUpdate{
				Name:       update.Name,
				UpdateType: update.UpdateType,
				Blocks:     update.Blocks[:blocksLeft],
			})

			conn.DownloadProgress(folder, currentBatch, 0, nil)

			currentBatch = append(currentBatch[:0], protocol.FileDownloadProgressUpdate{
				Name:       update.Name,
				UpdateType: update.UpdateType,
				Blocks:     update.Blocks[blocksLeft:],
			})
			blocksLeft = (indexTargetSize / indexPerBlockSize) - (blocks - blocksLeft)
		}
	}

	if len(currentBatch) > 0 {
		conn.DownloadProgress(folder, currentBatch, 0, nil)
	}
}
