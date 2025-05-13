package node

import "time"

type Peer struct {
	Address        string
	SequenceNumber uint64
	LastSeenTime   time.Time
}

type PeerRegistry struct {
	peers map[string]*Peer
}
