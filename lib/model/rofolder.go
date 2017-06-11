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
)

func init() {
	folderFactories[config.FolderTypeSendOnly] = newSendOnlyFolder
}

type sendOnlyFolder struct {
	*folderScanner
}

func newSendOnlyFolder(deps folderDependencies) service {
	return &sendOnlyFolder{
		folderScanner: newFolderScanner(context.Background(), deps),
	}
}

func (f *sendOnlyFolder) BringToFront(string) {
	panic("bug: BringToFront on send only folder")
}

func (f *sendOnlyFolder) IndexUpdated() {
}

func (f *sendOnlyFolder) Jobs() ([]string, []string) {
	return nil, nil
}

func (f *sendOnlyFolder) getState() (folderState, time.Time, error) {
	return f.folderScanner.folderDependencies.stateTracker.getState() // XXX: trololol
}
