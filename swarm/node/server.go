package node

import (
	"errors"
	"juren/swarm/protocol"

	log "github.com/sirupsen/logrus"
)

type Server struct {
	node *Node
}

// RPC: PeerSync
func (s *Server) PeerSync(req *protocol.PeerSyncRequest, res *protocol.PeerSyncResponse) error {
	log.Infof("Server.PeerSync from %s, seq: %d", req.NodeID.String(), req.SequenceNumber)
	blocks, err := s.node.BlockIndex.EnumerateBySeq(req.SequenceNumber, req.SequenceNumber+req.BatchSize)
	if err != nil {
		return err
	}
	res.NodeID = *s.node.NodeID
	res.Entries = blocks
	return nil
}

// RPC: BlockGet
func (s *Server) BlockGet(req *protocol.BlockGetRequest, res *protocol.BlockGetResponse) error {
	log.Infof("Server.BlockGet for %s", req.Oid.String())
	return errors.ErrUnsupported
}

// RPC: BlockPut
func (s *Server) BlockPut(req *protocol.BlockPutRequest, res *protocol.BlockPutResponse) error {
	log.Infof("Server.BlockPut for %s", req.Block.Oid.String())
	return errors.ErrUnsupported
}
