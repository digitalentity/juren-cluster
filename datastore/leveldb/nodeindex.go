package leveldb

import (
	"juren/datamodel/node"
	"juren/oid"

	"github.com/fxamacker/cbor/v2"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/util"

	log "github.com/sirupsen/logrus"
)

const (
	keyPrefixNode = "NOD" // Node metadata indexed by OID. Followed by textual OID representation
)

var _ node.NodeIndex = (*NodeIndex)(nil)

type NodeIndex struct {
	*LebelDB
}

func NewNodeIndex(path string) (*NodeIndex, error) {
	// Init the underlying LevelDB object
	ldb, err := New(path)
	if err != nil {
		return nil, err
	}

	return &NodeIndex{LebelDB: ldb}, nil
}

func (l *NodeIndex) Get(oid *oid.Oid) (*node.Metadata, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Fetch the object
	v, err := l.getByKey(keyFromOid(oid), &node.Metadata{})
	if err != nil {
		return nil, err
	}

	// Cast to the expected type
	md := v.(*node.Metadata)

	// Compare the OID just in case
	if md.NodeID != *oid {
		log.Errorf("Get: NodeID mismatch: %s != %s", oid.String(), md.NodeID.String())
		return nil, ErrCorrupted
	}

	return md, nil
}

func (l *NodeIndex) Put(metadata *node.Metadata) (*node.Metadata, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Fetch the existing metadata
	nodeID := &metadata.NodeID

	v, err := l.getByKey(keyFromOid(nodeID), &node.Metadata{})
	if err != nil && err != errors.ErrNotFound {
		// ErrNotFound is acceptable here, everything else is not.
		return nil, err
	}

	// Cast to the correct type
	existing := v.(*node.Metadata)

	// Compare the metadata
	if existing != nil && *existing == *metadata {
		log.Debugf("Put: Metadata for NodeID %s is unchanged, skipping update", nodeID.String())
		return existing, nil
	}

	// Marshall Metadata to CBOR
	raw, err := cbor.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	// Write the data
	err = l.db.Put(keyFromOid(nodeID), raw, nil)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

func (l *NodeIndex) Enumerate() ([]*oid.Oid, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var results []*oid.Oid

	// Create an iterator for the range of sequence numbers
	iter := l.db.NewIterator(util.BytesPrefix([]byte(keyPrefixNode)), nil)
	defer iter.Release()

	// Iterate over the range and collect metadata entries
	for iter.Next() {
		raw := iter.Value()

		metadata := &node.Metadata{}
		err := cbor.Unmarshal(raw, metadata)
		if err != nil {
			return nil, err
		}

		results = append(results, &metadata.NodeID)
	}

	return results, nil
}
