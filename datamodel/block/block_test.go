package block

import (
	"crypto/rand"
	"juren/oid"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

func createTestBlock(size uint64) *Block {
	b := &Block{
		Oid:    *oid.FromStringMustParse("AGVABZGP4JKRP2AK4YSGHKS7WH2C7P44XDNWJ4EDYP4B2GQ5UFQQQAXT"),
		Length: size,
		Data:   make([]byte, size),
	}
	rand.Read(b.Data)
	return b
}

func TestBlockMarshallUnmarshall(t *testing.T) {
	b := createTestBlock(1024 * 1024)

	// Marshall block
	enc, err := cbor.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Data size: %d, Encoded size: %d\n", len(b.Data), len(enc))

	// Unmarshall block
	var b2 Block
	err = cbor.Unmarshal(enc, &b2)
	if err != nil {
		t.Fatal(err)
	}

	// Compare original block and new block
	if b.Oid != b2.Oid {
		t.Fatalf("Oids do not match: %v != %v", b.Oid, b2.Oid)
	}
	if b.Length != b2.Length {
		t.Fatalf("Lengths do not match: %v != %v", b.Length, b2.Length)
	}
	if len(b.Data) != len(b2.Data) {
		t.Fatalf("Data lengths do not match: %v != %v", len(b.Data), len(b2.Data))
	}
	for i := range b.Data {
		if b.Data[i] != b2.Data[i] {
			t.Fatalf("Data does not match at index %v: %v != %v", i, b.Data[i], b2.Data[i])
		}
	}
}
