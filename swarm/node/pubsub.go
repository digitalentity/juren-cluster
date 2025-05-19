package node

import (
	"juren/datamodel/node"
	"juren/net/mpubsub"
	"juren/swarm/protocol"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

type PubSub struct {
	node *Node
}

func (s *PubSub) PeerAnnouncement(sender *mpubsub.PublisherAddress, msg *protocol.PeerAnnouncementMessage) {
	log.Infof("PeerAnnouncement from %s: node: %s, addresses: %s, seq: %d", sender.IP.String(), msg.NodeID.String(), msg.Addresses, msg.SequenceNumber)

	// Filter msg.Addresses to match the sender IP (if there is a match)
	var filteredAddresses []string
	for _, addr := range msg.Addresses {
		na, err := net.ResolveTCPAddr("tcp4", addr)
		if err != nil {
			log.Errorf("Failed to resolve address %s: %v", addr, err)
			continue
		}
		if na.IP.String() == sender.IP.String() {
			filteredAddresses = append(filteredAddresses, addr)
		}
	}

	// If there is no match, use the provided addresses regardless
	if len(filteredAddresses) == 0 {
		filteredAddresses = msg.Addresses
	}

	// Build the new node metadata:
	newmd := &node.Metadata{
		NodeID:         msg.NodeID,
		Addresses:      filteredAddresses,
		SequenceNumber: msg.SequenceNumber,
		LastSeen:       time.Now(),
		Capabilities:   node.BuildCapabilityMap(msg.Capabilities),
		StorageUsage:   msg.StorageUsage,
	}

	// Initiate a sync and metadata update. The syncBlockIndexAndUpdateMetadata will take care of the rest
	go s.node.syncBlockIndexAndUpdateMetadata(newmd)

	// Initiate block replication (will trigger resync either when the node has not been seen for a while or periodically)
	// go s.node.ProcessBlockReplication(newmd)
}

func (s *PubSub) BlockAnnouncement(sender *mpubsub.PublisherAddress, msg *protocol.BlockAnnouncementMessage) {
	log.Infof("BlockAnnouncement from %s: node: %s, block: %s, has: %t", sender.IP.String(), msg.NodeID.String(), msg.Block.Oid.String(), msg.Has)

	go s.node.updateBlockAvailability(msg)
}

func (s *PubSub) BlockDiscoveryMessage(sender *mpubsub.PublisherAddress, msg *protocol.BlockDiscoveryMessage) {
	log.Infof("BlockDiscoveryMessage from %s: block: %s", sender.IP.String(), msg.Oid.String())

	go s.node.processBlockDiscoveryMessage(msg)
}
