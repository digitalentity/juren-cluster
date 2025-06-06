package node

import (
	"juren/oid"
	"reflect"
	"time"
)

type Capability int

const (
	CapabilityPersistentStorage Capability = iota // Node has persistent storage backing it
	CapabilityAcceptingBlocks                     // Node accepts new blocks
)

type StorageUsage struct {
	TotalBytes     uint64 `cbor:"1,keyasint,omitempty"` // Total bytes in the Block Storage
	UsedBytes      uint64 `cbor:"2,keyasint,omitempty"` // Total size of all blocks in storage
	AvailableBytes uint64 `cbor:"3,keyasint,omitempty"` // Available bytes for new blocks
}

type Metadata struct {
	NodeID         oid.Oid             `cbor:"1,keyasint,omitempty"` // Node identifier
	Addresses      []string            `cbor:"2,keyasint,omitempty"` // Node network addresses and ports
	SequenceNumber uint64              `cbor:"3,keyasint,omitempty"` // Local Block Index sequence number we last synched with this node
	LastSeen       time.Time           `cbor:"4,keyasint,omitempty"` // Last time we heard from this node
	Capabilities   map[Capability]bool `cbor:"5,keyasint,omitempty"` // Node capabilities`
	StorageUsage   StorageUsage        `cbor:"6,keyasint,omitempty"` // Node storage usage`
}

// NodeIndex defines the interface for managing metadata about nodes.
type NodeIndex interface {
	// Get retrieves the metadata for a node, given the node's OID.
	// It returns a Metadata object if found, or an error if the OID does not exist or an issue occurs.
	Get(*oid.Oid) (*Metadata, error)

	// Put stores or updates a node's metadata in the index.
	// It returns the stored Metadata and an error if the operation fails.
	Put(*Metadata) (*Metadata, error)

	// Enumerate returns a list of OIDs for all nodes currently in the index.
	// It returns an error if an issue occurs during enumeration.
	Enumerate() ([]*oid.Oid, error)
}

func IsMetadataEqual(a *Metadata, b *Metadata) bool {
	return reflect.DeepEqual(a, b)
}

func BuildCapabilityMap(capabilities []Capability) map[Capability]bool {
	capabilityMap := make(map[Capability]bool, len(capabilities))
	for _, capability := range capabilities {
		capabilityMap[capability] = true
	}
	return capabilityMap
}
