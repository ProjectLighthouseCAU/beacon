package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/types"
)

func (handler *Handler) stop(client *types.Client, request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	resource, err := handler.directory.GetResource(request.PATH)
	if err != nil { // resource not found
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}
	stream := client.GetStream(request.PATH)
	if stream == nil {
		return response.Rnum(http.StatusNotFound).Warning("No open stream for this resource").Build()
	}
	resp := resource.StopStream(stream)
	if resp.Err != nil {
		response.Warning(resp.Err.Error())
	}
	client.RemoveStream(request.PATH)
	return response.Rnum(resp.Code).Build()
}
