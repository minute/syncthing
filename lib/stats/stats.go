// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

//go:generate go run ../../script/protofmt.go stats.proto
//go:generate protoc --proto_path=../../../../../:../../../../gogo/protobuf/protobuf:. --gogofast_out=. stats.proto

package stats

import "time"

func (d DeviceAddr) ValidUntil() time.Time {
	return time.Unix(d.validUntil, 0)
}

func (d DeviceAddr) Address() string {
	return d.address
}
