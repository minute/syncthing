package model

import (
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/db"
	"github.com/syncthing/syncthing/lib/fs"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/versioner"
)

func init() {
	folderFactories[config.FolderTypeReceiveOnly] = newReceiveOnlyFolder
}

type receiveOnlyFolder struct {
	*sendReceiveFolder
}

func newReceiveOnlyFolder(model *Model, cfg config.FolderConfiguration, ver versioner.Versioner, fs fs.Filesystem) service {
	sr := newSendReceiveFolder(model, cfg, ver, fs).(*sendReceiveFolder)
	sr.localFlags = protocol.FlagLocalReceiveOnly
	return &receiveOnlyFolder{sr}
}

func (f *receiveOnlyFolder) Revert(fs *db.FileSet, updateFn func([]protocol.FileInfo)) {
	f.setState(FolderScanning)
	defer f.setState(FolderIdle)

	batch := make([]protocol.FileInfo, 0, maxBatchSizeFiles)
	batchSizeBytes := 0
	fs.WithHave(protocol.LocalDeviceID, func(intf db.FileIntf) bool {
		fi := intf.(protocol.FileInfo)
		if !fi.IsReceiveOnlyChanged() {
			// We're only interested in files that have changed locally in
			// receive only mode.
			return true
		}

		// Revert means to throw away our local changes. We reset the
		// version to the empty vector, which is strictly older than any
		// other existing version. It is not in conflict with anything,
		// either, so we will not create a conflict copy of our local
		// changes.
		fi.Version = protocol.Vector{}
		fi.LocalFlags &^= protocol.FlagLocalReceiveOnly

		batch = append(batch, fi)
		batchSizeBytes += fi.ProtoSize()

		if len(batch) >= maxBatchSizeFiles || batchSizeBytes >= maxBatchSizeBytes {
			updateFn(batch)
			batch = batch[:0]
			batchSizeBytes = 0
		}
		return true
	})
	if len(batch) > 0 {
		updateFn(batch)
	}
}
