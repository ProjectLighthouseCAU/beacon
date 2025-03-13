package broker

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/resource"
)

type controlMsgType uint16

const (
	CLOSE controlMsgType = iota
	STREAM
	STOP
	LINK
	UNLINK
)

type broker[T any] struct {
	path []string // resource path

	input   chan inputMsg[T]      // input channel (only for PUT)
	control chan controlMsg[T]    // control channel (for everything else than PUT)
	streams map[chan T]bool       // keeps track of active subscriber streams (value indicates whether the channel is infinite->blocking-send or finite->non-blocking-send)
	links   map[*broker[T]]chan T // keeps track of active links from other resources

	value     T // latest input value
	valueLock sync.RWMutex
}

var _ resource.Resource[resource.Content] = (*broker[resource.Content])(nil) // ensure resource implements Resource

// Response struct for detailed response to the server
type response struct {
	Code int
	Err  error
}

// Message sent through input channel
type inputMsg[T any] struct { // PUT
	Content      T
	ResponseChan chan response
}

// Message sent through control channel
type controlMsg[T any] struct {
	Type         controlMsgType // enum - see const
	Content      any
	ResponseChan chan response
}

// Content for STREAM controlMsg
type streamContent[T any] struct {
	channel  chan T
	infinite bool
}

// Create creates and returns a new resource and starts a goroutine which acts as a message broker
// between the put channel and the subscribed stream channels as well the value of the resource.
func Create[T any](path []string, initialValue T) resource.Resource[T] {
	// create resource and initialize channels, maps and mutex
	r := &broker[T]{
		path: path,

		input:   make(chan inputMsg[T], config.ResourceInputChannelSize),
		control: make(chan controlMsg[T], config.ResourceControlChannelSize),

		streams: make(map[chan T]bool),
		links:   make(map[*broker[T]]chan T),

		value:     initialValue,
		valueLock: sync.RWMutex{},
	}
	go r.broker()
	// go r.monitor()
	return r
}

//lint:ignore U1000 unused
func (r *broker[T]) monitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		path := strings.Join(r.path, "/")
		log.Printf("["+path+"] PUT: %d\n", len(r.input))
		i := 0
		for c := range r.streams {
			log.Printf("["+path+"] STREAM #"+fmt.Sprint(i)+": %d\n", len(c))
			i++
		}

	}
}

func nonBlockingSend[T any](c chan T, v T) bool {
	select {
	case c <- v:
		return true
	default:
		return false
	}
}

func (r *broker[T]) broker() {
	log.Println("Resource " + strings.Join(r.path, "/") + " started")
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovering from panic in resource broker", r)
		}
		log.Println("Resource " + strings.Join(r.path, "/") + " closed")
	}()
	for {
		select {
		case inputMsg := <-r.input: // input message (PUT)
			payload := inputMsg.Content
			r.valueLock.Lock()
			r.value = payload
			r.valueLock.Unlock()
			// send new value to all subscribed streams
			anyStreamSkipped := false
			for stream, infinite := range r.streams {
				if infinite {
					stream <- payload // blocking send on infinite channel won't block
				} else {
					sent := nonBlockingSend(stream, payload) // non-blocking send for finite channels
					if !sent {
						anyStreamSkipped = true
						log.Println("[Warning] A stream channel is full and was skipped by the broker") // TODO: add prometheus metric "dropped_stream_packets"
					}
				}
			}
			if anyStreamSkipped {
				inputMsg.ResponseChan <- response{Code: 200, Err: resource.ErrWarnStreamSkipped}
			} else {
				inputMsg.ResponseChan <- response{Code: 200, Err: nil}
			}

		case controlMsg := <-r.control: // control message (CLOSE, STREAM, STOP, LINK, UNLINK)
			switch controlMsg.Type {
			case CLOSE:
				// close all active streams before closing the resource
				for stream := range r.streams {
					close(stream)
					delete(r.streams, stream)
				}
				for other, stream := range r.links {
					other.StopStream(stream)
					delete(r.links, other)
				}
				controlMsg.ResponseChan <- response{Code: 200, Err: nil}
				return

			case STREAM:
				stream := controlMsg.Content.(streamContent[T])
				r.streams[stream.channel] = stream.infinite
				controlMsg.ResponseChan <- response{Code: 200, Err: nil}

			case STOP:
				stream := controlMsg.Content.(chan T)
				_, ok := r.streams[stream]
				if !ok {
					controlMsg.ResponseChan <- response{Code: 404, Err: resource.ErrStreamNotFound}
					break
				}
				close(stream)
				delete(r.streams, stream)
				controlMsg.ResponseChan <- response{Code: 200, Err: nil}

			case LINK:
				otherResource := controlMsg.Content.(*broker[T])
				if _, ok := r.links[otherResource]; ok {
					controlMsg.ResponseChan <- response{Code: 200, Err: resource.ErrWarnLinkExists}
					break
				}
				if r.isLinkedBy(otherResource) {
					controlMsg.ResponseChan <- response{Code: 508, Err: resource.ErrLinkLoop}
					break
				}
				stream := otherResource.Stream()
				go func() { // forward data from other resources stream to this resources input
					for payload := range stream {
						r.Put(payload)
					}
				}()
				r.links[otherResource] = stream
				controlMsg.ResponseChan <- response{Code: 200, Err: nil}

			case UNLINK:
				otherResource := controlMsg.Content.(*broker[T])
				stream, ok := r.links[otherResource]
				if !ok {
					controlMsg.ResponseChan <- response{Code: 404, Err: resource.ErrLinkNotFound}
					break
				}
				otherResource.StopStream(stream)
				delete(r.links, otherResource)
				controlMsg.ResponseChan <- response{Code: 200, Err: nil}
			}
		}
	}
}

