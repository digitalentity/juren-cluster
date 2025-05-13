package block

import (
	"juren/oid"
	"time"
)

// OID of a BLOCK is identified by the blocks content
type Block struct {
	_      struct{} `cbor:",toarray"` // This is compact, but doesn't retain the field structure
	Oid    oid.Oid
	Length uint64
	Data   []byte
}

type Metadata struct {
	Oid        oid.Oid   `cbor:"1,keyasint"`
	Length     uint64    `cbor:"2,keyasint,omitempty"`
	UpdateTime time.Time `cbor:"3,keyasint,omitempty"`
	IsDeleted  bool      `cbor:"4,keyasint,omitempty"`
}

type MetadataWithSeq struct {
	Sequence uint64    `cbor:"1,keyasint"`
	Metadata *Metadata `cbor:"2,keyasint"`
}

type BlockStore interface {
	// Block Storage operations
	Get(*oid.Oid) (*Block, error)
	Has(*oid.Oid) (bool, error)
	Put(*Block) (*oid.Oid, error)
	Delete(*oid.Oid) error
	Close() error
}

type BlockIndex interface {
	// Fetches a Metadata entry with a given OID
	GetByOid(*oid.Oid) (*MetadataWithSeq, error)

	// Fetches a Metadata entry with a given _local_ Sequence Number
	GetBySeq(uint64) (*MetadataWithSeq, error)

	// Stores a Metadata entry in the Index Storage. Updates the Sequence Number if entry is different from an existing one.
	Put(*oid.Oid, *MetadataWithSeq) (*MetadataWithSeq, error)

	// Enumerates the blocks with the Seqence Number in the given range (inclusive)
	EnumerateBySeq(uint64, uint64) ([]*MetadataWithSeq, error)
}
