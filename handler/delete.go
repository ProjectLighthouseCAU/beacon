package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/types"
)

func (handler *Handler) delete(request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	err := handler.directory.Delete(request.PATH)
	if err != nil {
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}
	return response.Rnum(http.StatusOK).Build()
}
