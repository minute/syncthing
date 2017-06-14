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
