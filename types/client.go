package types

import (
	"log"
	"sync"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/tinylib/msgp/msgp"
	"github.com/vmihailenco/msgpack"
)

var (
	clientTimeout              = config.GetDuration("WEBSOCKET_CONNECTION_TIMEOUT", 5*time.Second) // TODO: change default
	clientCheckTimeoutInterval = config.GetDuration("WEBSOCKET_CONNECTION_TIMEOUT_CHECKING_INTERVAL", time.Second)
)

// The Client type stores a Send function via which the server can send a Response to the client
// as well as a Streams map that stores the active stream channels for each resource path.
type Client struct {
	Send    func(*Response) error
	streams map[reid]map[path]chan any
	timeout *time.Timer

	lock     sync.RWMutex
	username string
	token    string
}

type reid string // using REID (msgp.Raw) which is a []byte converted to string as map key
type path string // using PATH ([]string) converted to msgpack []byte converted to string as map key

func NewClient(send func(r *Response) error) *Client {
	return &Client{
		Send:    send,
		streams: make(map[reid]map[path]chan any, 0),
		timeout: nil,
	}
}

func (c *Client) SetAuth(username, token string) {
	log.Println("Setting auth:", username, token)
	c.lock.RLock()
	defer c.lock.RUnlock()
	if c.username == username && c.token == token {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.username = username
	c.token = token
}

// check automatically because api token might expire
func (c *Client) EnableTimeout(onTimeout func()) {
	log.Println("Timeout enabled")
	c.timeout = time.AfterFunc(clientTimeout, onTimeout)
	go func() {
		ticker := time.NewTicker(clientCheckTimeoutInterval)
		for range ticker.C {
			c.lock.RLock()
			authenticated := false
			// TODO: check if c.username + c.token authenticated
			if authenticated {
				c.timeout.Reset(clientTimeout)
			}
			c.lock.RUnlock()
		}
	}()
}

func (c *Client) GetNumberOfStreams() int {
	return len(c.streams)
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
