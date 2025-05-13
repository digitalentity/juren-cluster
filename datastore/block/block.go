package block

import (
	"juren/oid"
	"reflect"
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

// BlockStore defines the interface for storing and retrieving blocks of data.
type BlockStore interface {
	// Get retrieves a block from the store by its OID.
	// It returns the Block if found, or an error if the block does not exist or an issue occurs.
	Get(*oid.Oid) (*Block, error)

	// Has checks if a block with the given OID exists in the store.
	// It returns true if the block exists, false otherwise, and an error if an issue occurs during the check.
	Has(*oid.Oid) (bool, error)

	// Put stores a block in the store.
	// It returns the OID of the stored block, or an error if an issue occurs during storage.
	Put(*Block) (*oid.Oid, error)

	// Delete removes a block from the store by its OID.
	// It returns an error if the block does not exist or an issue occurs during deletion.
	Delete(*oid.Oid) error

	// Enumerate returns a list of OIDs for all blocks currently in the store.
	// It returns an error if an issue occurs during enumeration.
	Enumerate() ([]*oid.Oid, error)

	// Close releases any resources held by the BlockStore.
	Close() error
}

// BlockIndex defines the interface for managing metadata about blocks.
// It allows for tracking blocks by their OID and a local sequence number,
// which is useful for synchronization and understanding the order of operations.
type BlockIndex interface {
	// GetByOid retrieves the metadata (including its sequence number) for a block,
	// given the block's OID.
	// It returns a MetadataWithSeq object if found, or an error if the OID does not exist or an issue occurs.
	GetByOid(*oid.Oid) (*MetadataWithSeq, error)

	// GetBySeq retrieves the metadata (including its OID) for a block,
	// given its local sequence number.
	// It returns a MetadataWithSeq object if found, or an error if the sequence number does not exist or an issue occurs.
	GetBySeq(uint64) (*MetadataWithSeq, error)

	// Put stores or updates a block's metadata in the index.
	// If metadata for the given OID already exists and is identical, the operation might be a no-op, returning the existing entry.
	// Otherwise, it assigns a new, unique sequence number (typically by incrementing the current highest sequence number)
	// and stores the metadata.
	// It returns the stored or existing MetadataWithSeq (with its assigned sequence number) and an error if the operation fails.
	Put(*MetadataWithSeq) (*MetadataWithSeq, error)

	// Has checks if metadata for a block with the given OID exists in the index.
	// It returns true if the metadata exists, false otherwise, and an error if an issue occurs during the check.
	Has(*oid.Oid) (bool, error)

	// EnumerateBySeq retrieves a list of metadata entries whose sequence numbers fall within the specified range (inclusive).
	// This is useful for fetching a batch of changes or synchronizing data since a certain point.
	// It returns a slice of MetadataWithSeq objects and an error if the enumeration fails or the range is invalid.
	EnumerateBySeq(uint64, uint64) ([]*MetadataWithSeq, error)

	// GetSeq returns the current highest sequence number known to the BlockIndex.
	// This can be used to determine the latest state of the index.
	GetSeq() uint64

	// Close releases any resources held by the BlockIndex implementation (e.g., database connections).
	Close() error
}

func IsMetadataEqual(a *Metadata, b *Metadata) bool {
	return reflect.DeepEqual(a, b)
}
