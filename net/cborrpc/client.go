package cborrpc

import (
	"bufio"
	"io"
	"net/rpc"
	"sync"
)

type ClientCodec struct {
	rwc io.ReadWriteCloser
	wb  *bufio.Writer
	rmu sync.Mutex
	wmu sync.Mutex
}

func NewCBORClientCodec(rwc io.ReadWriteCloser) *ClientCodec {
	return &ClientCodec{
		rwc: rwc,
		wb:  bufio.NewWriter(rwc),
	}
}

func (c *ClientCodec) WriteRequest(request *rpc.Request, payload any) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()

	return writeAny(c.wb, request, payload)
}

func (c *ClientCodec) ReadResponseHeader(response *rpc.Response) error {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	return readAny(c.rwc, response)
}

func (c *ClientCodec) ReadResponseBody(body any) error {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	return readAny(c.rwc, body)
}

func (c *ClientCodec) Close() error {
	return c.rwc.Close()
}
