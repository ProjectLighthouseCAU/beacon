package handler

import (
	"fmt"
	"log"
	"net/http"

	"lighthouse.uni-kiel.de/lighthouse-server/auth"
	"lighthouse.uni-kiel.de/lighthouse-server/directory"
	"lighthouse.uni-kiel.de/lighthouse-server/directory/tree"
	"lighthouse.uni-kiel.de/lighthouse-server/network"
	"lighthouse.uni-kiel.de/lighthouse-server/types"
)

type Handler struct {
	directory directory.Directory
	auth      auth.Auth
}

// Handler implements network.RequestHandler
var _ network.RequestHandler = (*Handler)(nil)

func New(dir directory.Directory, a auth.Auth) *Handler {
	if dir == nil {
		dir = tree.NewTree()
	}
	if a == nil {
		a = &auth.AllowNone{}
	}
	return &Handler{
		directory: dir,
		auth:      a,
	}
}

func (handler *Handler) Close() {
	// TODO: foreach resource in directory: close resource
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
	// fmt.Printf("Request: %+v\n", request)

	// Authentication and Authorization
	if ok, code := handler.auth.IsAuthorized(client, request); !ok {
		response := types.NewResponse().Reid(request.REID).Rnum(code).Build()
		client.Send(response)
		return
	}

	// create response
	response := types.NewResponse().Reid(request.REID)

	// create resource in case of POST
	switch request.VERB {
	case "POST": // POST = CREATE + PUT
		err := handler.directory.CreateResource(request.PATH)
		if err != nil { // creation failed (already exists or other error)
			response.Warning(err.Error())
		}
		resource, err := handler.directory.GetResource(request.PATH)
		if err != nil { // other error during creation
			response.Warning(err.Error()).Rnum(http.StatusBadRequest).Build()
			client.Send(response)
			return
		}
		resp := resource.Put(request.PAYL)
		if resp.Err != nil {
			response.Warning(resp.Err.Error()).Rnum(resp.Code).Build()
			client.Send(response)
			return
		}
		response.Rnum(http.StatusCreated).Build()
		client.Send(response)
		return

	case "CREATE":
		err := handler.directory.CreateResource(request.PATH)
		if err != nil {
			response.Warning(err.Error()).Rnum(http.StatusBadRequest).Build()
			client.Send(response)
			return
		}

	case "DELETE":
		err := handler.directory.DeleteResource(request.PATH)
		if err != nil {
			response.Warning(err.Error()).Rnum(http.StatusBadRequest).Build()
			client.Send(response)
			return
		}
		response.Rnum(http.StatusOK).Build()
		client.Send(response)
		return

	case "LIST":
		// TODO: return nested maps instead of string representation
		// or both by using META?
		// also don't return the whole tree but rather the subtree from the request path
		str := handler.directory.String(request.PATH)
		response.Warning("This request method is work in progress and might change")
		response.Rnum(http.StatusOK).Payload(str).Build()
		client.Send(response)
		return
	}

	resource, err := handler.directory.GetResource(request.PATH)
	if err != nil { // resource not found
		response.Warning(err.Error()).Rnum(http.StatusBadRequest).Build()
		client.Send(response)
		return
	}

	switch request.VERB {
	case "GET":
		payload, resp := resource.Get()
		if resp.Err != nil {
			response.Warning(resp.Err.Error())
		}
		response.Rnum(resp.Code).Payload(payload)

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
			response.Rnum(resp.Code).Payload(payload)
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
				streamResponse.Payload(payload).Build()
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
		response.Rnum(resp.Code).Payload(payload)

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
			response.Rnum(http.StatusBadRequest).Warning(err.Error())
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
			response.Rnum(http.StatusBadRequest).Warning(err.Error())
			break
		}
		resp := resource.UnLink(source)
		if resp.Err != nil {
			response.Warning(resp.Err.Error())
		}
		response.Rnum(resp.Code)

	}

	response.Build() // checks response and fills missing fields if possible

	// fmt.Printf("Response: %+v", response)
	client.Send(response)
}
