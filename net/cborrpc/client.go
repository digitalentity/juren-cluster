package cborrpc

import (
	"context"
	"juren/net/cborcodec"
	"net"
	"net/rpc"

	log "github.com/sirupsen/logrus"
)

type Client struct {
	*rpc.Client
}

func DialCBOR(address string) (*Client, error) {
	log.Debugf("Dialing %s", address)
	conn, err := net.DialTimeout("tcp", address, DialTimeout)
	if err != nil {
		return nil, err
	}
	codec := cborcodec.NewCBORClientCodec(conn)
	return &Client{rpc.NewClientWithCodec(codec)}, nil
}

func (c *Client) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	call := c.Go(serviceMethod, args, reply, make(chan *rpc.Call, 1))
	select {
	case <-ctx.Done():
		c.Close()
		return ctx.Err()
	case resp := <-call.Done:
		return resp.Error
	}
}
