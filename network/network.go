package network

import (
	"github.com/ProjectLighthouseCAU/beacon/auth"
	"github.com/ProjectLighthouseCAU/beacon/handler"
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
	Type    EndpointType
	Handler *handler.Handler
	Auth    auth.Auth
}

// Endpoint is the interface which a specific endpoint has to implement
type Endpoint interface {
	// TODO: unified interface between websocket, tcp, unix domain, etc.
	Close()
}
