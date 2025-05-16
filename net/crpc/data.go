package crpc

type RequestHeader struct {
	Method string `cbor:"1,keyasint,omitempty"`
	Seq    uint64 `cbor:"2,keyasint,omitempty"`
	Token  string `cbor:"3,keyasint,omitempty"` // Authentication header: Bearer token
}

type ResponseHeader struct {
	Seq uint64 `cbor:"1,keyasint,omitempty"`
	Err string `cbor:"2,keyasint,omitempty"`
}
