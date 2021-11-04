package types

import (
	"log"
	"net/http"
)

//go:generate msgp

// The Response type is specified by the Lighthouse-Protocol.
type Response struct {
	REID     interface{}
	RNUM     int
	RESPONSE string
	META     map[interface{}]interface{}
	PAYL     interface{}
	WARNINGS []interface{}
}

func (r *Response) Equals(o *Response) bool {
	// TODO: check META and WARNINGS
	return r.REID == o.REID && r.RNUM == o.RNUM && r.RESPONSE == o.RESPONSE && r.PAYL == o.PAYL
}

func NewResponse() *Response {
	return &Response{
		META:     map[interface{}]interface{}{},
		WARNINGS: []interface{}{},
	}
}
func (r *Response) Reid(reid interface{}) *Response {
	r.REID = reid
	return r
}
func (r *Response) Rnum(rnum int) *Response {
	r.RNUM = rnum
	return r
}
func (r *Response) Response(response string) *Response {
	r.RESPONSE = response
	return r
}
func (r *Response) Meta(key interface{}, value interface{}) *Response {
	r.META[key] = value
	return r
}
func (r *Response) Payload(payl interface{}) *Response {
	r.PAYL = payl
	return r
}
func (r *Response) Warning(warning string) *Response {
	var iface interface{} = warning
	r.WARNINGS = append(r.WARNINGS, iface)
	return r
}
func (r *Response) Build() *Response {
	if r.REID == nil {
		log.Panicln("REID must be set")
	}
	if http.StatusText(r.RNUM) == "" {
		log.Panicln("RNUM must be set and valid HTTP status code")
	}
	if r.RESPONSE == "" {
		r.RESPONSE = http.StatusText(r.RNUM)
	}
	return r
}
