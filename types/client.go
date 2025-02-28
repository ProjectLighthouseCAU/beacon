package types

// The Client type stores a Send function via which the server can send a Response to the client
// as well as a Streams map that stores the active stream channels for each resource path.
type Client struct {
	Send    func(*Response) error
	streams []stream
}

type stream struct {
	path    []string
	channel chan any
}

func NewClient(send func(*Response) error) *Client {
	return &Client{
		Send:    send,
		streams: make([]stream, 0),
	}
}

func (c *Client) AddStream(path []string, ch chan any) {
	c.streams = append(c.streams, stream{
		path:    path,
		channel: ch,
	})
}

func (c *Client) GetStream(path []string) chan any {
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

func (c *Client) ForEachStream(f func(path []string, ch chan any)) {
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
