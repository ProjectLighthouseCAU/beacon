package types

import (
	"log"

	"github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/tinylib/msgp/msgp"
	"github.com/vmihailenco/msgpack"
)

// The Client type stores a Send function via which the server can send a Response to the client
// as well as a Streams map that stores the active stream channels for each resource path.
type Client struct {
	Send func(*Response) error

	streams map[reid]map[path]chan any
}

type reid string // using REID (msgp.Raw) which is a []byte converted to string as map key
type path string // using PATH ([]string) converted to msgpack []byte converted to string as map key

func NewClient(send func(*Response) error) *Client {
	return &Client{
		Send:    send,
		streams: make(map[reid]map[path]chan any, 0),
	}
}

func reidToMapKey(REID msgp.Raw) reid {
	return reid(REID)
}

func pathToMapKey(PATH []string) path {
	key, err := msgpack.Marshal(PATH)
	if err != nil {
		log.Println(err)
	}
	return path(key)
}

func pathFromMapKey(p path) []string {
	var PATH []string
	err := msgpack.Unmarshal([]byte(p), &PATH)
	if err != nil {
		log.Println(err)
	}
	return PATH
}

func (c *Client) AddStream(REID msgp.Raw, PATH []string, stream chan any) {
	reidKey := reidToMapKey(REID)
	_, ok := c.streams[reidKey]
	if !ok {
		c.streams[reidKey] = make(map[path]chan any)
	}
	c.streams[reidKey][pathToMapKey(PATH)] = stream
}

func (c *Client) GetStream(reid msgp.Raw, path []string) chan any {
	streams, ok := c.streams[reidToMapKey(reid)]
	if !ok {
		return nil
	}
	stream, ok := streams[pathToMapKey(path)]
	if !ok {
		return nil
	}
	return stream
}

func (c *Client) RemoveStream(reid msgp.Raw, path []string) {
	reidKey := reidToMapKey(reid)
	streams, ok := c.streams[reidKey]
	if !ok {
		return
	}
	pathKey := pathToMapKey(path)
	_, ok = streams[pathKey]
	if !ok {
		return
	}
	delete(streams, pathKey)
	if len(streams) == 0 {
		delete(c.streams, reidKey)
	}
}

func (c *Client) Disconnect(dir directory.Directory) {
	// Stop all streams of this client
	for _, streams := range c.streams {
		for path, stream := range streams {
			resource, err := dir.GetResource(pathFromMapKey(path))
			if err != nil {
				return
			}
			resource.StopStream(stream)
		}
	}
}
