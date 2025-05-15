// Package mpubsub implements a Multicast PubSub.
// Publish: a CBOR-encoded message is sent to a multicast group.
// Subscribe: a listener receives a message over the network and distributes it to a registered callback.
package mpubsub

import (
	"bytes"
	"context"
	"go/token"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/fxamacker/cbor/v2"

	log "github.com/sirupsen/logrus"
)

type MessageHeader struct {
	ServiceMethod string `cbor:"1,keyasint,omitempty"`
}

type handlerType struct {
	method  reflect.Method
	argType reflect.Type
}

type service struct {
	name    string
	sub     reflect.Value
	typ     reflect.Type
	methods map[string]*handlerType
}

type PubSub struct {
	rc         *net.UDPConn
	wc         *net.UDPConn
	serviceMap sync.Map
}

func New(rconn *net.UDPConn, wconn *net.UDPConn) *PubSub {
	return &PubSub{
		rc: rconn,
		wc: wconn,
	}
}

func (ps *PubSub) Register(rcvr any) {
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.sub = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.sub).Type().Name()
	if sname == "" {
		log.Errorf("mpubsub.Subscribe: no service name for type %s", s.typ.String())
		return
	}
	if !token.IsExported(sname) {
		log.Errorf("mpubsub.Subscribe: type %q is not exported", sname)
		return
	}
	s.name = sname

	// Install the methods
	s.methods = suitableHandlers(s.typ)
	if len(s.methods) == 0 {
		str := "mpubsub.Subscribe: type " + sname + " has no exported methods of suitable type"
		log.Error(str)
		return
	}
	ps.serviceMap.Store(sname, s)

	// Some debug logging
	for m := range s.methods {
		log.Debugf("mpubsub.Subscribe: %s.%s\n", sname, m)
	}
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type, so we need to check the type name as well.
	return token.IsExported(t.Name()) || t.PkgPath() == ""
}
func suitableHandlers(typ reflect.Type) map[string]*handlerType {
	handlers := make(map[string]*handlerType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if !method.IsExported() {
			continue
		}
		// Method needs one in: receiver, *args.
		if mtype.NumIn() != 2 {
			log.Errorf("mpubsub.Subscribe: method %q has %d input parameters; needs exactly two\n", mname, mtype.NumIn())
			continue
		}
		// First arg must be a pointer.
		argType := mtype.In(1)
		if argType.Kind() != reflect.Pointer {
			log.Errorf("mpubsub.Subscribe: argument type of method %q is not a pointer: %q\n", mname, argType)
			continue
		}
		// Arg type must be exported.
		if !isExportedOrBuiltinType(argType) {
			log.Errorf("mpubsub.Subscribe: argument type of method %q is not exported: %q\n", mname, argType)
			continue
		}
		// Method needs zero out.
		if mtype.NumOut() != 0 {
			log.Errorf("mpubsub.Subscribe: method %q has %d output parameters; needs exactly zero\n", mname, mtype.NumOut())
			continue
		}
		handlers[mname] = &handlerType{method: method, argType: argType}
	}
	return handlers
}

func (ps *PubSub) Publish(serviceMethod string, args interface{}) error {
	msg := MessageHeader{
		ServiceMethod: serviceMethod,
	}

	buf := new(bytes.Buffer)
	enc := cbor.NewEncoder(buf)
	enc.Encode(msg)
	enc.Encode(args)

	_, err := ps.wc.Write(buf.Bytes())
	if err != nil {
		return err
	}

	// log.Debugf("mpubsub: published message to %s (%d bytes)", serviceMethod, buf.Len())

	return nil
}

func (ps *PubSub) Listen(ctx context.Context) error {
	// It's good practice to define a buffer size, e.g., based on expected max message size or MTU.
	// For UDP, typical Ethernet MTU is 1500, minus IP/UDP headers, so ~1400 bytes is safe.
	// Let's use a reasonably sized buffer.
	buf := make([]byte, 2048) // Increased buffer size slightly

	// Set a read deadline to allow periodic checks of the context.
	// This makes the ReadFromUDP call non-blocking in the long run.
	deadlineInterval := time.Millisecond * 100

	for {
		select {
		case <-ctx.Done():
			log.Infof("mpubsub: context cancelled, stopping listener for %s", ps.rc.LocalAddr())
			return ctx.Err()

		default:
			// Set a deadline for the next read operation.
			if err := ps.rc.SetReadDeadline(time.Now().Add(deadlineInterval)); err != nil {
				log.Errorf("mpubsub: failed to set read deadline: %v", err)
				continue
			}

			n, _, err := ps.rc.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Timeout is expected, continue to check context.
					continue
				}
				log.Errorf("mpubsub: failed to read message: %v", err)
				// For other errors, we might want to log and continue, or return.
				continue
			}

			// Wrap the message in a reader and pass on to CBOR decoder
			dec := cbor.NewDecoder(bytes.NewReader(buf[:n]))

			var msg MessageHeader
			err = dec.Decode(&msg)
			if err != nil {
				log.Errorf("mpubsub: failed to unmarshal message header: %v", err)
				continue
			}

			dot := strings.LastIndex(msg.ServiceMethod, ".")
			if dot < 0 {
				log.Errorf("mpubsub: service/method request ill-formed: %s", msg.ServiceMethod)
				continue
			}
			serviceName := msg.ServiceMethod[:dot]
			methodName := msg.ServiceMethod[dot+1:]

			svci, ok := ps.serviceMap.Load(serviceName)
			if !ok {
				log.Warnf("mpubsub: can't find service %s for method %s", serviceName, msg.ServiceMethod)
				continue
			}
			svc := svci.(*service)

			handler := svc.methods[methodName]
			if handler == nil {
				log.Warnf("mpubsub: can't find handler for method %s", msg.ServiceMethod)
				continue
			}

			arg := reflect.New(handler.argType.Elem())
			err = dec.Decode(arg.Interface())
			if err != nil {
				log.Errorf("mpubsub: failed to unmarshal arguments for %s: %v", msg.ServiceMethod, err)
				continue
			}

			// log.Debugf("mpubsub: received message for %s", msg.ServiceMethod)

			function := handler.method.Func
			// Consider running the handler in a goroutine if it can block for a long time,
			// but be mindful of potential message ordering issues if that's important.
			// For now, direct call:
			function.Call([]reflect.Value{svc.sub, arg})
		}
	}
}
