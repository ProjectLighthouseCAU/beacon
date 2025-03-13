package types

import (
	"errors"

	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/tinylib/msgp/msgp"
	"github.com/vmihailenco/msgpack/v5"
)

//go:generate msgp

// The Request type is specified by the Lighthouse-Protocol.
type Request struct {
	REID msgp.Raw
	AUTH map[string]string
	VERB string
	PATH []string
	META map[any]any
	PAYL msgp.Raw
}

// PayloadToPath interprets the payload as a path and returns the path as string[] (error if payload is not a path)
func (r *Request) PayloadToPath() ([]string, error) {
	var path []string
	err := msgpack.Unmarshal(([]byte)(r.PAYL), &path)
	if err != nil {
		return nil, errors.New("Payload is not a path ([]string)")
	}
	return path, nil
}

func (r *Request) PayloadToContent() resource.Content {
	// special case: msgpack.Nil is decoded as empty array
	// empty arrays are decoded as [0x90] (msgpack array header with length 0)
	if len(r.PAYL) == 0 {
		return resource.Nil
	}
	return (resource.Content)(r.PAYL)
}
