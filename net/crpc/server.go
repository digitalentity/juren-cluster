package crpc

import (
	"context"
	"errors"
	"fmt"
	"go/token"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fxamacker/cbor/v2"

	log "github.com/sirupsen/logrus"
)

type methodType struct {
	sync.Mutex // protects counters
	method     reflect.Method
	ArgType    reflect.Type
	ReplyType  reflect.Type
	numCalls   uint
}

type service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

type Server struct {
	listener   net.Listener
	serviceMap sync.Map // map[string]*service
}

func NewServer(listener net.Listener) *Server {
	return &Server{
		listener: listener,
	}
}

func (srv *Server) Register(rcvr any) error {
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	if sname == "" {
		s := fmt.Sprintf("rpc.Register: no service name for type %s", s.typ.String())
		log.Error(s)
		return errors.New(s)
	}
	if !token.IsExported(sname) {
		s := "rpc.Register: type " + sname + " is not exported"
		log.Error(s)
		return errors.New(s)
	}
	s.name = sname

	// Install the methods
	s.method = suitableMethods(s.typ)
	if len(s.method) == 0 {
		str := "rpc.Register: type " + sname + " has no exported methods of suitable type"
		log.Error(str)
		return errors.New(str)
	}

	if _, dup := srv.serviceMap.LoadOrStore(sname, s); dup {
		return errors.New("rpc: service already defined: " + sname)
	}

	// Some debug logging
	for m := range s.method {
		log.Debugf("rpc.Register: %s.%s\n", sname, m)
	}

	return nil
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type, so we need to check the type name as well.
	return token.IsExported(t.Name()) || t.PkgPath() == ""
}

// suitableMethods returns suitable Rpc methods of typ.
func suitableMethods(typ reflect.Type) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if !method.IsExported() {
			continue
		}
		// Method needs three ins: receiver, *args, *reply.
		if mtype.NumIn() != 3 {
			log.Errorf("rpc.Register: method %q has %d input parameters; needs exactly three\n", mname, mtype.NumIn())
			continue
		}
		// First arg need not be a pointer.
		argType := mtype.In(1)
		if !isExportedOrBuiltinType(argType) {
			log.Errorf("rpc.Register: argument type of method %q is not exported: %q\n", mname, argType)
			continue
		}
		// Second arg must be a pointer.
		replyType := mtype.In(2)
		if replyType.Kind() != reflect.Pointer {
			log.Errorf("rpc.Register: reply type of method %q is not a pointer: %q\n", mname, replyType)
			continue
		}
		// Reply type must be exported.
		if !isExportedOrBuiltinType(replyType) {
			log.Errorf("rpc.Register: reply type of method %q is not exported: %q\n", mname, replyType)
			continue
		}
		// Method needs one out.
		if mtype.NumOut() != 1 {
			log.Errorf("rpc.Register: method %q has %d output parameters; needs exactly one\n", mname, mtype.NumOut())
			continue
		}
		// The return type of the method must be error.
		if returnType := mtype.Out(0); returnType != reflect.TypeFor[error]() {
			log.Errorf("rpc.Register: return type of method %q is %q, must be error\n", mname, returnType)
			continue
		}
		methods[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}
	}
	return methods
}

func (srv *Server) Serve(ctx context.Context) error {
	// Start a goroutine to close the listener when the context is cancelled.
	// This will cause srv.listener.Accept() to return an error.
	go func() {
		<-ctx.Done()
		log.Infof("crpc.Server: context cancelled, initiating shutdown for listener %s", srv.listener.Addr())
		// Closing the listener will cause the Accept loop to unblock.
		err := srv.listener.Close()
		if err != nil {
			// Log if closing the listener itself failed, but proceed with shutdown.
			log.Warnf("crpc.Server: error closing listener %s: %v", srv.listener.Addr(), err)
		}
	}()

	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		rw, err := srv.listener.Accept()
		if err != nil {
			// Check if the context is cancelled. This is the primary signal for shutdown.
			select {
			case <-ctx.Done():
				// Context was cancelled, listener.Close() was called (or is about to be).
				// Accept() returning an error is expected in this case.
				log.Infof("crpc.Server: shutting down listener %s due to context cancellation.", srv.listener.Addr())
				return ctx.Err()
			default:
				// Context not cancelled yet, so the error from Accept() is for another reason.
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					if tempDelay == 0 {
						tempDelay = 5 * time.Millisecond
					} else {
						tempDelay *= 2
					}
					if max := 1 * time.Second; tempDelay > max {
						tempDelay = max
					}
					log.Warnf("crpc.Server: Accept error on %s: %v; retrying in %v", srv.listener.Addr(), err, tempDelay)
					time.Sleep(tempDelay)
					continue
				}
				// If the error is not a timeout, and context is not done,
				// it's likely a non-recoverable error for the listener.
				log.Errorf("crpc.Server: critical accept error on %s: %v. Server stopping.", srv.listener.Addr(), err)
				return err // Return the unexpected error.
			}
		}

		tempDelay = 0 // Reset tempDelay on successful accept
		log.Infof("crpc.Server: accepted connection from %s on %s", rw.RemoteAddr().String(), srv.listener.Addr())
		go srv.serveConn(ctx, rw)
	}
}

