package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/ProjectLighthouseCAU/beacon/resource/brokerless"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

func (handler *Handler) post(request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	resrc := brokerless.Create(request.PATH, request.PayloadToContent())
	err := handler.directory.CreateLeaf(request.PATH, resrc)
	if err == nil {
		response.Rnum(http.StatusCreated)
		return response.Build()
	}
	// creation failed (already exists or other error)
	response.Warning(err.Error()).Rnum(http.StatusOK)
	resrc, err = handler.directory.GetLeaf(request.PATH)
	if err != nil { // other error during creation
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}
	err = resrc.Put(request.PayloadToContent())
	if err != nil {
		return response.Warning(err.Error()).Rnum(resource.ErrorToStatusCode(err)).Build()
	}
	return response.Build()
}
