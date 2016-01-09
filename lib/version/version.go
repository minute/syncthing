// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package version

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	Version     = "unknown-dev"
	Codename    = "Beryllium Bedbug"
	BuildStamp  = "0"
	BuildDate   time.Time
	BuildHost   = "unknown"
	BuildUser   = "unknown"
	IsRelease   bool
	IsBeta      bool
	LongVersion string
)

func init() {
	if Version != "unknown-dev" {
		// If not a generic dev build, version string should come from git describe
		exp := regexp.MustCompile(`^v\d+\.\d+\.\d+(-[a-z0-9]+)*(\+\d+-g[0-9a-f]+)?(-dirty)?$`)
		if !exp.MatchString(Version) {
			l.Fatalf("Invalid version string %q;\n\tdoes not match regexp %v", Version, exp)
		}
	}

	// Check for a clean release build. A release is something like "v0.1.2",
	// with an optional suffix of letters and dot separated numbers like
	// "-beta3.47". If there's more stuff, like a plus sign and a commit hash
	// and so on, then it's not a release. If there's a dash anywhere in
	// there, it's some kind of beta or prerelease version.

	exp := regexp.MustCompile(`^v\d+\.\d+\.\d+(-[a-z]+[\d\.]+)?$`)
	IsRelease = exp.MatchString(Version)
	IsBeta = strings.Contains(Version, "-")

	stamp, _ := strconv.Atoi(BuildStamp)
	BuildDate = time.Unix(int64(stamp), 0)

	date := BuildDate.UTC().Format("2006-01-02 15:04:05 MST")
	LongVersion = fmt.Sprintf(`syncthing %s "%s" (%s %s-%s) %s@%s %s`, Version, Codename, runtime.Version(), runtime.GOOS, runtime.GOARCH, BuildUser, BuildHost, date)
}
