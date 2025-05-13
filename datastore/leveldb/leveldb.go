// Package leveldb implements the block.BlockIndex interface
package leveldb

import (
	"fmt"
	"juren/datastore/block"
	"juren/oid"
	"sync"

	"github.com/fxamacker/cbor/v2"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"

	log "github.com/sirupsen/logrus"
)

var _ block.BlockIndex = (*LebelDB)(nil)

var ErrCorrupted = fmt.Errorf("corrupted")

const (
	keyPrefixOID = "OID" // Followed by textual OID representation
	keyPrefixSeq = "SEQ" // Followed by a 16-digit hexadecimal sequence number (64 bit)
)

type LebelDB struct {
	path string
	mu   sync.Mutex
	db   *leveldb.DB
	seq  uint64
}

func keyFromOid(oid *oid.Oid) []byte {
	return append([]byte(keyPrefixOID), []byte(oid.String())...)
}

func keyFromSeq(seq uint64) []byte {
	return append([]byte(keyPrefixSeq), []byte(fmt.Sprintf("%016x", seq))...)
}

func seqFromKey(key []byte) (uint64, error) {
	if len(key) != len(keyPrefixSeq)+16 {
		return 0, fmt.Errorf("seqFromKey: invalid key length: %d", len(key))
	}
	if string(key[:len(keyPrefixSeq)]) != keyPrefixSeq {
		return 0, fmt.Errorf("seqFromKey: invalid key prefix: %s", string(key[:len(keyPrefixSeq)]))
	}
	var seq uint64
	if _, err := fmt.Sscanf(string(key[len(keyPrefixSeq):]), "%016x", &seq); err != nil {
		return 0, err
	}
	return seq, nil
}

func New(path string) (*LebelDB, error) {
	opts := &opt.Options{
		Compression: opt.NoCompression,
	}

	// Open or create the new DB
	db, err := leveldb.OpenFile(path, opts)
	if errors.IsCorrupted(err) {
		db, err = leveldb.RecoverFile(path, nil)
	}

	if err != nil {
		return nil, err
	}

	// Scan the database to identify the sequence
	iter := db.NewIterator(util.BytesPrefix([]byte(keyPrefixSeq)), nil)
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

	log.Infof("Opened LevelDB at %s, max sequence: %d", path, maxSeq)

	// Return the LebelDB object.
	// The sequence number will be initialized to the maximum value found in the database.
	// This ensures that new sequence numbers will be unique.

	return &LebelDB{db: db, path: path, seq: maxSeq}, nil
}

func (l *LebelDB) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.db.Close()
}

func (l *LebelDB) getByKey(key []byte) (*block.MetadataWithSeq, error) {
	// Read raw bytes
	raw, err := l.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	// Unmarshall CBOR
	metadata := &block.MetadataWithSeq{}
	err = cbor.Unmarshal(raw, metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

func (l *LebelDB) GetByOid(oid *oid.Oid) (*block.MetadataWithSeq, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := keyFromOid(oid)

	// Fetch the object
	md, err := l.getByKey(key)
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

func (l *LebelDB) GetBySeq(seq uint64) (*block.MetadataWithSeq, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := keyFromSeq(seq)

	// Fetch the object
	md, err := l.getByKey(key)
	if err != nil {
		return nil, err
	}

	// Compare the Sequence Number just in case
	if md.Sequence != seq {
		log.Errorf("GetBySeq: Sequence Number mismatch: %d != %d", seq, md.Sequence)
		return nil, ErrCorrupted
	}

	return md, nil
}

func (l *LebelDB) Put(metadata *block.MetadataWithSeq) (*block.MetadataWithSeq, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Fetch the existing metadata block
	oid := &metadata.Metadata.Oid

	existing, err := l.getByKey(keyFromOid(oid))
	if err != nil && err != errors.ErrNotFound {
		// ErrNotFound is acceptable here, everything else is not.
		return nil, err
	}

	// Compare the metadata (not the seqeuence number)
	if existing != nil && block.IsMetadataEqual(existing.Metadata, metadata.Metadata) {
		log.Debugf("Put: Metadata for OID %s is unchanged, skipping update", oid.String())
		return existing, nil
	}

	// Create the new sequence number
	newSeq := l.seq + 1

	// Copy the metadata object for writing
	md := &block.MetadataWithSeq{
		Sequence: newSeq,
		Metadata: metadata.Metadata,
	}

	// Marshall Metadata to CBOR
	raw, err := cbor.Marshal(md)
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

func (l *LebelDB) Has(oid *oid.Oid) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := keyFromOid(oid)

	// Check if the key exists
	_, err := l.db.Get(key, nil)
	if err == nil {
		return true, nil // Key exists
	} else if err == errors.ErrNotFound {
		return false, nil // Key does not exist
	} else {
		return false, err // Some other error occurred
	}
}

func (l *LebelDB) EnumerateBySeq(start uint64, end uint64) ([]*block.MetadataWithSeq, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if start > end {
		return nil, fmt.Errorf("EnumerateBySeq: invalid range: start (%d) > end (%d)", start, end)
	}

	keyStart := keyFromSeq(start)
	keyEnd := keyFromSeq(end)

	var results []*block.MetadataWithSeq

	// Create an iterator for the range of sequence numbers
	iter := l.db.NewIterator(&util.Range{Start: keyStart, Limit: keyEnd}, nil)
	defer iter.Release()

	// Iterate over the range and collect metadata entries
	for iter.Next() {
		raw := iter.Value()

		metadata := &block.MetadataWithSeq{}
		err := cbor.Unmarshal(raw, metadata)
		if err != nil {
			return nil, err
		}

		results = append(results, metadata)
	}

	return results, nil
}

func (l *LebelDB) GetSeq() uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.seq
}
