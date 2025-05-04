package block

import (
	"juren/oid"
	"time"
)

// OID of a BLOCK is identified by the blocks content
type Block struct {
	_      struct{} `cbor:",toarray"`
	Oid    oid.Oid
	Length uint64
	Data   []byte
}

type BlockMetadata struct {
	_      struct{} `cbor:",toarray"`
	Length uint64
	CTime  time.Time
}

type BlockStore interface {
	// Block Storage operations
	Get(oid.Oid) (*Block, error)
	Put(*Block) (oid.Oid, error)
	Delete(oid.Oid) error
}

type BlockIndex interface {
	Get(oid.Oid) (*BlockMetadata, error)
	Put(oid.Oid, *BlockMetadata) error
}
