package block

import (
	"juren/oid"
)

// OID of a BLOCK is identified by the blocks content
type Block struct {
	Oid    oid.Oid `cbor:"1,keyasint"`
	Length uint64  `cbor:"2,keyasint,omitempty"`
	Data   []byte  `cbor:"3,keyasint,omitempty"`
}

type BlockStore interface {
	// Block Storage operations
	Get(oid.Oid) (*Block, error)
	Put(*Block) (oid.Oid, error)
	Delete(oid.Oid) error
}
