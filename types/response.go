package types

import (
	"log"
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/tinylib/msgp/msgp"
)

//go:generate msgp

// The Response type is specified by the Lighthouse-Protocol.
type Response struct {
	REID     msgp.Raw
	RNUM     int
	RESPONSE string
	META     map[any]any
	PAYL     msgp.Raw
	WARNINGS []string
}

func NewResponse() *Response {
	return &Response{
		META:     map[any]any{},
		WARNINGS: []string{},
	}
}
func (r *Response) Reid(reid msgp.Raw) *Response {
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
func (r *Response) Meta(key any, value any) *Response {
	r.META[key] = value
	return r
}
func (r *Response) Payload(payl resource.Content) *Response {
	r.PAYL = (msgp.Raw)(payl)
	return r
}
func (r *Response) Warning(warning string) *Response {
	r.WARNINGS = append(r.WARNINGS, warning)
	return r
}
func (r *Response) Build() *Response {
	if r.REID == nil {
		log.Println("REID must be set")
		r.REID = []byte{0xc0} // msgpack nil
		r.RNUM = http.StatusInternalServerError
	}
	statusText := http.StatusText(r.RNUM)
	if statusText == "" {
		log.Println("RNUM must be set and valid HTTP status code")
		r.RNUM = http.StatusInternalServerError
	}
	if r.RESPONSE == "" {
		r.RESPONSE = statusText
	}
	return r
}
