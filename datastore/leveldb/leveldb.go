// Package leveldb implements the generic
package leveldb

import (
	"fmt"
	"juren/oid"
	"sync"

	"github.com/fxamacker/cbor/v2"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"

	log "github.com/sirupsen/logrus"
)

var ErrCorrupted = fmt.Errorf("corrupted")

type LebelDB struct {
	path string
	mu   sync.Mutex
	db   *leveldb.DB
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

	log.Infof("Opened LevelDB at %s", path)

	// Return the LebelDB object.
	// The sequence number will be initialized to the maximum value found in the database.
	// This ensures that new sequence numbers will be unique.

	return &LebelDB{db: db, path: path}, nil
}

func (l *LebelDB) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.db.Close()
}

func (l *LebelDB) getByKey(key []byte, placeholder any) (any, error) {
	// Read raw bytes
	raw, err := l.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	// Unmarshall CBOR
	err = cbor.Unmarshal(raw, placeholder)
	if err != nil {
		return nil, err
	}

	return placeholder, nil
}

func (l *LebelDB) putByKey(key []byte, value any) error {
	// Marshall CBOR
	raw, err := cbor.Marshal(value)
	if err != nil {
		return err
	}

	return l.db.Put(key, raw, nil)
}
