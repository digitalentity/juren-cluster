package crpc

import (
	"errors"
	"fmt"
	"go/token"
	"net"
	"reflect"
	"strings"
	"sync"

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
	serviceMap sync.Map // map[string]*service
}

func NewServer() *Server {
	return &Server{}
}

func (srv *Server) Register(rcvr any) error {
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	if sname == "" {
		s := fmt.Sprintf("rpc.Register: no service name for type %s", s.typ.String())
		log.Errorf(s)
		return errors.New(s)
	}
	if !token.IsExported(sname) {
		s := "rpc.Register: type " + sname + " is not exported"
		log.Print(s)
		return errors.New(s)
	}
	s.name = sname

	// Install the methods
	s.method = suitableMethods(s.typ)
	if len(s.method) == 0 {
		str := ""

		// To help the user, see if a pointer receiver would work.
		method := suitableMethods(reflect.PointerTo(s.typ))
		if len(method) != 0 {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type"
		}
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

func (srv *Server) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go srv.serveConn(conn)
	}
}

func (srv *Server) serveConn(conn net.Conn) {
	decoder := cbor.NewDecoder(conn)
	defer conn.Close()
	for {
		// Read the argument header
		req := &RequestHeader{}
		err := decoder.Decode(req)
		if err != nil {
			log.Errorf("Error decoding request header: %v", err)
			return
		}

		// Parse the header
		dot := strings.LastIndex(req.Method, ".")
		if dot < 0 {
			log.Errorf("rpc: service/method request ill-formed: %s", req.Method)
			return
		}
		serviceName := req.Method[:dot]
		methodName := req.Method[dot+1:]

		// Look up the request.
		svci, ok := srv.serviceMap.Load(serviceName)
		if !ok {
			log.Errorf("rpc: can't find service %s", req.Method)
			return
		}
		svc := svci.(*service)
		mtype := svc.method[methodName]
		if mtype == nil {
			log.Errorf("rpc: can't find method %s", req.Method)
			return
		}

		// Decode the argument value
		argv := reflect.New(mtype.ArgType.Elem())
		err = decoder.Decode(argv.Interface())
		if err != nil {
			log.Errorf("Error decoding argument: %v", err)
			return
		}

		repl := &ResponseHeader{Seq: req.Seq}
		replyv := reflect.New(mtype.ReplyType.Elem())

		// Call the service
		callerr := svc.call(mtype, argv, replyv)
		repl.Err = callerr.Error()

		// Encode the Response Header
		encoder := cbor.NewEncoder(conn)
		err = encoder.Encode(repl)
		if err != nil {
			log.Errorf("Error encoding response header: %v", err)
			return
		}

		// Encode response body if call error was nil
		if callerr == nil {
			err = encoder.Encode(replyv.Interface())
			if err != nil {
				log.Errorf("Error encoding response body: %v", err)
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
