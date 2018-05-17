package model

import (
	"fmt"

	"github.com/syncthing/syncthing/lib/config"
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

	f := receiveOnlyFolder{sr}
	f.folder.filterer = receiveOnlyFilter{}

	return f
}

func (f *receiveOnlyFolder) String() string {
	return fmt.Sprintf("receiveOnlyFolder/%s@%p", f.folderID, f)
}

type receiveOnlyFilter struct{}

func (receiveOnlyFilter) filter(fs []protocol.FileInfo) []protocol.FileInfo {
	for i := range fs {
		fs[i].LocalFlags = protocol.FlagLocalReceiveOnly
	}
	return fs
}
