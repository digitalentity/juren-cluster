package crpc

type RequestHeader struct {
	Seq    uint64 `cbor:"1,keyasint,omitempty"`
	Method string `cbor:"2,keyasint,omitempty"`
}

type ResponseHeader struct {
	Seq uint64 `cbor:"1,keyasint,omitempty"`
	Err string `cbor:"2,keyasint,omitempty"`
}
