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
	"github.com/tinylib/msgp/msgp"
	"github.com/vmihailenco/msgpack"
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

	// create response
	response := types.NewResponse()
	response.Reid(request.REID)

	// create resource in case of POST
	switch request.VERB {
	case "POST": // POST = CREATE + PUT
		err := handler.directory.CreateResource(request.PATH)
		if err != nil { // creation failed (already exists or other error)
			response.Warning(err.Error()).Rnum(http.StatusOK)
		} else {
			response.Rnum(http.StatusCreated)
		}
		resource, err := handler.directory.GetResource(request.PATH)
		if err != nil { // other error during creation
			response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
			client.Send(response)
			return
		}
		resp := resource.Put(request.PAYL)
		if resp.Err != nil {
			response.Warning(resp.Err.Error()).Rnum(resp.Code).Build()
			client.Send(response)
			return
		}
		response.Build()
		client.Send(response)
		return

	case "CREATE":
		err := handler.directory.CreateResource(request.PATH)
		if err != nil {
			response.Warning(err.Error()).Rnum(http.StatusBadRequest).Build()
			client.Send(response)
			return
		}
		response.Rnum(http.StatusCreated).Build()
		client.Send(response)
		return

	case "MKDIR":
		err := handler.directory.CreateDirectory(request.PATH)
		if err != nil {
			response.Warning(err.Error()).Rnum(http.StatusBadRequest).Build()
			client.Send(response)
			return
		}
		response.Rnum(http.StatusCreated).Build()
		client.Send(response)
		return

	case "DELETE":
		err := handler.directory.Delete(request.PATH)
		if err != nil {
			response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
			client.Send(response)
			return
		}
		response.Rnum(http.StatusOK).Build()
		client.Send(response)
		return

	case "LIST":
		// TODO: return also string representation of the directory when request contains a META tag
		lst, err := handler.directory.List(request.PATH)
		if err != nil {
			response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
			client.Send(response)
			return
		}
		payl, err := msgpack.Marshal(lst)
		if err != nil {
			response.Warning(err.Error()).Rnum(http.StatusInternalServerError).Build()
			client.Send(response)
			return
		}
		response.Rnum(http.StatusOK).Payload(payl).Build()
		client.Send(response)
		return
	}

	resource, err := handler.directory.GetResource(request.PATH)
	if err != nil { // resource not found
		response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
		client.Send(response)
		return
	}

	switch request.VERB {
	case "GET":
		payload, resp := resource.Get()
		if resp.Err != nil {
			response.Warning(resp.Err.Error())
		}
		response.Rnum(resp.Code).Payload(payload.(msgp.Raw))

	case "PUT":
		resp := resource.Put(request.PAYL)
		if resp.Err != nil {
			response.Warning(resp.Err.Error())
		}
		response.Rnum(resp.Code)

	case "STREAM":
		if client.GetStream(request.PATH) != nil { // stream already exists
			// don't open another stream, only return resource content
			payload, resp := resource.Get()
			if resp.Err != nil {
				response.Warning(resp.Err.Error())
			}
			response.Warning("Already streaming this resource")
			response.Rnum(resp.Code).Payload(payload.(msgp.Raw))
			break
		}
		// create stream channel and add it to the client
		stream, resp := resource.Stream()
		if resp.Err != nil {
			response.Rnum(resp.Code).Warning(resp.Err.Error())
			break
		}
		client.AddStream(request.PATH, stream)
		// start goroutine for sending updates
		go func() {
			streamResponse := types.NewResponse().Reid(request.REID).Rnum(http.StatusOK)
			for payload := range stream {
				streamResponse.Payload(payload.(msgp.Raw)).Build()
				err := client.Send(streamResponse)
				if err != nil { // client closed
					resource.StopStream(stream)
					return
				}
			}
		}()
		// return resource content
		payload, resp := resource.Get()
		if resp.Err != nil {
			response.Warning(resp.Err.Error())
		}
		response.Rnum(resp.Code).Payload(payload.(msgp.Raw))

	case "STOP":
		stream := client.GetStream(request.PATH)
		if stream == nil {
			response.Rnum(http.StatusNotFound).Warning("No open stream for this resource")
			break
		}
		resp := resource.StopStream(stream)
		if resp.Err != nil {
			response.Warning(resp.Err.Error())
		}
		client.RemoveStream(request.PATH)
		response.Rnum(resp.Code)

	case "LINK": // destination: PATH, source: PAYL
		sourcePath, err := request.PayloadToPath()
		if err != nil {
			response.Rnum(http.StatusBadRequest).Warning(err.Error())
			break
		}
		source, err := handler.directory.GetResource(sourcePath)
		if err != nil {
			response.Rnum(http.StatusNotFound).Warning(err.Error())
			break
		}
		resp := resource.Link(source)
		if resp.Err != nil {
			response.Warning(resp.Err.Error())
		}
		response.Rnum(resp.Code)

	case "UNLINK":
		sourcePath, err := request.PayloadToPath()
		if err != nil {
			response.Rnum(http.StatusBadRequest).Warning(err.Error())
			break
		}
		source, err := handler.directory.GetResource(sourcePath)
		if err != nil {
			response.Rnum(http.StatusNotFound).Warning(err.Error())
			break
		}
		resp := resource.UnLink(source)
		if resp.Err != nil {
			response.Warning(resp.Err.Error())
		}
		response.Rnum(resp.Code)

	}

	response.Build() // checks response and fills missing fields if possible
	if verbose {
		fmt.Printf("Response: %+v", response)
	}
	client.Send(response)
}
