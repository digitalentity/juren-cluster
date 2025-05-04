package object

import (
	"juren/oid"
	"time"
)

// Chunk is a helper descriptor of a block with offset. This is intended to be used as part of the ChunkSet.
type Chunk struct {
	_      struct{} `cbor:",toarray"`
	Offset uint64   // Offset of a chunk within the chunkset
	Length uint64   // Length of a chunk
	Oid    oid.Oid  // OID of a block that stores the data (OidTypeRawData)
}

// Chunkset is a serializable type that is stored in a block.
// It represents a set of connected chunks that make up object data (or a file).
// OID of a chinkset is SHA256-based hash of all object data.
type ChunkSet struct {
	Oid    oid.Oid `cbor:"1,keyasint,omitempty"` // OID of the ChunkSet (OidTypeChunkSet)
	Size   uint64  `cbor:"2,keyasint,omitempty"` // Size of the ChunkSet
	Chunks []Chunk `cbor:"3,keyasint,omitempty"` // Chunks of the ChunkSet
}

// Object represents a named ChunkSet. This is a helper object that is intended to be used only with the ObjectSet.
type Object struct {
	_     struct{}  `cbor:",toarray"`
	Name  string    // Name of the Object
	CTime time.Time // Creation time of the Object
	Oid   oid.Oid   // OID of the ChunkSet
}

// Objectset is a named collection of objects.
// OID of an Objectset is based on SHA256 of the ObjectSet Name.
type ObjectSet struct {
	Oid      oid.Oid   `cbor:"1,keyasint,omitempty"` // OID of the ObjectSet (OidTypeObjectSet)`
	Name     string    `cbor:"2,keyasint,omitempty"` // Name of the ObjectSet
	Sequence uint64    `cbor:"3,keyasint"`           // Sequence number of the ObjectSet
	CTime    time.Time `cbor:"4,keyasint"`           // Creation time of the ObjectSet
	Objects  []Object  `cbor:"5,keyasint,omitempty"` // Objects of the ObjectSet
}
