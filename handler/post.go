package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/types"
)

func (handler *Handler) post(request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	err := handler.directory.CreateResource(request.PATH)
	if err != nil { // creation failed (already exists or other error)
		response.Warning(err.Error()).Rnum(http.StatusOK)
	} else {
		response.Rnum(http.StatusCreated)
	}
	resource, err := handler.directory.GetResource(request.PATH)
	if err != nil { // other error during creation
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}
	resp := resource.Put(request.PAYL)
	if resp.Err != nil {
		return response.Warning(resp.Err.Error()).Rnum(resp.Code).Build()
	}
	return response.Build()
}
