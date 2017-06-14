package protocol

import (
	"encoding/base32"
	"errors"
	"fmt"

	"bytes"

	"github.com/syncthing/syncthing/lib/rand"
	"github.com/syncthing/syncthing/lib/sha256"
	"golang.org/x/crypto/nacl/secretbox"
)

type PSK [32]byte

func NewPSK() *PSK {
	var psk PSK
	if n, err := rand.Reader.Read(psk[:]); err != nil || n != len(psk) {
		panic("random failure")
	}
	return &psk
}

func (psk *PSK) MarshalText() ([]byte, error) {
	enc := make([]byte, base32.HexEncoding.EncodedLen(len(*psk)))
	base32.StdEncoding.Encode(enc, (*psk)[:])
	enc = bytes.TrimSuffix(enc, []byte("===="))
	return enc, nil
}

func (psk *PSK) UnmarshalText(bs []byte) error {
	bs = append(bs, "===="...)
	if l := base32.StdEncoding.DecodedLen(len(bs)); l != 35 {
		// The length includes padding due to base32 things being a
		// multiple of 5 bytes.
		return fmt.Errorf("incorrect decoded len %d", l)
	}
	var tmp [35]byte
	n, err := base32.StdEncoding.Decode(tmp[:], bs)
	if err != nil {
		return err
	}
	if n != len(*psk) {
		// Despite needing a 35 byte buffer, it should return only
		// 32 bytes of data.
		return fmt.Errorf("short decode: %d != %d", n, len(*psk))
	}
	copy((*psk)[:], tmp[:])
	return nil
}

func newNonce() *[24]byte {
	var nonce [24]byte
	if n, err := rand.Reader.Read(nonce[:]); err != nil || n != len(nonce) {
		panic("random failure")
	}
	return &nonce
}

func numBlocks(size int64) int32 {
	switch size % BlockSize {
	case 0:
		return int32(size / BlockSize)
	default:
		return int32(size/BlockSize + 1)
	}
}

func fileIdentifier(psk *PSK, name string) string {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write((*psk)[:])
	return base32.HexEncoding.EncodeToString(h.Sum(nil))
}

func EncryptFileInfo(psk *PSK, fi FileInfo) EncryptedFileInfo {
	nonce := newNonce()
	bs, _ := fi.Marshal() // cannot fail?
	enc := secretbox.Seal(nil, bs, nonce, (*[32]byte)(psk))
	return EncryptedFileInfo{
		Identifier:        fileIdentifier(psk, fi.Name),
		EncryptedFileinfo: enc,
		Nonce:             (*nonce)[:],
		NumBlocks:         numBlocks(fi.Size),
		Version:           fi.Version,
	}
}

func DecryptFileInfo(psk *PSK, efi EncryptedFileInfo) (FileInfo, error) {
	var nonce [24]byte
	copy(nonce[:], efi.Nonce)
	bs, ok := secretbox.Open(nil, efi.EncryptedFileinfo, &nonce, (*[32]byte)(psk))
	if !ok {
		return FileInfo{}, errors.New("crypto fail")
	}
	var fi FileInfo
	err := fi.Unmarshal(bs)
	return fi, err
}
