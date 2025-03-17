package handler

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

type Handler struct {
	directory directory.Directory[resource.Resource[resource.Content]]
}

func New(dir directory.Directory[resource.Resource[resource.Content]]) *Handler {
	if dir == nil {
		panic("cannot create handler without directory (nil)")
	}
	return &Handler{
		directory: dir,
	}
}

func (handler *Handler) GetDirectory() directory.Directory[resource.Resource[resource.Content]] {
	return handler.directory
}

func (handler *Handler) Close() {
	handler.directory.ForEach([]string{}, func(path []string, res resource.Resource[resource.Content]) (bool, error) {
		res.Close()
		return true, nil
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

	// check if path contains "/" (must currently be enforced for snapshotting)
	for _, pathElement := range request.PATH {
		if strings.Contains(pathElement, "/") {
			warning := "path must not contain \"/\""
			response := types.NewResponse().Reid(request.REID).Rnum(http.StatusBadRequest).Warning(warning).Build()
			client.Send(response)
			return
		}
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
		return
	}

	if config.VerboseLogging {
		log.Printf("\nRequest: %+v\nResponse: %+v\n", request, response)
	}
	client.Send(response)
}