func (srv *Server) serveConn(ctx context.Context, conn net.Conn) {
	decoder := cbor.NewDecoder(conn)
	defer func() {
		// log.Debugf("crpc.Server: serveConn for %s finished, closing connection.", conn.RemoteAddr())
		conn.Close()
	}()

	for {
		// Check context before attempting to read.
		select {
		case <-ctx.Done():
			log.Infof("crpc.Server: serveConn for %s stopping due to server context cancellation.", conn.RemoteAddr())
			return // Exit the goroutine if context is cancelled
		default:
			// Proceed with decoding
		}

		// Read the argument header
		req := &RequestHeader{}
		err := decoder.Decode(req)
		if err != nil {
			logMessage := fmt.Sprintf("crpc.Server: error decoding request header for %s: %v", conn.RemoteAddr(), err)
			if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "use of closed network connection") {
				log.Debugf("crpc.Server: connection %s closed (EOF or closed explicitly): %v", conn.RemoteAddr(), err)
			} else {
				log.Error(logMessage)
			}
			return
		}

		// Parse the header
		dot := strings.LastIndex(req.Method, ".")
		if dot < 0 {
			log.Errorf("crpc.Server: service/method request ill-formed: %q from %s", req.Method, conn.RemoteAddr())
			return
		}
		serviceName := req.Method[:dot]
		methodName := req.Method[dot+1:]

		// Look up the request.
		svci, ok := srv.serviceMap.Load(serviceName)
		if !ok {
			log.Errorf("crpc.Server: can't find service %q for method %q from %s", serviceName, req.Method, conn.RemoteAddr())
			return
		}
		svc := svci.(*service)
		mtype := svc.method[methodName]
		if mtype == nil {
			log.Errorf("crpc.Server: can't find method %q for service %q from %s", methodName, serviceName, conn.RemoteAddr())
			return
		}

		// Decode the argument value
		var argv reflect.Value
		if mtype.ArgType.Kind() == reflect.Pointer {
			argv = reflect.New(mtype.ArgType.Elem())
		} else {
			argv = reflect.New(mtype.ArgType)
		}

		err = decoder.Decode(argv.Interface())
		if err != nil {
			log.Errorf("crpc.Server: error decoding argument for %s.%s on connection %s: %v", serviceName, methodName, conn.RemoteAddr(), err)
			return
		}

		repl := &ResponseHeader{Seq: req.Seq}
		replyv := reflect.New(mtype.ReplyType.Elem())

		// Call the service
		var callErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("crpc.Server: panic during RPC call %s.%s from %s: %v", serviceName, methodName, conn.RemoteAddr(), r)
					callErr = fmt.Errorf("rpc: internal server error during %s.%s", serviceName, methodName)
				}
			}()
			callErr = svc.call(mtype, argv, replyv)
		}()

		if callErr != nil {
			repl.Err = callErr.Error()
		}

		// Encode the Response Header
		encoder := cbor.NewEncoder(conn)
		err = encoder.Encode(repl)
		if err != nil {
			log.Errorf("crpc.Server: error encoding response header for %s.%s on connection %s: %v", serviceName, methodName, conn.RemoteAddr(), err)
			return
		}

		// Encode response body if call error was nil
		if callErr == nil {
			err = encoder.Encode(replyv.Interface())
			if err != nil {
				log.Errorf("crpc.Server: error encoding response body for %s.%s on connection %s: %v", serviceName, methodName, conn.RemoteAddr(), err)
				return
			}
		}
	}
}

func (svc *service) call(mtype *methodType, argv, replyv reflect.Value) error {
	mtype.Lock()
	mtype.numCalls++
	mtype.Unlock()
	function := mtype.method.Func
	// Invoke the method, providing a new value for the reply.
	returnValues := function.Call([]reflect.Value{svc.rcvr, argv, replyv})
	// The return value for the method is an error.
	errInter := returnValues[0].Interface()
	if errInter != nil {
		return errInter.(error)
	}
	return nil
}

