// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package version

import "github.com/syncthing/syncthing/lib/logger"

// Empty description makes this not show up in the "syncthing -help" output
var l = logger.DefaultLogger.NewFacility("version", "")
