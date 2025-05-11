package server

import (
	"context"
	"errors"
	"juren/datastore/block"
	"juren/net/cborrpc"
	"juren/oid"
	"juren/swarm/protocol"
	"net"
	"net/rpc"

	log "github.com/sirupsen/logrus"
)

type Server struct {
	rpc.Server
	nodeID      oid.Oid          // Node ID
	index       block.BlockIndex // Block Index Storage
	rpcListener net.Listener
}

func NewServer(nodeID oid.Oid, index block.BlockIndex) *Server {
	l, err := net.Listen("tcp", ":5001")
	if err != nil {
		log.Fatalf("Failed to create a listener: %v", err)
	}

	srv := &Server{
		nodeID:      nodeID,
		index:       index,
		rpcListener: l,
	}

	srv.Register(srv)
	return srv
}

func (s *Server) Serve(ctx context.Context) error {
	log.Info("Starting RPC server...")
	go cborrpc.Serve(s.rpcListener, s)
	return nil
}

// RPC: PeerSync
func (s *Server) PeerSync(req *protocol.PeerSyncRequest, res *protocol.PeerSyncResponse) error {
	log.Infof("Received PeerSync from %s, seq: %d", req.NodeID.String(), req.SequenceNumber)
	blocks, err := s.index.EnumerateBySeq(req.SequenceNumber, req.SequenceNumber+req.BatchSize)
	if err != nil {
		return err
	}
	res.NodeID = s.nodeID
	res.Entries = blocks
	return nil
}

// RPC: BlockFetch
func (s *Server) BlockGet(req *protocol.BlockGetRequest, res *protocol.BlockGetResponse) error {
	log.Infof("Received BlockFetch for %s", req.Oid.String())
	return errors.ErrUnsupported
}

// RPC: BlockPut
func (s *Server) BlockPut(req *protocol.BlockPutRequest, res *protocol.BlockPutResponse) error {
	log.Infof("Received BlockPut for %s", req.Block.Oid.String())
	return errors.ErrUnsupported
}
