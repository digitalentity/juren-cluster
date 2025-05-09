package server

import (
	"context"
	"juren/datastore/block"
	"juren/net/cborrpc"
	"juren/oid"
	"juren/swarm/protocol"
	"net"
	"net/rpc"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

type Server struct {
	nodeID      oid.Oid          // Node ID
	index       block.BlockIndex // Block Index Storage
	server      *rpc.Server      // RPC Server instance
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
		server:      rpc.NewServer(),
	}

	srv.server.Register(srv)

	return srv
}

func (s *Server) serve(ctx context.Context) error {
	for {
		conn, err := s.rpcListener.Accept()
		log.Infof("Accepted connection from %s", conn.RemoteAddr().String())
		if err != nil {
			log.Errorf("Failed to accept connection: %v", err)
			continue
		}
		go s.server.ServeCodec(cborrpc.NewCBORServerCodec(conn))
	}
}

func (s *Server) Serve(ctx context.Context) error {
	log.Info("Starting RPC server...")
	go s.serve(ctx)
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
