// Package flatfs implements the block/BlockStore interface
package flatfs

import (
	"juren/datamodel/block"
	"juren/oid"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// Do an indirection to make sure FlatFS implementa the required interfaces
var _ block.BlockStore = (*FlatFS)(nil)

// FlatFS implements the block.BlockStore interface
// File name is a string representation of the OID.
// The storage is organized by the OID. First 4 characters of the OID string representation are used as a subdirectory.
// The block length is infered from the file length, the file on disk stores raw data without any additional metadata.

type FlatFS struct {
	basePath string
}

func New(basePath string) (*FlatFS, error) {
	// Sanitize the basePath
	basePath = filepath.Clean(basePath)

	// Make sure the directory exists and create if missing
	if err := ensureDir(basePath); err != nil {
		return nil, err
	}

	log.Infof("Opened FlatFS at %s", basePath)

	return &FlatFS{basePath: basePath}, nil
}

// ensureDir checks if a directory exists at the given path, and if not, creates it.
func ensureDir(path string) error {
	// Stat the path
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory does not exist, create it
			return os.MkdirAll(path, 0755)
		}
		return err
	}
	if !stat.IsDir() {
		return &os.PathError{Op: "ensureDir", Path: path, Err: os.ErrExist}
	}
	return nil
}

// Enumerate creates a list of OIDs of all existing blocks in the FlatFS.
// It scans the directory structure, attempting to parse filenames as OIDs.
// Warnings are logged for entries that don't conform to the expected structure or cannot be parsed as OIDs.
func (f *FlatFS) Enumerate() ([]*oid.Oid, error) {
	var oids []*oid.Oid

	// Read the top-level directories (shards based on OID prefix)
	shardDirEntries, err := os.ReadDir(f.basePath)
	if err != nil {
		log.Errorf("Error reading base path %s for enumeration: %v", f.basePath, err)
		return nil, err
	}

	for _, shardDirEntry := range shardDirEntries {
		if !shardDirEntry.IsDir() {
			// Expected structure is basePath/prefix/OID_filename
			log.Warnf("Skipping non-directory entry in FlatFS base path during enumeration: %s", filepath.Join(f.basePath, shardDirEntry.Name()))
			continue
		}

		shardPath := filepath.Join(f.basePath, shardDirEntry.Name())
		blockFileEntries, err := os.ReadDir(shardPath)
		if err != nil {
			log.Errorf("Error reading shard directory %s during enumeration: %v", shardPath, err)
			// To ensure a complete list or none, we return an error here.
			// If partial results were acceptable, this behavior could be changed.
			return nil, err
		}

		for _, blockFileEntry := range blockFileEntries {
			if blockFileEntry.IsDir() {
				log.Warnf("Skipping unexpected subdirectory in shard %s during enumeration: %s", shardPath, blockFileEntry.Name())
				continue
			}

			oidStr := blockFileEntry.Name()
			o, parseErr := oid.FromString(oidStr)
			if parseErr != nil {
				log.Warnf("Skipping file %s in shard %s during enumeration, not a valid OID: %v", oidStr, shardPath, parseErr)
				continue
			}
			oids = append(oids, o)
		}
	}

	return oids, nil
}

func (f *FlatFS) Close() error {
	return nil
}

func (f *FlatFS) Get(oid *oid.Oid) (*block.Block, error) {
	_, filePath := f.oidToPath(oid)

	data, err := os.ReadFile(filePath)
	if err != nil {
		// os.ReadFile returns an error that os.IsNotExist can check.
		// It also correctly returns an error if filePath is a directory.
		return nil, err
	}

	b := &block.Block{
		Oid:    *oid, // The OID used to retrieve the block
		Length: uint64(len(data)),
		Data:   data,
	}

	return b, nil
}

// oidToPath converts an OID to its corresponding file path within the FlatFS structure.
// It also returns the directory path.
func (f *FlatFS) oidToPath(oid *oid.Oid) (dirPath string, filePath string) {
	oidStr := oid.String()

	// The storage is organized by the OID. First 4 characters of the OID string representation are used as a subdirectory.
	// The block length is infered from the file length, the file on disk stores raw data without any additional metadata.
	dirPath = filepath.Join(f.basePath, oidStr[:4])
	filePath = filepath.Join(dirPath, oidStr)

	return dirPath, filePath
}

func (f *FlatFS) Has(oid *oid.Oid) (bool, error) {
	_, filePath := f.oidToPath(oid)
	stat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !stat.IsDir(), nil
}

func (f *FlatFS) Put(b *block.Block) (*oid.Oid, error) {
	if b == nil {
		return nil, os.ErrInvalid // Or a more specific error
	}

	dirPath, filePath := f.oidToPath(&b.Oid)

	// Ensure the subdirectory exists
	if err := ensureDir(dirPath); err != nil {
		return nil, err
	}

	// Write the block data to the file.
	// os.WriteFile creates the file if it doesn't exist, and truncates it if it does.
	// Using 0644 permissions (rw-r--r--).
	err := os.WriteFile(filePath, b.Data, 0644)
	return &b.Oid, err
}

func (f *FlatFS) Delete(oid *oid.Oid) error {
	_, filePath := f.oidToPath(oid)

	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, it's already "deleted".
			// Consider this a success for the Delete operation.
			return nil
		}
		// For other errors (e.g., permission issues), return the error.
		return err
	}
	return nil
}
