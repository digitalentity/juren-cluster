// Package leveldb implements the generic
package leveldb

import (
	"fmt"
	"juren/oid"
	"sync"

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

func initLevelDb(path string) (*leveldb.DB, error) {
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

	return db, nil
}

func (l *LebelDB) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.db.Close()
}
