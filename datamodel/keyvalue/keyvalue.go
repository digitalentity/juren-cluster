package keyvalue

type Key []byte

type KeyValue interface {
	Has(key Key) (bool, error)
	Put(key Key, value []byte) error
	Get(key Key) ([]byte, error)
	GetRange(start, end Key) ([]any, error)
	Delete(key Key) error
}
