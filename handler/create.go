package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/types"
)

func (handler *Handler) create(request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	err := handler.directory.CreateResource(request.PATH)
	if err != nil {
		return response.Warning(err.Error()).Rnum(http.StatusBadRequest).Build()
	}
	return response.Rnum(http.StatusCreated).Build()
}
