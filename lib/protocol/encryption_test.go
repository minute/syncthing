package protocol

import "testing"
import "fmt"

func TestPSK(t *testing.T) {
	var emptyPSK PSK
	k1 := NewPSK()
	k2 := NewPSK()
	if *k1 == emptyPSK || *k1 == *k2 {
		t.Fatal("keys should not be identical")
	}

	k1str, err := k1.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(k1str))

	var k3 PSK
	if err := k3.UnmarshalText(k1str); err != nil {
		t.Fatal(err)
	}

	if k3 != *k1 {
		t.Error("unmarshal didn't")
	}
}

func TestEncryptedFileInfo(t *testing.T) {
	fi := FileInfo{
		Name:        "a test file",
		Type:        FileInfoTypeFile,
		Size:        1234567,
		Permissions: 0666,
		ModifiedS:   123456789,
		ModifiedNs:  234567890,
		ModifiedBy:  345678901,
		Version:     Vector{Counters: []Counter{{1, 2}, {3, 4}}},
		Sequence:    456789012,
		Blocks: []BlockInfo{
			BlockInfo{Size: 1234, Hash: []byte{1, 2, 3, 4}, WeakHash: 5678},
			BlockInfo{Size: 2345, Hash: []byte{2, 3, 4, 5}, WeakHash: 6789},
		},
	}
	psk := NewPSK()
	efi := EncryptFileInfo(psk, fi)
	t.Log(efi)
	dfi, err := DecryptFileInfo(psk, efi)
	if err != nil {
		t.Fatal(err)
	}
	if fi.String() != dfi.String() {
		t.Error("results differ")
	}
}
