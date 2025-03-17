package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

func (handler *Handler) stop(client *types.Client, request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	resrc, err := handler.directory.GetLeaf(request.PATH)
	if err != nil { // resource not found
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}
	stream := client.GetStream(request.REID, request.PATH)
	if stream == nil {
		warning := fmt.Sprintf("No open stream for resource %s with REID %v", strings.Join(request.PATH, "/"), request.REID)
		return response.Rnum(http.StatusNotFound).Warning(warning).Build()
	}
	err = resrc.StopStream(stream)
	if err != nil {
		response.Warning(err.Error())
	}
	client.RemoveStream(request.REID, request.PATH)
	return response.Rnum(resource.ErrorToStatusCode(err)).Build()
}
