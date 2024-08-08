package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/types"
	"github.com/tinylib/msgp/msgp"
)

func (handler *Handler) get(request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	resource, err := handler.directory.GetResource(request.PATH)
	if err != nil { // resource not found
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}
	payload, resp := resource.Get()
	if resp.Err != nil {
		response.Warning(resp.Err.Error())
	}
	return response.Rnum(resp.Code).Payload(payload.(msgp.Raw)).Build()
}
