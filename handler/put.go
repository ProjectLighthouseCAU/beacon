package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

func (handler *Handler) put(request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	resrc, err := handler.directory.GetLeaf(request.PATH)
	if err != nil { // resource not found
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}
	err = resrc.Put(request.PayloadToContent())
	if err != nil {
		response.Warning(err.Error())
	}
	return response.Rnum(resource.ErrorToStatusCode(err)).Build()
}
