package node

import (
	"juren/datamodel/node"
	"juren/swarm/protocol"
	"time"

	log "github.com/sirupsen/logrus"
)

type PubSub struct {
	node *Node
}

func (s *PubSub) PeerAnnouncement(msg *protocol.PeerAnnouncementMessage) {
	log.Infof("PeerAnnouncement: node: %s, addresses: %s, seq: %d", msg.NodeID.String(), msg.Addresses, msg.SequenceNumber)

	// Store the metadata in the NodeIndex
	_, err := s.node.NodeIndex.Put(&node.Metadata{
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

func (s *PubSub) BlockAnnouncement(msg *protocol.BlockAnnouncementMessage) {
	log.Infof("BlockAnnouncement: node: %s, block: %s, presence: %v", msg.NodeID.String(), msg.Block.Oid.String(), msg.Has)
}