// Close this resources broker and all active streams. A Get request is still possible after the resource was closed.
func (r *broker[T]) Close() {
	respChan := make(chan response)
	defer close(respChan)
	r.control <- controlMsg[T]{Type: CLOSE, Content: nil, ResponseChan: respChan}
	<-respChan
}

// Stream subscribes to this resource.
// This returns a new channel where all updates to this resource will be sent to.
func (r *broker[T]) Stream() chan T {
	stream := make(chan T, config.ResourceStreamChannelSize)
	respChan := make(chan response)
	defer close(respChan)
	r.control <- controlMsg[T]{Type: STREAM, Content: streamContent[T]{stream, false}, ResponseChan: respChan}
	<-respChan
	return stream
}

// TODO: change interface such that this method can be used
func (r *broker[T]) StreamWithGuaranteedDelivery() chan T {
	streamIn, streamOut := makeInfinite[T]()
	respChan := make(chan response)
	defer close(respChan)
	r.control <- controlMsg[T]{Type: STREAM, Content: streamContent[T]{streamIn, true}, ResponseChan: respChan}
	<-respChan
	return streamOut
}

// StopStream unsubscribes from this resource.
// The channel created by Stream() needs to be passed
func (r *broker[T]) StopStream(stream chan T) error {
	respChan := make(chan response)
	defer close(respChan)
	r.control <- controlMsg[T]{Type: STOP, Content: stream, ResponseChan: respChan}
	resp := <-respChan
	return resp.Err
}

// Put updates the value of this resource.
func (r *broker[T]) Put(payload T) error {
	respChan := make(chan response)
	defer close(respChan)
	r.input <- inputMsg[T]{Content: payload, ResponseChan: respChan}
	resp := <-respChan
	return resp.Err
}

// Get returns the current (latest written) value of this resource
func (r *broker[T]) Get() T {
	r.valueLock.RLock()
	defer r.valueLock.RUnlock()
	return r.value
}

// Link links one resources input to another resources output.
// The link fails if it causes a loop in the linking graph.
func (r *broker[T]) Link(other resource.Resource[T]) error {
	respChan := make(chan response)
	defer close(respChan)
	r.control <- controlMsg[T]{Type: LINK, Content: other, ResponseChan: respChan}
	resp := <-respChan
	return resp.Err
}

// UnLink removes the link created by Link
func (r *broker[T]) UnLink(other resource.Resource[T]) error {
	respChan := make(chan response)
	defer close(respChan)
	r.control <- controlMsg[T]{Type: UNLINK, Content: other, ResponseChan: respChan}
	resp := <-respChan
	return resp.Err
}

// checks whether a given resource links to this resource (using depth first search)
func (r *broker[T]) isLinkedBy(other *broker[T]) bool {
	// if the resources are the same, they are considered linked
	if other == r {
		return true
	}
	// if the other resource has no links, it cannot link to the resource
	if len(other.links) == 0 {
		return false
	}
	// check if any of the other resources links does link to the resource (transitive linking)
	for res := range other.links {
		if r.isLinkedBy(res) {
			return true
		}
	}
	return false
}

// Makes an infinite channel by using a slice and a goroutine
func makeInfinite[T any]() (in chan T, out chan T) {
	in = make(chan T, config.ResourceStreamChannelSize)
	out = make(chan T, config.ResourceStreamChannelSize)
	go func() {
		var inQ []T
		outC := func() chan T {
			if len(inQ) == 0 {
				return nil
			}
			return out
		}
		curVal := func() T {
			if len(inQ) == 0 {
				var zeroValue T
				return zeroValue
			}
			return inQ[0]
		}
		for len(inQ) > 0 || in != nil {
			select {
			case v, ok := <-in:
				if !ok {
					in = nil
				} else {
					inQ = append(inQ, v)
				}
			case outC() <- curVal():
				inQ = inQ[1:]
			}
		}
		close(out)
	}()
	return in, out
}
