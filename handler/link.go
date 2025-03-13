package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

func (handler *Handler) link(request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	resrc, err := handler.directory.GetLeaf(request.PATH)
	if err != nil { // resource not found
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}
	sourcePath, err := request.PayloadToPath()
	if err != nil {
		return response.Rnum(http.StatusBadRequest).Warning(err.Error()).Build()
	}
	source, err := handler.directory.GetLeaf(sourcePath)
	if err != nil {
		return response.Rnum(http.StatusNotFound).Warning(err.Error()).Build()
	}
	err = resrc.Link(source)
	if err != nil {
		response.Warning(err.Error())
	}
	return response.Rnum(resource.ErrorToStatusCode(err)).Build()
}
