package resource

type Resource interface {
	Stream() (chan interface{}, Response)
	StopStream(chan interface{}) Response
	Put(interface{}) Response
	Get() (interface{}, Response)
	Link(Resource) Response
	UnLink(Resource) Response
	Close() Response
}

// type Channel chan interface{}

// type Channel interface {
// 	Write(interface{})
// 	Read() (interface{}, bool)
// 	Close()
// }

// Response struct for detailed response to the server
type Response struct {
	Code int
	Err  error
}
