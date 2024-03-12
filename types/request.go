package types

import (
	"errors"

	"github.com/tinylib/msgp/msgp"
	"github.com/vmihailenco/msgpack"
)

//go:generate msgp

// The Request type is specified by the Lighthouse-Protocol.
type Request struct {
	REID msgp.Raw
	AUTH map[string]string
	VERB string
	PATH []string
	META map[interface{}]interface{}
	PAYL msgp.Raw
}

// PayloadToPath interprets the payload as a path and returns the path as string[] (error if payload is not a path)
func (r *Request) PayloadToPath() ([]string, error) {
	var path []string
	err := msgpack.Unmarshal(([]byte)(r.PAYL), &path)
	if err != nil {
		return nil, errors.New("Payload is not an array of strings")
	}
	return path, nil
}
