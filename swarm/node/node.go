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
