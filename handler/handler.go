package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/auth"
	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

type Handler struct {
	directory directory.Directory
	auth      auth.Auth
}

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

func (handler *Handler) GetDirectory() directory.Directory {
	return handler.directory
}

func (handler *Handler) Close() {
	handler.directory.ForEach(func(res resource.Resource) (bool, error) {
		res.Close()
		return true, nil
	})
}

func (handler *Handler) HandleRequest(client *types.Client, request *types.Request) bool {
	defer func() { // recover from any panic while handling the request to prevent complete server crash
		if r := recover(); r != nil {
			log.Println("Recovering from panic in handler:", r)
			response := types.NewResponse().Reid(request.REID).Rnum(http.StatusInternalServerError).Warning(fmt.Sprint(r)).Build()
			client.Send(response)
		}
	}()

	// Authentication and Authorization
	if ok, code := handler.auth.IsAuthorized(client, request); !ok {
		response := types.NewResponse().Reid(request.REID).Rnum(code).Build()
		client.Send(response)
		return false
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
	default:
		response = types.NewResponse().Reid(request.REID).Rnum(http.StatusMethodNotAllowed).Build()
		client.Send(response)
		return false
	}

	if config.VerboseLogging {
		log.Printf("\nRequest: %+v\nResponse: %+v\n", request, response)
	}
	client.Send(response)
	return true
}
