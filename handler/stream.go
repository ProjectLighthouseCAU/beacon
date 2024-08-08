package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/types"
	"github.com/tinylib/msgp/msgp"
)

func (handler *Handler) stream(client *types.Client, request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	resource, err := handler.directory.GetResource(request.PATH)
	if err != nil { // resource not found
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}

	if client.GetStream(request.PATH) != nil { // stream already exists
		// don't open another stream, only return resource content
		payload, resp := resource.Get()
		if resp.Err != nil {
			response.Warning(resp.Err.Error())
		}
		response.Warning("Already streaming this resource")
		return response.Rnum(resp.Code).Payload(payload.(msgp.Raw)).Build()
	}
	// create stream channel and add it to the client
	stream, resp := resource.Stream()
	if resp.Err != nil {
		return response.Rnum(resp.Code).Warning(resp.Err.Error()).Build()
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
	return response.Rnum(resp.Code).Payload(payload.(msgp.Raw)).Build()
}
