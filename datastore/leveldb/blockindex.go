// Package leveldb implements the block.BlockIndex and node.NodeIndex interfaces
package leveldb

import (
	"fmt"
	"juren/datamodel/block"
	"juren/oid"

	"github.com/fxamacker/cbor/v2"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	keyPrefixOID = "OID" // Block metadata indexed by OID. Followed by textual OID representation
	keyPrefixSeq = "SEQ" // Block metadata indexed by Sequence number (local). Followed by a 16-digit hexadecimal sequence number (64 bit)
)

var _ block.BlockIndex = (*BlockIndex)(nil)

type BlockIndex struct {
	LebelDB
	seq uint64
}

func NewBlockIndex(path string) (*BlockIndex, error) {
	// Init the underlyiung LevelDB object
	ldb, err := initLevelDb(path)
	if err != nil {
		return nil, err
	}

	// Scan the database to identify the sequence
	iter := ldb.NewIterator(util.BytesPrefix([]byte(keyPrefixSeq)), nil)
	defer iter.Release()

	var maxSeq uint64 = 0
	if iter.Last() {
		key := iter.Key()
		seq, err := seqFromKey(key)
		if err != nil {
			return nil, err
		}
		maxSeq = seq
	}

	return &BlockIndex{
		LebelDB: LebelDB{
			path: path,
			db:   ldb,
		},
		seq: maxSeq,
	}, nil
}

func (l *BlockIndex) GetByOid(oid *oid.Oid) (*block.ExtendedMedatadata, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Fetch the object
	raw, err := l.db.Get(keyFromOid(oid), nil)
	if err != nil {
		return nil, err
	}

	// Unmarshall CBOR
	md := &block.ExtendedMedatadata{}
	err = cbor.Unmarshal(raw, md)
	if err != nil {
		return nil, err
	}

	// Compare the OID just in case
	if md.Metadata.Oid != *oid {
		log.Errorf("GetByOid: OID mismatch: %s != %s", oid.String(), md.Metadata.Oid.String())
		return nil, ErrCorrupted
	}

	return md, nil
}

func (l *BlockIndex) GetBySeq(seq uint64) (*block.ExtendedMedatadata, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Fetch the object
	raw, err := l.db.Get(keyFromSeq(seq), nil)
	if err != nil {
		return nil, err
	}

	// Unmarshall CBOR
	md := &block.ExtendedMedatadata{}
	err = cbor.Unmarshal(raw, md)
	if err != nil {
		return nil, err
	}

	// Compare the Sequence Number just in case
	if md.SequenceNumber != seq {
		log.Errorf("GetBySeq: Sequence Number mismatch: %d != %d", seq, md.SequenceNumber)
		return nil, ErrCorrupted
	}

	return md, nil
}

func (l *BlockIndex) Put(metadata *block.ExtendedMedatadata) (*block.ExtendedMedatadata, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Fetch the existing metadata block
	oid := &metadata.Metadata.Oid

	// Generic fetch and unmarshall
	raw, err := l.db.Get(keyFromOid(oid), nil)
	if err != nil && err != errors.ErrNotFound {
		return nil, err
	}

	if err == nil {
		existing := &block.ExtendedMedatadata{}
		err = cbor.Unmarshal(raw, existing)
		if err == nil && block.IsMetadataEqual(existing.Metadata, metadata.Metadata) {
			log.Debugf("Put: Metadata for OID %s is unchanged, skipping update", oid.String())
			return existing, nil
		}
	}

	// Create the new sequence number
	newSeq := l.seq + 1

	// Copy the metadata object for writing
	md := &block.ExtendedMedatadata{
		SequenceNumber: newSeq,
		Metadata:       metadata.Metadata,
	}

	// Marshall Metadata to CBOR
	raw, err = cbor.Marshal(md)
	if err != nil {
		return nil, err
	}

	// Create a batch for atomic update
	batch := new(leveldb.Batch)

	// Insert OID -> Metadata and Seq -> Metadata objects
	batch.Put(keyFromOid(oid), raw)
	batch.Put(keyFromSeq(newSeq), raw)

	// Write the batch
	err = l.db.Write(batch, nil)
	if err != nil {
		return nil, err
	}

	// Keep the last sequence number
	l.seq = newSeq

	return md, nil
}

func (l *BlockIndex) Has(oid *oid.Oid) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if the key exists
	_, err := l.db.Get(keyFromOid(oid), nil)
	if err == nil {
		return true, nil // Key exists
	} else if err == errors.ErrNotFound {
		return false, nil // Key does not exist
	} else {
		return false, err // Some other error occurred
	}
}

func (l *BlockIndex) EnumerateBySeq(start uint64, end uint64) ([]*block.ExtendedMedatadata, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if start > end {
		return nil, fmt.Errorf("EnumerateBySeq: invalid range: start (%d) > end (%d)", start, end)
	}

	keyStart := keyFromSeq(start)
	keyEnd := keyFromSeq(end)

	var results []*block.ExtendedMedatadata

	// Create an iterator for the range of sequence numbers
	iter := l.db.NewIterator(&util.Range{Start: keyStart, Limit: keyEnd}, nil)
	defer iter.Release()

	// Iterate over the range and collect metadata entries
	for iter.Next() {
		raw := iter.Value()

		metadata := &block.ExtendedMedatadata{}
		err := cbor.Unmarshal(raw, metadata)
		if err != nil {
			return nil, err
		}

		results = append(results, metadata)
	}

	return results, nil
}

func (l *BlockIndex) GetSeq() uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.seq
}
