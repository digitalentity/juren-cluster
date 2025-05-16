package node

import (
	"context"
	"errors"
	"fmt"
	"juren/config"
	"juren/datamodel/block"
	"juren/datamodel/node"
	"juren/helper/timer"
	"juren/net/crpc"
	"juren/net/mpubsub"
	"juren/oid"
	"juren/swarm/client"
	"juren/swarm/protocol"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	log "github.com/sirupsen/logrus"
)

type Node struct {
	// Node ID
	NodeID    *oid.Oid
	Addresses []string

	// Storage
	BlockStore block.BlockStore
	BlockIndex block.BlockIndex
	NodeIndex  node.NodeIndex

	// Networking
	RpcServer *crpc.Server
	PubSub    *mpubsub.PubSub

	// RPC and PubSub implementations
	RpcHandlers    *Server
	PubSubHandlers *PubSub

	// Helpers
	sg singleflight.Group
}

func New(cfg *config.Config, blockstore block.BlockStore, blockindex block.BlockIndex, nodeindex node.NodeIndex, rpcServer *crpc.Server, pubsub *mpubsub.PubSub) (*Node, error) {
	// Create the node object
	node := &Node{
		NodeID:     cfg.Node.NodeID,
		BlockStore: blockstore,
		BlockIndex: blockindex,
		NodeIndex:  nodeindex,
	}

	if cfg.Network.RpcAdvertizedAddress != "" {
		node.Addresses = append(node.Addresses, cfg.Network.RpcAdvertizedAddress)
	} else {
		// Figure out the IP addresses and ports on which the RPCServer is listening:
		addrs := rpcServer.Addr()

		// Populate the node.addresses with non-loopback addresses from addrs slice.
		for _, addr := range addrs {
			if tcpAddr, ok := addr.(*net.TCPAddr); ok {
				if !tcpAddr.IP.IsLoopback() {
					node.Addresses = append(node.Addresses, tcpAddr.String())
				}
			}
		}
	}

	if len(node.Addresses) == 0 {
		return nil, errors.New("no non-loopback addresses found")
	}

	log.Infof("Advertized RPC addresses: %s", node.Addresses)

	// Set up RPC Server
	node.RpcHandlers = &Server{node: node}
	node.RpcServer = rpcServer
	node.RpcServer.Register(node.RpcHandlers)

	// Set up PubSub
	node.PubSubHandlers = &PubSub{node: node}
	node.PubSub = pubsub
	node.PubSub.Register(node.PubSubHandlers)

	log.Infof("I am %s, listening on %s", node.NodeID.String(), node.Addresses)

	return node, nil
}

// This is run via the RunWithTicker() helper
func (n *Node) publishPeerAnnouncement(ctx context.Context) error {
	msg := &protocol.PeerAnnouncementMessage{
		NodeID:         *n.NodeID,
		Addresses:      n.Addresses,
		SequenceNumber: n.BlockIndex.GetSeq(),
	}

	if err := n.PubSub.Publish("PubSub.PeerAnnouncement", msg); err != nil {
		log.Errorf("Failed to publish peer announcement: %v", err)
	}

	return nil
}

// Initiates a block index sync with the given node. This function doesn't return an error
func (n *Node) syncBlockIndexAndUpdateMetadata(newMetadata *node.Metadata) {
	// Check if we received our own announcement
	if newMetadata.NodeID == *n.NodeID {
		log.Debugf("Received our own announcement - ignoring")
		return
	}

	_, err, _ := n.sg.Do("SyncBlockIndexAndUpdateMetadata", func() (interface{}, error) {
		addr := newMetadata.Addresses[0]

		// Fetch the metadata from the NodeIndex
		existingMetadata, err := n.NodeIndex.Get(&newMetadata.NodeID)
		if err != nil {
			// If the node metadata is not found in the NodeIndex, we assume we never saw this node and need to sync.
			existingMetadata = &node.Metadata{
				NodeID:         newMetadata.NodeID,
				SequenceNumber: 0,
			}
		}

		// This is a potential bug (duplicate NodeID)
		if existingMetadata.SequenceNumber > newMetadata.SequenceNumber {
			return nil, fmt.Errorf("SyncBlockIndexAndUpdateMetadata(%s): known sequence (%d) is greater than new sequence (%d), ignoring",
				newMetadata.NodeID.String(), existingMetadata.SequenceNumber, newMetadata.SequenceNumber)
		}

		// If the sequence numbers match, we have nothing to do
		if existingMetadata.SequenceNumber == newMetadata.SequenceNumber {
			log.Infof("SyncBlockIndexAndUpdateMetadata(%s) no update needed", newMetadata.NodeID.String())

			// Update the metadata in the NodeIndex
			_, err := n.NodeIndex.Put(newMetadata)
			if err != nil {
				return nil, fmt.Errorf("failed to update node metadata: %w", err)
			}

			return nil, nil
		}

		log.Infof("Syncing block index with %s @ %s: %d -> %d", newMetadata.NodeID.String(), addr, existingMetadata.SequenceNumber, newMetadata.SequenceNumber)

		// Connect to the announcing node.
		rpcc, err := crpc.Dial("tcp4", addr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
		}

		c := &client.Client{Client: rpcc}
		defer c.Close()

		// Keep track of the block sequence up to which we synched
		maxSeenBlockSeq := existingMetadata.SequenceNumber

		// Fetch blocks in batches until we're up to date
		for {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			ps, err := c.PeerSync(ctx, &protocol.PeerSyncRequest{
				NodeID:         newMetadata.NodeID,
				SequenceNumber: maxSeenBlockSeq, // +1 ?
				BatchSize:      100,
			})

			if err != nil {
				return nil, fmt.Errorf("failed to sync block index: %w", err)
			}

			// We received no blocks this time - we're done
			if len(ps.Entries) == 0 {
				break
			}

			log.Infof("SyncBlockIndexAndUpdateMetadata: received %d blocks from %s", len(ps.Entries), newMetadata.NodeID.String())
			for _, block := range ps.Entries {
				_, err := n.BlockIndex.Put(block)
				if err != nil {
					return nil, fmt.Errorf("failed to put block metadata: %w", err)
				}
				maxSeenBlockSeq = max(maxSeenBlockSeq, block.SequenceNumber)
			}

			// We're done processing this batch, write Node Metadata to avoid loosing progress

			// Override the sequence number - record the number until which we have synced
			newMetadata.SequenceNumber = maxSeenBlockSeq

			// Update the metadata in the NodeIndex. This only happens if we successfully fetched and processed all the blocks
			_, err = n.NodeIndex.Put(newMetadata)
			if err != nil {
				return nil, fmt.Errorf("failed to update node metadata: %v", err)
			}
		}

		return nil, nil
	})

	if err != nil {
		log.Errorf("SyncBlockIndexAndUpdateMetadata: %v", err)
	}
}

func (n *Node) Run(ctx context.Context) error {
	wg, cctx := errgroup.WithContext(ctx)

	wg.Go(func() error {
		return n.PubSub.Listen(cctx)
	})

	wg.Go(func() error {
		return n.RpcServer.Serve(cctx)
	})

	wg.Go(func() error {
		interval := &timer.Interval{
			Duration: time.Second * 5,
			Jitter:   time.Millisecond * 0,
		}
		return timer.RunWithTicker(cctx, interval, n.publishPeerAnnouncement)
	})

	wg.Go(func() error {
		// FIXME: Implement the main loop
		<-cctx.Done()
		return nil
	})

	err := wg.Wait()
	if err != nil {
		return err
	}

	return nil
}
