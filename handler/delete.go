package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

func (handler *Handler) delete(request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	// Close all resources before deleting
	err := handler.directory.ForEach(request.PATH, func(path []string, resource resource.Resource[resource.Content]) (bool, error) {
		resource.Close()
		return true, nil
	})
	if err != nil {
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}
	err = handler.directory.Delete(request.PATH)
	if err != nil {
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}
	return response.Rnum(http.StatusOK).Build()
}
