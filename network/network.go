package network

import (
	"github.com/ProjectLighthouseCAU/beacon/types"
)

type EndpointType uint16 // Enum
const (
	Websocket EndpointType = iota // Enum Fields
	TCP
	UDP
	UNIX_DOMAIN
	// ...
)

// BaseEndpoint contains fields that are shared between all Endpoint implementations
type BaseEndpoint struct {
	// Serializer serialization.Serializer
	Type     EndpointType
	Handlers []RequestHandler
}

// Endpoint is the interface which a specific endpoint has to implement
type Endpoint interface {
	// TODO: unified interface between websocket, tcp, unix domain, etc.
	Close()
}

// RequestHandler is an interface for a handler that registers itself at the endpoints
type RequestHandler interface {
	HandleRequest(*types.Client, *types.Request, *types.Response)
	Disconnect(*types.Client)
	Close()
}
