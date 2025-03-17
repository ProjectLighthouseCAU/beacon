package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/ProjectLighthouseCAU/beacon/resource/brokerless"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

func (handler *Handler) create(request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	resrc := brokerless.Create(request.PATH, resource.Nil)
	err := handler.directory.CreateLeaf(request.PATH, resrc)
	if err != nil {
		return response.Warning(err.Error()).Rnum(http.StatusBadRequest).Build()
	}
	return response.Rnum(http.StatusCreated).Build()
}
