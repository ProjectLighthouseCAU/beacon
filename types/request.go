package types

import (
	"errors"
	"strconv"
)

//go:generate msgp

// The Request type is specified by the Lighthouse-Protocol.
type Request struct {
	REID interface{}
	AUTH map[string]interface{}
	VERB string
	PATH []string
	META map[interface{}]interface{}
	PAYL interface{}
}

// PayloadToPath interprets the payload as a path and returns the path as string[] (error if payload is not a path)
func (r *Request) PayloadToPath() ([]string, error) {
	p, ok := r.PAYL.([]interface{})
	if !ok {
		return nil, errors.New("Payload is not an array")
	}
	var strings []string
	for i, v := range p {
		vStr, ok := v.(string)
		if !ok {
			return nil, errors.New("Payload array contains non-string entry at " + strconv.Itoa(i))
		}
		strings = append(strings, vStr)
	}
	return strings, nil
}