// Addr returns all IP addresses on which the server is listening.
// If the listener is bound to a specific IP, it returns that IP:port.
// If the listener is bound to an unspecified IP (e.g., 0.0.0.0 or ::) or an address like ":port",
// it enumerates IP addresses of active network interfaces and returns them with the listener's port.
func (srv *Server) Addr() []net.Addr {
	if srv.listener == nil {
		log.Error("crpc.Server.Addr: listener is nil")
		return []net.Addr{}
	}

	listenerAddr := srv.listener.Addr()
	if listenerAddr == nil {
		log.Error("crpc.Server.Addr: listener.Addr() returned nil")
		return []net.Addr{}
	}

	var listenIP net.IP
	var port int
	var networkType = listenerAddr.Network() // e.g., "tcp", "udp"

	switch a := listenerAddr.(type) {
	case *net.TCPAddr:
		listenIP = a.IP
		port = a.Port
	case *net.UDPAddr: // Should not happen for crpc, but good for completeness
		listenIP = a.IP
		port = a.Port
	default:
		hostStr, portStr, err := net.SplitHostPort(listenerAddr.String())
		if err != nil {
			log.Errorf("crpc.Server.Addr: failed to parse listener address '%s': %v", listenerAddr.String(), err)
			return []net.Addr{listenerAddr} // Fallback to returning the original problematic address
		}
		p, err := strconv.Atoi(portStr)
		if err != nil {
			log.Errorf("crpc.Server.Addr: failed to parse port from listener address '%s': %v", listenerAddr.String(), err)
			return []net.Addr{listenerAddr}
		}
		port = p
		if hostStr != "" {
			listenIP = net.ParseIP(hostStr)
		}
		// If hostStr was empty (e.g. ":8080"), listenIP remains nil, indicating "any"
	}

	if port == 0 {
		log.Errorf("crpc.Server.Addr: could not determine port for listener: %s", listenerAddr.String())
		return []net.Addr{listenerAddr} // Fallback
	}

	var addresses []net.Addr

	// IsUnspecified correctly handles nil IPs (it returns false for nil).
	// So, if listenIP is nil (from ":port") or an unspecified IP (0.0.0.0, ::), this is false.
	isSpecificIP := listenIP != nil && !listenIP.IsUnspecified()

	if isSpecificIP {
		// Return the specific address the listener is bound to.
		// We reconstruct it to ensure it's the correct type, e.g. *net.TCPAddr
		// based on the original listener's network type.
		switch networkType {
		case "tcp", "tcp4", "tcp6":
			addresses = append(addresses, &net.TCPAddr{IP: listenIP, Port: port})
		case "udp", "udp4", "udp6": // Unlikely for CRPC but included for robustness
			addresses = append(addresses, &net.UDPAddr{IP: listenIP, Port: port})
		default:
			// Fallback for unknown network types, return the original address
			log.Warnf("crpc.Server.Addr: unhandled network type '%s' for specific IP, returning original listener address", networkType)
			addresses = append(addresses, listenerAddr)
		}
		return addresses
	}

	// If listenIP is unspecified (0.0.0.0, ::) or nil (e.g. from ":port"),
	// iterate over all network interfaces.
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Errorf("crpc.Server.Addr: failed to get network interfaces: %v", err)
		// Fallback to returning the original "any" address if we can't list interfaces
		return []net.Addr{listenerAddr}
	}

	for _, iface := range interfaces {
		if (iface.Flags & net.FlagUp) == 0 {
			continue // Interface is down
		}

		ifaddrs, err := iface.Addrs()
		if err != nil {
			log.Warnf("crpc.Server.Addr: could not get addresses for interface %s: %v", iface.Name, err)
			continue
		}

		for _, addr := range ifaddrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsUnspecified() { // Also skip unspecified IPs from interfaces themselves
				continue
			}

			// For CRPC, we assume TCP.
			// Filter IP version based on the listener's "any" address type if it was explicit.
			// If listenIP is nil (from ":port"), we accept both IPv4 and IPv6.
			isIPv4 := ip.To4() != nil
			if listenIP != nil { // Original listener was explicit 0.0.0.0 or ::
				if listenIP.Equal(net.IPv4zero) && !isIPv4 { // Listener was 0.0.0.0, IP is IPv6
					continue
				}
				if listenIP.Equal(net.IPv6unspecified) && isIPv4 { // Listener was ::, IP is IPv4
					// Note: dual-stack [::] can accept IPv4. This lists interface IPs.
					// For simplicity, if explicitly [::], we list IPv6 interface IPs.
					continue
				}
			}
			addresses = append(addresses, &net.TCPAddr{IP: ip, Port: port})
		}
	}

	// Deduplicate addresses
	if len(addresses) > 0 {
		seen := make(map[string]struct{})
		uniqueAddresses := make([]net.Addr, 0, len(addresses))
		for _, addr := range addresses {
			addrStr := addr.String() // Use string representation for deduplication key
			if _, exists := seen[addrStr]; !exists {
				seen[addrStr] = struct{}{}
				uniqueAddresses = append(uniqueAddresses, addr)
			}
		}
		addresses = uniqueAddresses
	}

	// If no specific IPs were found for an "any" listener (e.g., no active non-loopback interfaces),
	// it's more informative to return the original listenerAddr than an empty list.
	if len(addresses) == 0 && !isSpecificIP {
		log.Warnf("crpc.Server.Addr: no specific IP addresses found for 'any' listener, returning original listener address: %s", listenerAddr.String())
		return []net.Addr{listenerAddr}
	}

	return addresses
}
