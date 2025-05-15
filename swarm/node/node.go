package node

import (
	"context"
	"errors"
	"juren/datamodel/block"
	"juren/datamodel/node"
	"juren/net/crpc"
	"juren/net/mpubsub"
	"juren/oid"
	"juren/swarm/protocol"
	"net"
	"time"

	"golang.org/x/sync/errgroup"

	log "github.com/sirupsen/logrus"
)

type SwarmRPCServer struct {
	node *Node
}

type SwarmPubSub struct {
	node *Node
}

func (s *SwarmPubSub) PeerAnnouncement(msg *protocol.PeerAnnouncementMessage) {
	log.Infof("PeerAnnouncement: node: %s, addresses: %s, seq: %d", msg.NodeID.String(), msg.Addresses, msg.SequenceNumber)

	// Store the metadata in the NodeIndex
	_, err := s.node.nodeIndex.Put(&node.Metadata{
		NodeID:    msg.NodeID,
		Addresses: msg.Addresses,

		// FIXME: This is wrong, the locally stored SequenceNumber is the _synchronized_ number, not the announced number
		SequenceNumber: msg.SequenceNumber,
		LastSeen:       time.Now(),
	})
	if err != nil {
		log.Errorf("Failed to store node metadata: %v", err)
	}
}

func (s *SwarmPubSub) BlockAnnouncement(msg *protocol.BlockAnnouncementMessage) {
	log.Infof("BlockAnnouncement: node: %s, block: %s, presence: %v", msg.NodeID.String(), msg.Block.Oid.String(), msg.Has)
}

type Node struct {
	// Node ID
	nodeID    *oid.Oid
	addresses []string

	// Storage
	blockStore block.BlockStore
	blockIndex block.BlockIndex
	nodeIndex  node.NodeIndex

	// Networking
	rpcServer *crpc.Server
	pubsub    *mpubsub.PubSub

	// RPC and PubSub implementations
	rpcImpl    *SwarmRPCServer
	pubsubImpl *SwarmPubSub
}

func New(nodeID *oid.Oid, blockstore block.BlockStore, blockindex block.BlockIndex, nodeindex node.NodeIndex, rpcServer *crpc.Server, pubsub *mpubsub.PubSub) (*Node, error) {
	// Create the node object
	node := &Node{
		nodeID:     nodeID,
		blockStore: blockstore,
		blockIndex: blockindex,
		nodeIndex:  nodeindex,
	}

	// Figure out the IP addresses and ports on which the RPCServer is listening:
	addrs := rpcServer.Addr()

	// Populate the node.addresses with non-loopback addresses from addrs slice.
	for _, addr := range addrs {
		if tcpAddr, ok := addr.(*net.TCPAddr); ok {
			if !tcpAddr.IP.IsLoopback() {
				node.addresses = append(node.addresses, tcpAddr.String())
			}
		}
	}

	if len(node.addresses) == 0 {
		return nil, errors.New("no non-loopback addresses found")
	}

	// Set up RPC Server
	node.rpcImpl = &SwarmRPCServer{node: node}
	node.rpcServer = rpcServer
	node.rpcServer.Register(node.rpcImpl)

	// Set up PubSub
	node.pubsubImpl = &SwarmPubSub{node: node}
	node.pubsub = pubsub
	node.pubsub.Register(node.pubsubImpl)

	log.Infof("I am %s, listening on %s", node.nodeID.String(), node.addresses)

	return node, nil
}

func (n *Node) publishPeerAnnouncement(ctx context.Context) error {
	ticker := time.NewTicker(time.Second * 1)

	for {
		select {
		case <-ticker.C:
			// FIXME: Get the NodeID and Address
			msg := &protocol.PeerAnnouncementMessage{
				NodeID:         *n.nodeID,
				Addresses:      n.addresses,
				SequenceNumber: n.blockIndex.GetSeq(),
			}

			if err := n.pubsub.Publish("SwarmPubSub.PeerAnnouncement", msg); err != nil {
				log.Errorf("Failed to publish peer announcement: %v", err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (n *Node) Run(ctx context.Context) error {
	wg := errgroup.Group{}

	wg.Go(func() error {
		return n.pubsub.Listen(ctx)
	})

	wg.Go(func() error {
		return n.rpcServer.Serve(ctx)
	})

	wg.Go(func() error {
		return n.publishPeerAnnouncement(ctx)
	})

	wg.Go(func() error {
		// FIXME: Implement the main loop
		<-ctx.Done()
		return nil
	})

	err := wg.Wait()
	if err != nil {
		return err
	}

	return nil
}
