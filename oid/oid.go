package oid

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"

	log "github.com/sirupsen/logrus"
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

func (o *Oid) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

func (o *Oid) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	oid, err := FromString(s)
	if err != nil {
		return err
	}
	*o = *oid
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

func Random(t OidType) (*Oid, error) {
	// Generate 32 random bytes and craft a OID
	buf := make([]byte, 32)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}

	oid, err := Encode(t, [32]byte(buf))
	if err != nil {
		return nil, err
	}

	return oid, nil
}

// Equal helper
func (o *Oid) Equal(other *Oid) bool {
	if o == nil && other == nil {
		return true
	}
	if o == nil || other == nil {
		return false
	}
	return o.b == other.b
}
