package broker

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/tinylib/msgp/msgp"
)

type controlMsgType uint16

const (
	CLOSE controlMsgType = iota
	STREAM
	STOP
	LINK
	UNLINK
)

var (
	inputChanSize   = config.GetInt("RESOURCE_PUT_CHANNEL_SIZE", 1000)
	streamChanSize  = config.GetInt("RESOURCE_STREAM_CHANNEL_SIZE", 1000)
	controlChanSize = config.GetInt("RESOURCE_CONTROL_CHANNEL_SIZE", 100)
)

type broker struct {
	path []string // resource path

	input   chan inputMsg                // input channel (only for PUT)
	control chan controlMsg              // control channel (for everything else than PUT)
	streams map[chan interface{}]bool    // keeps track of active subscriber streams (value indicates whether the channel is infinite->blocking-send or finite->non-blocking-send)
	links   map[*broker]chan interface{} // keeps track of active links from other resources

	value     interface{} // latest input value
	valueLock sync.RWMutex
}

var _ resource.Resource = (*broker)(nil) // ensure resource implements Resource

// Message sent through input channel
type inputMsg struct { // PUT
	Content      interface{}
	ResponseChan chan resource.Response
}

// Message sent through control channel
type controlMsg struct {
	Type         controlMsgType // enum - see const
	Content      interface{}
	ResponseChan chan resource.Response
}

// Content for STREAM controlMsg
type streamContent struct {
	channel  chan interface{}
	infinite bool
}

// Create creates and returns a new resource and starts a goroutine which acts as a message broker
// between the put channel and the subscribed stream channels as well the value of the resource.
func Create(path []string) resource.Resource {
	// create resource and initialize channels, maps and mutex
	r := &broker{
		path: path,

		input:   make(chan inputMsg, inputChanSize),
		control: make(chan controlMsg, controlChanSize),

		streams: make(map[chan interface{}]bool),
		links:   make(map[*broker]chan interface{}),

		value:     msgp.Raw{},
		valueLock: sync.RWMutex{},
	}
	go r.broker()
	// go r.monitor()
	return r
}

//lint:ignore U1000 unused
func (r *broker) monitor() {
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

func nonBlockingSend(c chan interface{}, v interface{}) bool {
	select {
	case c <- v:
		return true
	default:
		return false
	}
}

func (r *broker) broker() {
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
			for stream, infinite := range r.streams {
				if infinite {
					stream <- payload // blocking send on infinite channel won't block
				} else {
					sent := nonBlockingSend(stream, payload) // non-blocking send for finite channels
					if !sent {
						log.Println("[Warning] A stream channel is full and was skipped by the broker") // TODO: add prometheus metric "dropped_stream_packets"
					}
				}
			}
			inputMsg.ResponseChan <- resource.Response{Code: 200, Err: nil}

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
				controlMsg.ResponseChan <- resource.Response{Code: 200, Err: nil}
				return

			case STREAM:
				stream := controlMsg.Content.(streamContent)
				r.streams[stream.channel] = stream.infinite
				controlMsg.ResponseChan <- resource.Response{Code: 200, Err: nil}

			case STOP:
				stream := controlMsg.Content.(chan interface{})
				_, ok := r.streams[stream]
				if !ok {
					controlMsg.ResponseChan <- resource.Response{Code: 404, Err: errors.New("the stream does not exist and therefore cannot be closed")}
					break
				}
				close(stream)
				delete(r.streams, stream)
				controlMsg.ResponseChan <- resource.Response{Code: 200, Err: nil}

			case LINK:
				otherResource := controlMsg.Content.(*broker)
				if _, ok := r.links[otherResource]; ok {
					controlMsg.ResponseChan <- resource.Response{Code: 200, Err: errors.New("the link already exists")}
					break
				}
				if r.isLinkedBy(otherResource) {
					controlMsg.ResponseChan <- resource.Response{Code: 508, Err: errors.New("the link causes a loop in the linking graph which is not allowed")}
					break
				}
				stream, _ := otherResource.Stream()
				go func() { // forward data from other resources stream to this resources input
					for payload := range stream {
						r.Put(payload)
					}
				}()
				r.links[otherResource] = stream
				controlMsg.ResponseChan <- resource.Response{Code: 200, Err: nil}

			case UNLINK:
				otherResource := controlMsg.Content.(*broker)
				stream, ok := r.links[otherResource]
				if !ok {
					controlMsg.ResponseChan <- resource.Response{Code: 404, Err: errors.New("the link does not exist and therefore cannot be removed")}
					break
				}
				otherResource.StopStream(stream)
				delete(r.links, otherResource)
				controlMsg.ResponseChan <- resource.Response{Code: 200, Err: nil}
			}
		}
	}
}

