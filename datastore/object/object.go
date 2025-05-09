package object

import (
	"juren/oid"
	"time"
)

// Object is a serializable type that is stored in a block.
// It represents a set of connected chunks that make up object data (or a file).
// OID of a chinkset is SHA256-based hash of all object data.
type Object struct {
	Oid    oid.Oid `cbor:"1,keyasint,omitempty"` // OID of the ChunkSet (OidTypeChunkSet)
	Size   uint64  `cbor:"2,keyasint,omitempty"` // Size of the ChunkSet
	Chunks []struct {
		Offset uint64  `cbor:"3,keyasint,omitempty"` // Offset of a chunk within the chunkset
		Length uint64  `cbor:"4,keyasint,omitempty"` // Length of a chunk
		Oid    oid.Oid `cbor:"5,keyasint,omitempty"` // OID of a block that stores the data (OidTypeRawData)
	}
}

// Objectset is a named collection of objects.
// OID of an Objectset is based on SHA256 of the ObjectSet Name.
type ObjectSet struct {
	Oid        oid.Oid   `cbor:"1,keyasint,omitempty"` // OID of the ObjectSet (OidTypeObjectSet)`
	Name       string    `cbor:"2,keyasint,omitempty"` // Name of the ObjectSet
	UpdateTime time.Time `cbor:"3,keyasint"`           // Creation time of the ObjectSet
	Items      []struct {
		Path string  `cbor:"1,keyasint,omitempty"`
		Oid  oid.Oid `cbor:"2,keyasint,omitempty"`
	}
}
