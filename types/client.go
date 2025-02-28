package types

import (
	"log"
	"sync"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/config"
)

var (
	clientTimeout              = config.GetDuration("WEBSOCKET_CONNECTION_TIMEOUT", 5*time.Second) // TODO: change default
	clientCheckTimeoutInterval = config.GetDuration("WEBSOCKET_CONNECTION_TIMEOUT_CHECKING_INTERVAL", time.Second)
)

// The Client type stores a Send function via which the server can send a Response to the client
// as well as a Streams map that stores the active stream channels for each resource path.
type Client struct {
	Send    func(*Response) error
	streams []stream
	timeout *time.Timer

	lock     sync.RWMutex
	username string
	token    string
}

type stream struct {
	path    []string
	channel chan interface{}
}

func NewClient(send func(r *Response) error) *Client {
	return &Client{
		Send:    send,
		streams: make([]stream, 0),
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

func (c *Client) AddStream(path []string, ch chan interface{}) {
	c.streams = append(c.streams, stream{
		path:    path,
		channel: ch,
	})
}

func (c *Client) GetStream(path []string) chan interface{} {
	for _, s := range c.streams {
		if equals(s.path, path) {
			return s.channel
		}
	}
	return nil
}

func (c *Client) RemoveStream(path []string) {
	idx := -1
	for i := 0; i < len(c.streams); i++ {
		if equals(c.streams[i].path, path) {
			idx = i
			break
		}
	}
	if idx == -1 {
		return
	}
	// delete from slice
	c.streams[idx] = c.streams[len(c.streams)-1]
	c.streams[len(c.streams)-1] = stream{}
	c.streams = c.streams[:len(c.streams)-1]
}

func (c *Client) ForEachStream(f func(path []string, ch chan interface{})) {
	for _, s := range c.streams {
		f(s.path, s.channel)
	}
}

func equals(s1 []string, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}
