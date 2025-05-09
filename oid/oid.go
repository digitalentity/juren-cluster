package oid

import (
	"encoding/base32"
	"errors"
	"log"
)

type OidType int

const (
	OidVersionV01 = 0x01

	OidTypeRawBlock  = 0x00 // Raw data. Replicated on select nodes in a swarm via consistent hashing approach.
	OidTypeObject    = 0x01 // Chunkset. Replicated on every node in a swarm.
	OidTypeObjectSet = 0x02 // Objectset. Replicated on every node in a swarm.
	OidTypeNode      = 0x03

	OidPaddingByte = 0xAA
)

var ErrorHashNot32Bytes = errors.New("hash must be 32 bytes")
var ErrorInvalidOidString = errors.New("invalid OID string")
var ErrorInvalidOidFormat = errors.New("invalid OID format")

// Byte structure of an OID is as follows <version:1><padding:1><type:1><hash:32>
// Raw bytes are encoded by Base32

// Oid structure holds the string representation of the OID as well as cached type and binary representation.
// Oid implements the MarshalBinary and UnmarshalBinary interfaces to assist CBOR encoding and avoid redundancy
type Oid struct {
	b [35]byte
	t OidType
	s string
}

func (o *Oid) String() string {
	return o.s
}

func (o *Oid) Type() OidType {
	return o.t
}

func (o *Oid) MarshalBinary() ([]byte, error) {
	return o.b[:], nil
}

func (o *Oid) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return ErrorInvalidOidFormat
	}

	switch data[0] {
	case OidVersionV01:
		if len(data) != 35 {
			return ErrorInvalidOidString
		}
		if data[1] != OidPaddingByte {
			return ErrorInvalidOidString
		}
		o.t = OidType(data[2])
		o.s = base32.StdEncoding.EncodeToString(data)
		copy(o.b[:], data)
	default:
		return ErrorInvalidOidFormat
	}

	return nil
}

func Encode(t OidType, hash [32]byte) (*Oid, error) {
	oidbytes := []byte{}

	// Add version and type
	oidbytes = append(oidbytes, byte(OidVersionV01))
	oidbytes = append(oidbytes, OidPaddingByte)
	oidbytes = append(oidbytes, byte(t))
	oidbytes = append(oidbytes, hash[:]...)

	o := &Oid{
		t: t,
		s: base32.StdEncoding.EncodeToString(oidbytes),
	}
	copy(o.b[:], oidbytes)
	return o, nil
}

func FromString(s string) (*Oid, error) {
	oidBytes, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	o := &Oid{}
	if err := o.UnmarshalBinary(oidBytes); err != nil {
		return nil, err
	}
	return o, nil
}

func FromStringMustParse(s string) *Oid {
	o, err := FromString(s)
	if err != nil {
		log.Fatalf("Failed to parse OID: %v", err)
	}
	return o
}