// Close this resources broker and all active streams. A Get request is still possible after the resource was closed.
func (r *broker) Close() resource.Response {
	respChan := make(chan resource.Response)
	defer close(respChan)
	r.control <- controlMsg{Type: CLOSE, Content: nil, ResponseChan: respChan}
	return <-respChan
}

// Stream subscribes to this resource.
// This returns a new channel where all updates to this resource will be sent to.
func (r *broker) Stream() (chan interface{}, resource.Response) {
	stream := make(chan interface{}, streamChanSize)
	respChan := make(chan resource.Response)
	defer close(respChan)
	r.control <- controlMsg{Type: STREAM, Content: streamContent{stream, false}, ResponseChan: respChan}
	return stream, <-respChan
}

// TODO: change interface sucht that this method can be used
func (r *broker) StreamWithGuaranteedDelivery() (chan interface{}, resource.Response) {
	streamIn, streamOut := makeInfinite()
	respChan := make(chan resource.Response)
	defer close(respChan)
	r.control <- controlMsg{Type: STREAM, Content: streamContent{streamIn, true}, ResponseChan: respChan}
	return streamOut, <-respChan
}

// StopStream unsubscribes from this resource.
// The channel created by Stream() needs to be passed
func (r *broker) StopStream(stream chan interface{}) resource.Response {
	respChan := make(chan resource.Response)
	defer close(respChan)
	r.control <- controlMsg{Type: STOP, Content: stream, ResponseChan: respChan}
	return <-respChan
}

// Put updates the value of this resource.
func (r *broker) Put(payload interface{}) resource.Response {
	respChan := make(chan resource.Response)
	defer close(respChan)
	r.input <- inputMsg{Content: payload, ResponseChan: respChan}
	return <-respChan
}

// Get returns the current (latest written) value of this resource
func (r *broker) Get() (interface{}, resource.Response) {
	r.valueLock.RLock()
	defer r.valueLock.RUnlock()
	return r.value, resource.Response{Code: 200, Err: nil}
}

// Link links one resources input to another resources output.
// The link fails if it causes a loop in the linking graph.
func (r *broker) Link(other resource.Resource) resource.Response {
	respChan := make(chan resource.Response)
	defer close(respChan)
	r.control <- controlMsg{Type: LINK, Content: other, ResponseChan: respChan}
	return <-respChan
}

// UnLink removes the link created by Link
func (r *broker) UnLink(other resource.Resource) resource.Response {
	respChan := make(chan resource.Response)
	defer close(respChan)
	r.control <- controlMsg{Type: UNLINK, Content: other, ResponseChan: respChan}
	return <-respChan
}

// checks whether a given resource links to this resource (using depth first search)
func (r *broker) isLinkedBy(other *broker) bool {
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
func makeInfinite() (in chan interface{}, out chan interface{}) {
	in = make(chan interface{}, streamChanSize)
	out = make(chan interface{}, streamChanSize)
	go func() {
		var inQ []interface{}
		outC := func() chan interface{} {
			if len(inQ) == 0 {
				return nil
			}
			return out
		}
		curVal := func() interface{} {
			if len(inQ) == 0 {
				return nil
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
