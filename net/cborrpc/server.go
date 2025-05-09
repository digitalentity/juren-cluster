package cborrpc

import (
	"juren/net/cborcodec"
	"net"
	"net/rpc"
	"time"
)

type ServerInterface interface {
	ServeCodec(c rpc.ServerCodec)
}

type Server struct {
	Handler ServerInterface
}

func (srv *Server) Serve(l net.Listener) error {
	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		rw, err := l.Accept()
		log.Infof("Accepted connection from %s", rw.RemoteAddr().String())
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Errorf("CBOR-RPC: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		go srv.Handler.ServeCodec(cborcodec.NewCBORServerCodec(rw))
	}

}

// Serve accepts incoming CBOR-RPC connections on the listener l,
// creating a new service goroutine for each. The service goroutines
// read requests and then call handler to reply to them.
func Serve(l net.Listener, handler ServerInterface) error {
	srv := &Server{Handler: handler}
	return srv.Serve(l)
}
