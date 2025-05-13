package node

import (
	"juren/config"
	"juren/datastore/block"
	"juren/net/crpc"
	"juren/net/mpubsub"
)

type SwarmRPCServer struct {
	crpc.Server
	node *Node
}

type SwarmPubSub struct {
	mpubsub.PubSub
	node *Node
}

type Node struct {
	// Storage
	blockStore block.BlockStore
	blockIndex block.BlockIndex

	// Networking
	rpcServer SwarmRPCServer
	pubsub    SwarmPubSub

	// Node registry
	peerRegistry PeerRegistry
}

func New(cfg *config.Config) (*Node, error) {
	node := &Node{}
	return node, nil
}

func (n *Node) Start() error {
	return nil
}

func (n *Node) Stop() error {
	return nil
}
