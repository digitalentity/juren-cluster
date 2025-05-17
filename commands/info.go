package commands

import (
	"context"
	"crypto/sha256"
	"juren/config"
	"juren/datamodel/block"
	"juren/datastore/leveldb"
	"juren/oid"
	"time"

	log "github.com/sirupsen/logrus"
)

func RunInfo(ctx context.Context, cfg *config.Config) {
	bidx, err := leveldb.NewBlockIndex(cfg.DataStore.BlockIndexPath)
	if err != nil {
		log.Fatalf("Failed to create block index: %v", err)
	}

	nidx, err := leveldb.NewNodeIndex(cfg.DataStore.NodeIndexPath)
	if err != nil {
		log.Fatalf("Failed to create node index: %v", err)
	}

	o, _ := oid.Encode(oid.OidTypeRawBlock, sha256.Sum256([]byte("test2")))
	log.Infof("OID %s", o.String())
	blk := &block.ExtendedMedatadata{
		SequenceNumber: 0,
		Metadata: &block.Metadata{
			Oid:        *o,
			Length:     10,
			UpdateTime: time.Now(),
			IsDeleted:  false,
		},
	}
	md, err := bidx.Put(blk)
	if err != nil {
		log.Fatalf("Failed to put metadata: %v", err)
	}
	log.Infof("Put metadata: %s, seq: %d", md.Metadata.Oid.String(), md.SequenceNumber)

	nodes, err := nidx.Enumerate()
	if err != nil {
		log.Errorf("Failed to enumerate node index: %v", err)
		return
	}
	log.Infof("Node index: %d nodes known", len(nodes))
	for _, nodeid := range nodes {
		node, err := nidx.Get(nodeid)
		if err != nil {
			log.Errorf("Failed to get node metadata: %v", err)
			continue
		}
		log.Infof("Node: %s, addr: %s, seq: %d, last seen: %v", node.NodeID.String(), node.Addresses[0], node.SequenceNumber, node.LastSeen.Sub(time.Now()))
	}

	blocks, err := bidx.Enumerate()
	if err != nil {
		log.Errorf("Failed to enumerate block index: %v", err)
		return
	}

	log.Infof("Block index: %d blocks known", len(blocks))
	for _, block := range blocks {
		log.Infof("Block: %s, seq: %d, len: %d, updated: %v, deleted: %t",
			block.Metadata.Oid.String(), block.SequenceNumber, block.Metadata.Length, block.Metadata.UpdateTime, block.Metadata.IsDeleted)
	}
}
