package brokerless

import (
	"log"
	"strings"
	"sync"

	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/resource"
)

type brokerless[T any] struct {
	path []string

	streams     map[chan T]struct{}
	streamsLock sync.Mutex

	links     map[*brokerless[T]]struct{}
	linksLock sync.Mutex

	value     T // exported for serialization during snapshotting
	valueLock sync.RWMutex
}

var _ resource.Resource[resource.Content] = (*brokerless[resource.Content])(nil)

func Create[T any](path []string, initialValue T) resource.Resource[T] {
	return &brokerless[T]{
		path:        path,
		streams:     make(map[chan T]struct{}),
		streamsLock: sync.Mutex{},
		links:       make(map[*brokerless[T]]struct{}),
		linksLock:   sync.Mutex{},
		value:       initialValue,
		valueLock:   sync.RWMutex{},
	}
}

// Close implements resource.Resource.
func (r *brokerless[T]) Close() {
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
}

// Get implements resource.Resource.
func (r *brokerless[T]) Get() T {
	r.valueLock.RLock()
	defer r.valueLock.RUnlock()
	return r.value
}

// Put implements resource.Resource.
func (r *brokerless[T]) Put(value T) error {
	r.valueLock.Lock()
	r.value = value
	r.valueLock.Unlock()
	// TODO: if all streams and links should receive the values in the same order, we need to lock them
	anyStreamSkipped := false
	for stream := range r.streams {
		select {
		case stream <- value:
		default:
			anyStreamSkipped = true
			// skip stream if channel is full
			if config.VerboseLogging {
				log.Printf("[Warning] A stream channel of %s is full and was skipped by the brokerless\n", strings.Join(r.path, "/"))
			}
		}
	}
	for link := range r.links {
		link.Put(value)
	}
	if anyStreamSkipped {
		return resource.ErrWarnStreamSkipped
	}
	return nil
}

// Stream implements resource.Resource.
func (r *brokerless[T]) Stream() chan T {
	r.streamsLock.Lock()
	defer r.streamsLock.Unlock()

	stream := make(chan T, config.ResourceStreamChannelSize)
	r.streams[stream] = struct{}{}
	return stream
}

// StopStream implements resource.Resource.
func (r *brokerless[T]) StopStream(stream chan T) error {
	r.streamsLock.Lock()
	defer r.streamsLock.Unlock()

	_, ok := r.streams[stream]
	if !ok {
		return resource.ErrStreamNotFound
	}
	close(stream)
	delete(r.streams, stream)
	return nil
}

// Link implements resource.Resource.
func (r *brokerless[T]) Link(otherResource resource.Resource[T]) error {
	other, ok := otherResource.(*brokerless[T])
	if !ok {
		return resource.ErrWrongResourceImpl
	}

	other.linksLock.Lock()
	defer other.linksLock.Unlock()

	if _, ok := other.links[r]; ok {
		return resource.ErrWarnLinkExists
	}

	if r.linksTo(other) {
		return resource.ErrLinkLoop
	}

	other.links[r] = struct{}{}

	return nil
}

// UnLink implements resource.Resource.
func (r *brokerless[T]) UnLink(otherResource resource.Resource[T]) error {
	other, ok := otherResource.(*brokerless[T])
	if !ok {
		return resource.ErrWrongResourceImpl
	}

	other.linksLock.Lock()
	defer other.linksLock.Unlock()

	_, ok = other.links[r]
	if !ok {
		return resource.ErrLinkNotFound
	}

	delete(other.links, r)

	return nil
}

// checks whether a resource links to another resource (using depth first search)
func (r *brokerless[T]) linksTo(other *brokerless[T]) bool {
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
