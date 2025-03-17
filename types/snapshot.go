package types

import "github.com/tinylib/msgp/msgp"

//go:generate msgp

// The snapshot type defines the contents of the snapshot.beacon file
// It maps paths (concatenated with "/") to resource contents (raw msgpack)
type Snapshot map[string]msgp.Raw

func NewSnapshot() Snapshot {
	return make(map[string]msgp.Raw)
}
