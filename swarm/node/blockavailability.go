package node

import (
	"juren/oid"
	"sync"
)

type blockInfo struct {
	nodes       map[oid.Oid]bool
	subscribers map[uint64]chan *oid.Oid
}

type BlockAvailabilityTracker struct {
	mu     sync.Mutex
	seq    uint64 // Used for sync
	blocks map[oid.Oid]*blockInfo
}

func NewBlockAvailabilityTracker() *BlockAvailabilityTracker {
	return &BlockAvailabilityTracker{
		blocks: make(map[oid.Oid]*blockInfo),
	}
}

func (b *BlockAvailabilityTracker) getBlockInfo(blockOid *oid.Oid) *blockInfo {
	// No lock here, lock is assumed to be acquired by caller
	if bi, ok := b.blocks[*blockOid]; ok {
		return bi
	}

	bi := &blockInfo{
		nodes:       make(map[oid.Oid]bool),
		subscribers: make(map[uint64]chan *oid.Oid),
	}
	b.blocks[*blockOid] = bi
	return bi
}

func (b *BlockAvailabilityTracker) Update(blockOid *oid.Oid, nodeOid *oid.Oid, available bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	bi := b.getBlockInfo(blockOid)
	bi.nodes[*nodeOid] = available

	// Notify subscribers about an available block
	if available {
		for _, ch := range bi.subscribers {
			ch <- nodeOid
		}
	}
}

// // SubscribeForAvailable for updates on a given block and wait until at least one node ID is received
// func (b *BlockAvailabilityTracker) SubscribeForAvailable(ctx context.Context, blockOid *oid.Oid) (*oid.Oid, error) {
// 	// Set up the channel that will be notified once a given block is updated
// 	b.mu.Lock()
// 	seq := b.seq
// 	bi := b.getBlockInfo(blockOid)
// 	ch := make(chan *oid.Oid)
// 	bi.subscribers[seq] = ch
// 	b.seq++
// 	b.mu.Unlock()

// 	// Defer cleanup: close the channel and delete the subscriber
// 	defer func() {
// 		b.mu.Lock()
// 		close(ch)
// 		delete(bi.subscribers, seq)
// 		b.mu.Unlock()
// 	}()

// 	select {
// 	case <-ctx.Done():
// 		return nil, ctx.Err()
// 	case nodeOid := <-ch:
// 		return nodeOid, nil
// 	}
// }

func (b *BlockAvailabilityTracker) WhoHas(blockOid *oid.Oid) []*oid.Oid {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.blocks[*blockOid]; !ok {
		return nil
	}

	var nodes []*oid.Oid
	for nodeOid, available := range b.blocks[*blockOid].nodes {
		if available {
			nodes = append(nodes, &nodeOid)
		}
	}

	return nodes
}
