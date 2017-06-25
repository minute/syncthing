package protocol

import "sync"
import "golang.org/x/crypto/nacl/secretbox"

type KeyFunc func(folderID string) (*PSK, bool)

type encryptedConnection struct {
	Connection
	keyFn       KeyFunc
	keyCache    map[string]*PSK
	keyCacheMut sync.Mutex
}

func NewEncryptedConnection(next Connection, keyFn KeyFunc) Connection {
	return &encryptedConnection{
		Connection: next,
		keyFn:      keyFn,
		keyCache:   make(map[string]*PSK),
	}
}

func (c *encryptedConnection) Index(folder string, files []FileInfo) error {

}

func (c *encryptedConnection) IndexUpdate(folder string, files []FileInfo) error {

}

func (c *encryptedConnection) Request(folder string, name string, offset int64, size int, hash []byte, fromTemporary bool) ([]byte, error) {
	psk, ok := c.keyFor(folder)
	if !ok {
		return c.Connection.Request(folder, name, offset, size, hash, fromTemporary)
	}

	folder = fileIdentifier(psk, folder)    // XXX: cache these
	identifier := fileIdentifier(psk, name) // but not these, probably, other than for a short time
	block := int32(size / BlockSize)

	data, nonce, err := c.Connection.EncryptedRequest(folder, identifier, block)
	if err != nil {
		return nil, err
	}

	data, ok = secretbox.Open(nil, data, toNonce(nonce), psk.raw())
	if !ok {
		return nil, errCannotDecrypt
	}
	return data, nil
}

func (c *encryptedConnection) ClusterConfig(config ClusterConfig) {
	// Rewrite folder IDs for encrypted folders.
	for i := range config.Folders {
		fld := &config.Folders[i]
		if psk, ok := c.keyFor(fld.ID); ok {
			fld.ID = fileIdentifier(psk, fld.ID)
			fld.Label = "(Encrypted folder)"
			fld.Encrypted = true
			// XXX: maybe filter list of devices?
		}
	}
	c.Connection.ClusterConfig(config)
}

func (c *encryptedConnection) DownloadProgress(folder string, updates []FileDownloadProgressUpdate) {
	if _, ok := c.keyFor(folder); ok {
		// There is an encryption key set. Do not send download progress.
		return
	}
	c.Connection.DownloadProgress(folder, updates)
}

func (c *encryptedConnection) keyFor(folder string) (*PSK, bool) {
	c.keyCacheMut.Lock()
	psk, ok := c.keyCache[folder]
	if ok {
		c.keyCacheMut.Unlock()
		return psk, psk != nil
	}
	psk, ok = c.keyFn(folder)
	c.keyCache[folder] = psk
	c.keyCacheMut.Unlock()
	return psk, ok
}
