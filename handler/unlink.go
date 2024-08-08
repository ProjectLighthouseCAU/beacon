package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/types"
)

func (handler *Handler) unlink(request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	resource, err := handler.directory.GetResource(request.PATH)
	if err != nil { // resource not found
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}
	sourcePath, err := request.PayloadToPath()
	if err != nil {
		return response.Rnum(http.StatusBadRequest).Warning(err.Error()).Build()
	}
	source, err := handler.directory.GetResource(sourcePath)
	if err != nil {
		return response.Rnum(http.StatusNotFound).Warning(err.Error()).Build()
	}
	resp := resource.UnLink(source)
	if resp.Err != nil {
		response.Warning(resp.Err.Error())
	}
	return response.Rnum(resp.Code).Build()
}
