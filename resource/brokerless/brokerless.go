package brokerless

import (
	"errors"
	"log"
	"strings"
	"sync"

	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/tinylib/msgp/msgp"
)

type brokerless struct {
	path []string

	streams     map[chan any]struct{}
	streamsLock sync.Mutex

	links     map[*brokerless]struct{}
	linksLock sync.Mutex

	value     any
	valueLock sync.RWMutex
}

var _ resource.Resource = (*brokerless)(nil)

var (
	streamChanSize = config.GetInt("RESOURCE_STREAM_CHANNEL_SIZE", 10)
)

func Create(path []string) resource.Resource {
	return &brokerless{
		path:        path,
		streams:     make(map[chan any]struct{}),
		streamsLock: sync.Mutex{},
		links:       make(map[*brokerless]struct{}),
		linksLock:   sync.Mutex{},
		value:       msgp.Raw{},
		valueLock:   sync.RWMutex{},
	}
}

// Close implements resource.Resource.
func (r *brokerless) Close() resource.Response {
	r.streamsLock.Lock()
	defer r.streamsLock.Unlock()

	r.linksLock.Lock()
	defer r.linksLock.Unlock()

	for stream := range r.streams {
		close(stream)
		delete(r.streams, stream)
	}

	for other := range r.links {
		delete(r.links, other)
	}

	return resource.Response{Code: 200, Err: nil}
}

// Get implements resource.Resource.
func (r *brokerless) Get() (interface{}, resource.Response) {
	r.valueLock.RLock()
	defer r.valueLock.RUnlock()
	return r.value, resource.Response{Code: 200, Err: nil}
}

// Put implements resource.Resource.
func (r *brokerless) Put(value interface{}) resource.Response {
	r.valueLock.Lock()
	r.value = value
	r.valueLock.Unlock()
	// TODO: if all streams and links should receive the values in the same order, we need to lock them
	for stream := range r.streams {
		select {
		case stream <- value:
		default:
			// skip stream if channel is full
			log.Printf("[Warning] A stream channel of %s is full and was skipped by the brokerless\n", strings.Join(r.path, "/"))
		}
	}
	for link := range r.links {
		link.Put(value)
	}
	return resource.Response{Code: 200, Err: nil}
}

// Stream implements resource.Resource.
func (r *brokerless) Stream() (chan interface{}, resource.Response) {
	r.streamsLock.Lock()
	defer r.streamsLock.Unlock()

	stream := make(chan any, streamChanSize)
	r.streams[stream] = struct{}{}
	return stream, resource.Response{Code: 200, Err: nil}
}

// StopStream implements resource.Resource.
func (r *brokerless) StopStream(stream chan interface{}) resource.Response {
	r.streamsLock.Lock()
	defer r.streamsLock.Unlock()

	_, ok := r.streams[stream]
	if !ok {
		return resource.Response{Code: 404, Err: errors.New("stream does not exist")}
	}
	close(stream)
	delete(r.streams, stream)
	return resource.Response{Code: 200, Err: nil}
}

// Link implements resource.Resource.
func (r *brokerless) Link(otherResource resource.Resource) resource.Response {
	other, ok := otherResource.(*brokerless)
	if !ok {
		return resource.Response{Code: 500, Err: errors.New("link resource must be of the same type as this resource (brokerless)")}
	}

	other.linksLock.Lock()
	defer other.linksLock.Unlock()

	if _, ok := other.links[r]; ok {
		return resource.Response{Code: 200, Err: errors.New("link already exists")}
	}

	if r.linksTo(other) {
		return resource.Response{Code: 409, Err: errors.New("link causes a loop")}
	}

	other.links[r] = struct{}{}

	return resource.Response{Code: 200, Err: nil}
}

// UnLink implements resource.Resource.
func (r *brokerless) UnLink(otherResource resource.Resource) resource.Response {
	other, ok := otherResource.(*brokerless)
	if !ok {
		return resource.Response{Code: 500, Err: errors.New("link resource must be of the same type as this resource (brokerless)")}
	}

	other.linksLock.Lock()
	defer other.linksLock.Unlock()

	_, ok = other.links[r]
	if !ok {
		return resource.Response{Code: 404, Err: errors.New("link does not exist")}
	}

	delete(other.links, r)

	return resource.Response{Code: 200, Err: nil}
}

// checks whether a resource links to another resource (using depth first search)
func (r *brokerless) linksTo(other *brokerless) bool {
	if other == r {
		return true
	}
	for res := range r.links {
		if res.linksTo(other) {
			return true
		}
	}
	return false
}
