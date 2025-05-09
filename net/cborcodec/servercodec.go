package cborcodec

// Original implementation Copyright (c) 2022 Lukas Zapletal
// https://github.com/lzap/cborpc

import (
	"bufio"
	"io"
	"net/rpc"
	"sync"
)

type ServerCodec struct {
	rwc io.ReadWriteCloser
	wb  *bufio.Writer
	rmu sync.Mutex
	wmu sync.Mutex
}

func NewCBORServerCodec(rwc io.ReadWriteCloser) *ServerCodec {
	return &ServerCodec{
		rwc: rwc,
		wb:  bufio.NewWriter(rwc),
	}
}

func (c *ServerCodec) ReadRequestHeader(request *rpc.Request) error {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	return readAny(c.rwc, request)
}

func (c *ServerCodec) ReadRequestBody(body any) error {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	return readAny(c.rwc, body)
}

func (c *ServerCodec) WriteResponse(response *rpc.Response, payload any) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()

	return writeAny(c.wb, response, payload)
}

func (c *ServerCodec) Close() error {
	return c.rwc.Close()
}
