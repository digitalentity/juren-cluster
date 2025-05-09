package client

import (
	"context"
	"juren/net/cborrpc"
	"juren/swarm/protocol"
)

type Client struct {
	*cborrpc.Client
}

func Dial(address string) (*Client, error) {
	c, err := cborrpc.DialCBOR(address)
	if err != nil {
		return nil, err
	}
	return &Client{c}, nil
}

func (c *Client) PeerSync(ctx context.Context, req *protocol.PeerSyncRequest) (*protocol.PeerSyncResponse, error) {
	res := &protocol.PeerSyncResponse{}
	err := c.Call(ctx, "Server.PeerSync", req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) BlockGet(ctx context.Context, req *protocol.BlockGetRequest) (*protocol.BlockGetResponse, error) {
	res := &protocol.BlockGetResponse{}
	err := c.Call(ctx, "Server.BlockGet", req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) BlockPut(ctx context.Context, req *protocol.BlockPutRequest) (*protocol.BlockPutResponse, error) {
	res := &protocol.BlockPutResponse{}
	err := c.Call(ctx, "Server.BlockPut", req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
