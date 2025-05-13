// Package leveldb implements the block.BlockIndex interface
package leveldb

import (
	"juren/datastore/block"
	"juren/oid"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var _ block.BlockIndex = (*LebelDB)(nil)

type LebelDB struct {
	mu   sync.Mutex
	path string
	db   *leveldb.DB
}

func New(path string) (*LebelDB, error) {
	opts := &opt.Options{
		Compression: opt.NoCompression,
	}

	db, err := leveldb.OpenFile(path, opts)
	if errors.IsCorrupted(err) {
		db, err = leveldb.RecoverFile(path, nil)
	}

	if err != nil {
		return nil, err
	}

	return &LebelDB{db: db, path: path}, nil
}

func (l *LebelDB) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.db.Close()
}

func (l *LebelDB) GetByOid(*oid.Oid) (*block.MetadataWithSeq, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return nil, nil
}
