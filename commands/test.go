package commands

import (
	"context"
	"juren/config"
	"juren/datastore/flatfs"
	"juren/datastore/leveldb"
	"juren/net/crpc"
	"juren/net/mpubsub"
	"juren/oid"
	"juren/swarm/node"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

func RunTest(ctx context.Context, cfg *config.Config) {
	log.Infof("Running test for ipfs-go-storage...")

	// Creating storage
	blk, err := flatfs.New(cfg.DataStore.BlockStorePath)
	if err != nil {
		log.Fatalf("Failed to create block storage: %v", err)
	}

	bidx, err := leveldb.NewBlockIndex(cfg.DataStore.BlockIndexPath)
	if err != nil {
		log.Fatalf("Failed to create block index: %v", err)
	}

	pidx, err := leveldb.NewNodeIndex(cfg.DataStore.NodeIndexPath)
	if err != nil {
		log.Fatalf("Failed to create node index: %v", err)
	}

	// Create the CRPC server and listerner
	rpcl, err := net.Listen("tcp4", cfg.Network.RpcListenAddress)
	if err != nil {
		log.Fatalf("Failed to create RPC listener: %v", err)
	}

	rsrv := crpc.NewServer(rpcl)

	// Create pubsub
	psaddr, err := net.ResolveUDPAddr("udp", cfg.Network.PubSubMulticastAddress)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}

	rs, err := net.ListenMulticastUDP("udp4", nil, psaddr)
	if err != nil {
		log.Fatalf("Failed to create multicast listener: %v", err)
	}

	ws, err := net.DialUDP("udp4", nil, psaddr)
	if err != nil {
		log.Fatalf("Failed to create multicast writer: %v", err)
	}

	pubsub := mpubsub.New(rs, ws)

	// Create the node
	node, err := node.New(cfg, blk, bidx, pidx, rsrv, pubsub)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	go func() {
		time.Sleep(1 * time.Second)

		o := oid.FromStringMustParse("BAAAAYBQHLRCXGMIMG6OHMUPGPXMDPTVRIQTZBWJHQDW3PU7KWGBDR2S")
		n, err := node.DiscoverBlock(ctx, o)
		if err != nil {
			log.Errorf("Failed to discover block: %v", err)
		}
		for _, nodeid := range n {
			log.Infof("Block %s is available at %s", o.String(), nodeid.String())
		}
	}()

	// Run the node
	if err := node.Run(ctx); err != nil {
		log.Fatalf("Failed to run node: %v", err)
	}
}
