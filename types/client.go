package types

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/tinylib/msgp/msgp"
	"github.com/vmihailenco/msgpack/v5"
)

// The Client type stores a Send function via which the server can send a Response to the client
// as well as a Streams map that stores the active stream channels for each resource path.
type Client struct {
	Send    func(*Response) error
	ip      string
	streams map[reid]map[path]chan resource.Content

	authCache                   map[string]*AuthCacheEntry
	authCacheLock               sync.RWMutex
	authCacheUpdaterCancelFuncs map[string]context.CancelFunc
}

type reid string // using REID (msgp.Raw) which is a []byte converted to string as map key
type path string // using PATH ([]string) converted to msgpack []byte converted to string as map key

type AuthCacheEntry struct {
	Token     string
	ExpiresAt time.Time
	Permanent bool
	Roles     []string
}

func NewClient(ip string, send func(*Response) error) *Client {
	return &Client{
		Send:                        send,
		ip:                          ip,
		streams:                     make(map[reid]map[path]chan resource.Content),
		authCache:                   make(map[string]*AuthCacheEntry),
		authCacheUpdaterCancelFuncs: make(map[string]context.CancelFunc),
	}
}

func (c *Client) Ip() string {
	return c.ip
}

// helpers

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

// streams

func (c *Client) AddStream(REID msgp.Raw, PATH []string, stream chan resource.Content) {
	reidKey := reidToMapKey(REID)
	_, ok := c.streams[reidKey]
	if !ok {
		c.streams[reidKey] = make(map[path]chan resource.Content)
	}
	c.streams[reidKey][pathToMapKey(PATH)] = stream
}

func (c *Client) GetStream(reid msgp.Raw, path []string) chan resource.Content {
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

// auth cache

func (c *Client) IsAuthCacheEmpty() bool {
	c.authCacheLock.RLock()
	defer c.authCacheLock.RUnlock()
	return len(c.authCache) == 0
}

func (c *Client) LookupAuthCache(username string) *AuthCacheEntry {
	c.authCacheLock.RLock()
	defer c.authCacheLock.RUnlock()
	return c.authCache[username]
}

func (c *Client) SetAuthCacheEntry(username string, entry *AuthCacheEntry) {
	c.authCacheLock.Lock()
	defer c.authCacheLock.Unlock()
	c.authCache[username] = entry
}

func (c *Client) DeleteAuthCacheEntry(username string) {
	c.authCacheLock.Lock()
	defer c.authCacheLock.Unlock()
	delete(c.authCache, username)
}

func (c *Client) AddAuthCacheUpdaterCancelFunc(username string, cancel context.CancelFunc) {
	c.authCacheLock.Lock()
	c.authCacheUpdaterCancelFuncs[username] = cancel
	c.authCacheLock.Unlock()
}

func (c *Client) RemoveAuthCacheUpdaterCancelFunc(username string) {
	c.authCacheLock.Lock()
	delete(c.authCacheUpdaterCancelFuncs, username)
	c.authCacheLock.Unlock()
}

func (c *Client) Disconnect(dir directory.Directory[resource.Resource[resource.Content]]) {
	// Stop all streams of this client
	for _, streams := range c.streams {
		for path, stream := range streams {
			resource, err := dir.GetLeaf(pathFromMapKey(path))
			if err != nil {
				continue
			}
			_ = resource.StopStream(stream)
		}
	}
	// Stop all cache updaters of this client
	c.authCacheLock.Lock()
	for _, cancel := range c.authCacheUpdaterCancelFuncs {
		cancel()
	}
	c.authCacheLock.Unlock()
}
