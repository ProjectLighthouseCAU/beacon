package resource

type Resource interface {
	Stream() (chan any, Response)
	StopStream(chan any) Response
	Put(any) Response
	Get() (any, Response)
	Link(Resource) Response
	UnLink(Resource) Response
	Close() Response
}

// type Channel chan any

// type Channel interface {
// 	Write(any)
// 	Read() (any, bool)
// 	Close()
// }

// Response struct for detailed response to the server
type Response struct {
	Code int
	Err  error
}
