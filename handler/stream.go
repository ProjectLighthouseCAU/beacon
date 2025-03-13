package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ProjectLighthouseCAU/beacon/types"
)

func (handler *Handler) stream(client *types.Client, request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	resource, err := handler.directory.GetLeaf(request.PATH)
	if err != nil { // resource not found
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}

	// stream with this REID on this PATH already exists
	if s := client.GetStream(request.REID, request.PATH); s != nil {
		// don't open another stream, only return resource content
		payload := resource.Get()
		response.Warning(fmt.Sprintf("Already streaming %s", strings.Join(request.PATH, "/")))
		return response.Rnum(http.StatusOK).Payload(payload).Build()
	}

	// create stream channel and add it to the client
	stream := resource.Stream()
	client.AddStream(request.REID, request.PATH, stream)
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
	payload := resource.Get()
	return response.Rnum(http.StatusOK).Payload(payload).Build()
}
