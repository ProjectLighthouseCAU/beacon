package resource

import (
	"errors"
	"net/http"

	"github.com/tinylib/msgp/msgp"
)

// used to quickly change all definitions of Resource for the specific type
// usually we only want one type of Content (previously any, now msgp.Raw)
type Content msgp.Raw

var Nil Content = msgp.AppendNil(msgp.Raw{}) // msgp.AppendInt8(msgp.Raw{}, 0) // msgp.AppendNil(msgp.Raw{}) // note: msgp.Raw with encoded nil is decoded as empty []byte

// Generic definition of a resource that implements storage and retrieval of a generic value (Put, Get) as well as the publish-subscribe mechanism (Stream, StopStream)
// and linking other resources (Link, Unlink) as well as a destructor/deinitialization-function (Close)
type Resource[T any] interface {
	Stream() chan T
	StopStream(chan T) error
	Put(T) error
	Get() T
	Link(Resource[T]) error
	UnLink(Resource[T]) error
	Close()
}

var (
	// 200
	ErrWarnLinkExists    = errors.New("link already exists")
	ErrWarnStreamSkipped = errors.New("a stream channel was skipped because it is full")
	// 404
	ErrStreamNotFound = errors.New("stream does not exist")
	ErrLinkNotFound   = errors.New("link does not exist")
	// 409
	ErrLinkLoop = errors.New("link causes a loop")
	// 500
	ErrWrongResourceImpl = errors.New("link resource must be of the same type as this resource")
)

func ErrorToStatusCode(err error) int {
	switch err {
	// 200 OK
	case nil:
		fallthrough
	case ErrWarnLinkExists:
		fallthrough
	case ErrWarnStreamSkipped:
		return http.StatusOK

	// 404 Not Found
	case ErrStreamNotFound:
		fallthrough
	case ErrLinkNotFound:
		return http.StatusNotFound

	// 409 Conflict
	case ErrLinkLoop:
		return http.StatusConflict

	// 500 Internal Server Error
	case ErrWrongResourceImpl:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
