package model

import (
	"fmt"

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

func (f *receiveOnlyFolder) String() string {
	return fmt.Sprintf("receiveOnlyFolder/%s@%p", f.folderID, f)
}

func (f *receiveOnlyFolder) Revert(fs *db.FileSet, updateFn func([]protocol.FileInfo)) {
	f.setState(FolderScanning)
	defer f.setState(FolderIdle)

	batch := make([]protocol.FileInfo, 0, maxBatchSizeFiles)
	batchSizeBytes := 0
	fs.WithHave(protocol.LocalDeviceID, func(intf db.FileIntf) bool {
		fi := intf.(protocol.FileInfo)

		// Incrementing our version and resetting the others to zero ensures
		// we are in conflict with any remote change. The next pull will
		// move our conflicting changes out of the way and grab the latest
		// from the cluster.
		fi.Version = fi.Version.Update(f.shortID).DropOthers(f.shortID)
		batch = append(batch, fi)
		batchSizeBytes += fi.ProtoSize()

		if len(batch) >= maxBatchSizeFiles || batchSizeBytes >= maxBatchSizeBytes {
			updateFn(batch)
			batch = batch[:0]
		}
		return true
	})
	if len(batch) > 0 {
		updateFn(batch)
	}
}
