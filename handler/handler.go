package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/auth"
	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/ProjectLighthouseCAU/beacon/network"
	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

var (
	verbose = config.GetBool("VERBOSE_LOGGING", false)
)

type Handler struct {
	directory directory.Directory
	auth      auth.Auth
}

// Handler implements network.RequestHandler
var _ network.RequestHandler = (*Handler)(nil)

func New(dir directory.Directory, a auth.Auth) *Handler {
	if dir == nil {
		panic("cannot create handler without directory (nil)")
	}
	if a == nil {
		panic("cannot create handler without auth (nil)")
	}
	return &Handler{
		directory: dir,
		auth:      a,
	}
}

func (handler *Handler) Close() {
	handler.directory.ForEach(func(res resource.Resource) (bool, error) {
		res.Close()
		return true, nil
	})
}

func (handler *Handler) Disconnect(client *types.Client) {
	client.ForEachStream(func(path []string, ch chan interface{}) {
		resource, err := handler.directory.GetResource(path)
		if err != nil {
			return
		}
		resource.StopStream(ch)
	})
}

func (handler *Handler) HandleRequest(client *types.Client, request *types.Request) {
	defer func() { // recover from any panic while handling the request to prevent complete server crash
		if r := recover(); r != nil {
			log.Println("Recovering from panic in handler:", r)
			response := types.NewResponse().Reid(request.REID).Rnum(http.StatusInternalServerError).Warning(fmt.Sprint(r)).Build()
			client.Send(response)
		}
	}()
	if verbose {
		fmt.Printf("Request: %+v\n", request)
	}
	// Authentication and Authorization
	if ok, code := handler.auth.IsAuthorized(client, request); !ok {
		response := types.NewResponse().Reid(request.REID).Rnum(code).Build()
		client.Send(response)
		return
	}

	var response *types.Response

	// create resource in case of POST
	switch request.VERB {
	case "POST": // POST = CREATE + PUT
		response = handler.post(request)
	case "CREATE":
		response = handler.create(request)
	case "MKDIR":
		response = handler.mkdir(request)
	case "DELETE":
		response = handler.delete(request)
	case "LIST":
		response = handler.list(request)
	case "GET":
		response = handler.get(request)
	case "PUT":
		response = handler.put(request)
	case "STREAM":
		response = handler.stream(client, request)
	case "STOP":
		response = handler.stop(client, request)
	case "LINK": // destination: PATH, source: PAYL
		response = handler.link(request)
	case "UNLINK":
		response = handler.unlink(request)
	}

	if verbose {
		fmt.Printf("Response: %+v", response)
	}
	client.Send(response)
}
