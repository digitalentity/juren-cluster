package crpc

type RequestHeader struct {
	Method string `cbor:"1,keyasint,omitempty"`
	Seq    uint64 `cbor:"2,keyasint,omitempty"`
}

type ResponseHeader struct {
	Seq uint64 `cbor:"1,keyasint,omitempty"`
	Err string `cbor:"2,keyasint,omitempty"`
}
