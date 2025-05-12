package protocol

import (
	"juren/datastore/block"
	"juren/oid"
)

var ()

type PeerAnnouncementMessage struct {
	NodeID         oid.Oid `cbor:"1,keyasint,omitempty"` // Node identifier
	Address        string  `cbor:"2,keyasint,omitempty"` // Node network address and port
	SequenceNumber uint64  `cbor:"3,keyasint,omitempty"` // Local Block Index sequence number
}

type PeerSyncRequest struct {
	NodeID         oid.Oid `cbor:"1,keyasint,omitempty"` // Requesting Node
	SequenceNumber uint64  `cbor:"2,keyasint,omitempty"` // Return operations starting at this Seqence Number
	BatchSize      uint64  `cbor:"3,keyasint,omitempty"` // Batch size
}

type PeerSyncResponse struct {
	NodeID  oid.Oid                  `cbor:"1,keyasint,omitempty"` // Responding Node
	Entries []*block.MetadataWithSeq `cbor:"2,keyasint,omitempty"`
}

type BlockGetRequest struct {
	Oid oid.Oid `cbor:"1,keyasint,omitempty"` // OID of the Block`
}

type BlockGetResponse struct {
	Block *block.Block `cbor:"1,keyasint,omitempty"` // Requested Block
}

type BlockPutRequest struct {
	Block *block.Block `cbor:"1,keyasint,omitempty"` // Block to be stored
}

type BlockPutResponse struct {
	Oid oid.Oid `cbor:"1,keyasint,omitempty"` // OID of the stored Block
}
