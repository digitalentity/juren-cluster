package crpc

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/fxamacker/cbor/v2"

	log "github.com/sirupsen/logrus"
)

type ServerError string

func (e ServerError) Error() string {
	return string(e)
}

var ErrShutdown = errors.New("connection is shut down")

// Call represents an active RPC.
type Call struct {
	ServiceMethod string     // The name of the service and method to call.
	Args          any        // The argument to the function (*struct).
	Reply         any        // The reply from the function (*struct).
	Error         error      // After completion, the error status.
	Done          chan *Call // Receives *Call when Go is complete.
}

type Client struct {
	conn    io.ReadWriteCloser
	mutex   sync.Mutex // protects following
	seq     uint64
	pending map[uint64]*Call
}

func (client *Client) send(call *Call) {
	// Register this call.
	client.mutex.Lock()
	// if client.shutdown || client.closing {
	// 	client.mutex.Unlock()
	// 	call.Error = ErrShutdown
	// 	call.done()
	// 	return
	// }
	seq := client.seq
	client.seq++
	client.pending[seq] = call
	client.mutex.Unlock()

	// Encode and send the request.
	req := &RequestHeader{
		Method: call.ServiceMethod,
		Seq:    seq,
	}

	// Encode to CBOR and send
	encoder := cbor.NewEncoder(client.conn)
	err := encoder.Encode(req)
	if err == nil {
		err = encoder.Encode(call.Args)
	}
	if err != nil {
		client.mutex.Lock()
		delete(client.pending, seq)
		client.mutex.Unlock()
		call.Error = err
		call.done()
	}
}

func (call *Call) done() {
	select {
	case call.Done <- call:
		// ok
	default:
		// We don't want to block here. It is the caller's responsibility to make
		// sure the channel has enough buffer space. See comment in Go().
		log.Debugf("rpc: discarding Call reply due to insufficient Done chan capacity")
	}
}

func (client *Client) input() {
	var err error

	decoder := cbor.NewDecoder(client.conn)
	for err == nil {
		response := ResponseHeader{}
		err = decoder.Decode(&response)
		if err != nil {
			log.Errorf("rpc: client protocol error: %v", err)
			break
		}
		seq := response.Seq
		client.mutex.Lock()
		call := client.pending[seq]
		delete(client.pending, seq)
		client.mutex.Unlock()
		switch {
		case call == nil:
			// We've got no pending call. That usually means that WriteRequest partially failed, and call was already removed.
			// We should still attempt to read the body and ignore the potential error response.
			_ = decoder.Decode(call.Reply)
			log.Errorf("rpc: discarding Call reply due to no pending call")
		case response.Err != "":
			call.Error = ServerError(response.Err)
			call.done()
		default:
			err = decoder.Decode(call.Reply)
			if err != nil {
				call.Error = err
			}
			call.done()
		}
	}

	log.Infof("rpc.Client shutdown: %v", err)

	client.mutex.Lock()
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
	client.mutex.Unlock()
}

func NewClient(conn io.ReadWriteCloser) *Client {
	client := &Client{
		conn:    conn,
		pending: make(map[uint64]*Call),
	}
	go client.input()
	return client
}

// Dial connects to an RPC server at the specified network address.
func Dial(network, address string) (*Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(conn), nil
}

// Go invokes the function asynchronously. It returns the Call structure representing
// the invocation. The done channel will signal when the call is complete by returning
// the same Call object. If done is nil, Go will allocate a new channel.
// If non-nil, done must be buffered or Go will deliberately crash.
func (client *Client) Go(serviceMethod string, args any, reply any, done chan *Call) *Call {
	call := new(Call)
	call.ServiceMethod = serviceMethod
	call.Args = args
	call.Reply = reply
	if done == nil {
		done = make(chan *Call, 1) // buffered.
	}
	call.Done = done
	client.send(call)
	return call
}

// Call invokes the named function, waits for it to complete, and returns its error status.
func (client *Client) Call(ctx context.Context, serviceMethod string, args any, reply any) error {
	call := client.Go(serviceMethod, args, reply, make(chan *Call, 1))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case resp := <-call.Done:
		return resp.Error
	}
}
