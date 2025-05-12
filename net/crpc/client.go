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
	conn     io.ReadWriteCloser
	mutex    sync.Mutex // protects following fields
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // user has called Close
	shutdown bool // server has told us to stop
}

func (client *Client) send(call *Call) {
	// Register this call.
	client.mutex.Lock()
	if client.closing || client.shutdown {
		client.mutex.Unlock()
		call.Error = ErrShutdown
		call.done()
		return
	}
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

	// If either request encoding fails, we should remove the call from pending map
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
			// Error will be handled by the cleanup logic after the loop
			break
		}

		seq := response.Seq

		client.mutex.Lock()
		call, ok := client.pending[seq]
		delete(client.pending, seq)
		client.mutex.Unlock()

		switch {
		case call == nil:
			// We've got no pending call. That usually means that WriteRequest partially failed, and call was already removed.
			// We should still attempt to read the body and ignore the potential error response.
			if response.Err == "" { // Only try to read body if header indicates no error
				var dummy interface{}
				if e := decoder.Decode(&dummy); e != nil {
					err = e // This error will break the main loop
					log.Warnf("rpc: error consuming body for unknown sequence %d: %v", seq, err)
				}
			}
			log.Warnf("rpc: received reply for unknown sequence %d (call %t), discarding", seq, ok)
			// If err was set by decoder.Decode(&dummy), the outer loop 'for err == nil' will break.

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

	// Terminate pending calls
	client.mutex.Lock()
	defer client.mutex.Unlock()

	client.shutdown = true
	shutdownError := err // Default to the error that broke the loop
	if client.closing {  // If Close() was called, ErrShutdown is the reason
		shutdownError = ErrShutdown
	} else if err == io.EOF || errors.Is(err, net.ErrClosed) {
		// If it's a connection closed error and Close() wasn't called,
		// it's an unexpected closure. ErrShutdown is a reasonable error.
		shutdownError = ErrShutdown
	}
	// For other errors (e.g., decoding malformed packet), 'err' (which is shutdownError by default) will be propagated.

	if err != nil && err != io.EOF && !errors.Is(err, net.ErrClosed) {
		log.Warnf("rpc: client input loop error: %v. Notifying pending calls with: %v", err, shutdownError)
	} else if err == io.EOF || errors.Is(err, net.ErrClosed) {
		log.Debugf("rpc: client connection closed/EOF. Notifying pending calls with: %v", shutdownError)
	} else { // err == nil (should not happen with for err == nil unless loop broken by other means)
		log.Debugf("rpc: client input loop terminated cleanly. Notifying pending calls with: %v", shutdownError)
	}

	for _, call := range client.pending {
		call.Error = shutdownError
		call.done()
	}
	client.pending = make(map[uint64]*Call) // Clear pending calls
}

func NewClient(conn io.ReadWriteCloser) *Client {
	client := &Client{
		conn:    conn,
		pending: make(map[uint64]*Call),
		// closing and shutdown are false by default
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

// Close calls the underlying connection's Close method.
// If the connection is already shutting down, ErrShutdown is returned.
func (client *Client) Close() error {
	client.mutex.Lock()
	if client.closing {
		client.mutex.Unlock()
		return ErrShutdown
	}
	client.closing = true
	client.mutex.Unlock()
	return client.conn.Close() // This will cause client.input() to exit and cleanup
}
