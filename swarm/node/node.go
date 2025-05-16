package node

import (
	"context"
	"errors"
	"juren/config"
	"juren/datamodel/block"
	"juren/datamodel/node"
	"juren/net/crpc"
	"juren/net/mpubsub"
	"juren/oid"
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

func (n *Node) publishPeerAnnouncement(ctx context.Context) error {
	ticker := time.NewTicker(time.Second * 1)

	for {
		select {
		case <-ticker.C:
			// FIXME: Get the NodeID and Address
			msg := &protocol.PeerAnnouncementMessage{
				NodeID:         *n.NodeID,
				Addresses:      n.Addresses,
				SequenceNumber: n.BlockIndex.GetSeq(),
			}

			if err := n.PubSub.Publish("PubSub.PeerAnnouncement", msg); err != nil {
				log.Errorf("Failed to publish peer announcement: %v", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Initiates a block index sync with the given node. This function doesn't return an error
func (n *Node) SyncBlockIndexAndUpdateMetadata(newMetadata *node.Metadata) {
	// Check if we received our own announcement
	if newMetadata.NodeID == *n.NodeID {
		return
	}

	n.sg.Do("SyncBlockIndexAndUpdateMetadata", func() (interface{}, error) {
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
			log.Errorf("SyncBlockIndexAndUpdateMetadata(%s): known sequence number (%d) is greater than new sequence number (%d), ignoring",
				newMetadata.NodeID.String(), existingMetadata.SequenceNumber, newMetadata.SequenceNumber)
			return nil, nil
		}

		// If the sequence numbers match, we have nothing to do
		if existingMetadata.SequenceNumber == newMetadata.SequenceNumber {
			log.Infof("SyncBlockIndexAndUpdateMetadata(%s) no update needed", newMetadata.NodeID.String())

			// Update the metadata in the NodeIndex
			_, err := n.NodeIndex.Put(newMetadata)
			if err != nil {
				log.Errorf("SyncBlockIndexAndUpdateMetadata: failed to update node metadata: %v", err)
			}

			return nil, nil
		}

		log.Infof("Syncing block index with %s @ %s: %d -> %d", newMetadata.NodeID.String(), addr, existingMetadata.SequenceNumber, newMetadata.SequenceNumber)

		// TODO: Implement the actual sync logic

		return nil, nil
	})
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
		return n.publishPeerAnnouncement(cctx)
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
