package protocol

import (
	"juren/datamodel/block"
	"juren/oid"
)

// PeerAnnouncementMessage is broadcast by a Node to announce its presence, RPC address and Local Block Index sequence.
type PeerAnnouncementMessage struct {
	NodeID         oid.Oid  `cbor:"1,keyasint,omitempty"` // Node identifier
	Addresses      []string `cbor:"2,keyasint,omitempty"` // Node network address and port
	SequenceNumber uint64   `cbor:"3,keyasint,omitempty"` // Local Block Index sequence number
}

// BlockAnnouncementMessage is broadcast by a Node to announce presence of the block in local storage.
type BlockAnnouncementMessage struct {
	NodeID oid.Oid        `cbor:"1,keyasint,omitempty"` // Node identifier
	Block  block.Metadata `cbor:"2,keyasint,omitempty"` // Block metadata
	Has    bool           `cbor:"3,keyasint,omitempty"` // Whether this node has this block in storage or not
}

// BlockDiscoveryMessage is broadcast to find peers that have a specific block.
type BlockDiscoveryMessage struct {
	Oid oid.Oid `cbor:"1,keyasint,omitempty"` // OID of the Block
}

// PeerSyncRequest is sent by a peer to request metadata updates from another peer.
type PeerSyncRequest struct {
	NodeID         oid.Oid `cbor:"1,keyasint,omitempty"` // Requesting Node
	SequenceNumber uint64  `cbor:"2,keyasint,omitempty"` // Return operations starting at this Seqence Number
	BatchSize      uint64  `cbor:"3,keyasint,omitempty"` // Batch size
}

// PeerSyncResponse is sent by a peer in response to a PeerSyncRequest, containing a list of metadata entries.
type PeerSyncResponse struct {
	NodeID  oid.Oid                     `cbor:"1,keyasint,omitempty"` // Responding Node
	Entries []*block.ExtendedMedatadata `cbor:"2,keyasint,omitempty"`
}

// BlockGetRequest is sent to request a specific block from a peer.
type BlockGetRequest struct {
	Oid oid.Oid `cbor:"1,keyasint,omitempty"` // OID of the Block`
}

// BlockGetResponse is sent in response to a BlockGetRequest, containing the requested block.
type BlockGetResponse struct {
	Block *block.Block `cbor:"1,keyasint,omitempty"` // Requested Block
}

// BlockPutRequest is sent to store a block on a peer.
type BlockPutRequest struct {
	Block *block.Block `cbor:"1,keyasint,omitempty"` // Block to be stored
}

// BlockPutResponse is sent in response to a BlockPutRequest, confirming the OID of the stored block.
type BlockPutResponse struct {
	Oid oid.Oid `cbor:"1,keyasint,omitempty"` // OID of the stored Block
}
