package client

import (
	"juren/net/cborrpc"
	"juren/swarm/protocol"
	"net"
	"net/rpc"
)

type Client struct {
	conn   net.Conn
	client rpc.Client
}

func Dial(address string) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:   conn,
		client: *rpc.NewClientWithCodec(cborrpc.NewCBORClientCodec(conn)),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) PeerSync(req *protocol.PeerSyncRequest) (*protocol.PeerSyncResponse, error) {
	res := &protocol.PeerSyncResponse{}
	err := c.client.Call("Server.PeerSync", req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
