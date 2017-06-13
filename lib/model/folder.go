// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"context"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/db"
	"github.com/syncthing/syncthing/lib/fs"
	"github.com/syncthing/syncthing/lib/ignore"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/scanner"
)

type folderFactory func(deps folderDependencies) service

var (
	folderFactories = make(map[config.FolderType]folderFactory, 0)
)

type dbPrefixIterator interface {
	// XXX: This interface should not depend on db.Iterator (meaning db.FileIntf)
	// or we have the coupling anyway and might as well just take a *db.DB
	iterate(prefix string, iterator db.Iterator)
}

type dbUpdater interface {
	update(files []protocol.FileInfo)
}

type folderDependencies struct {
	deviceID         protocol.DeviceID
	folderCfg        config.FolderConfiguration
	currentFiler     scanner.CurrentFiler
	filesystem       fs.Filesystem
	ignores          *ignore.Matcher
	stateTracker     *stateTracker
	dbUpdater        dbUpdater
	dbPrefixIterator dbPrefixIterator
	ignoresChanged   chan<- time.Time
}

type folder struct {
	stateTracker

	scan                folderScanner
	model               *Model
	ctx                 context.Context
	cancel              context.CancelFunc
	initialScanFinished chan struct{}
}

func (f *folder) IndexUpdated() {
}

func (f *folder) DelayScan(next time.Duration) {
	f.scan.Delay(next)
}

func (f *folder) Scan(subdirs []string) error {
	<-f.initialScanFinished
	return f.scan.Scan(subdirs)
}

func (f *folder) Stop() {
	f.cancel()
}

func (f *folder) Jobs() ([]string, []string) {
	return nil, nil
}

func (f *folder) BringToFront(string) {}
