// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

// Package archiver implements common interfaces for file versioning and a
// simple default versioning scheme.
package archiver

type Archiver interface {
	Archive(filePath string) error
}

var Factories = map[string]func(folderID string, folderDir string, params map[string]string) Archiver{}

const (
	TimeFormat = "20060102-150405"
	TimeGlob   = "[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]-[0-9][0-9][0-9][0-9][0-9][0-9]" // glob pattern matching TimeFormat
)
