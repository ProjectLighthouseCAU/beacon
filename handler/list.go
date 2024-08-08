package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/types"
	"github.com/vmihailenco/msgpack"
)

func (handler *Handler) list(request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	// TODO: return also string representation of the directory when request contains a META tag
	lst, err := handler.directory.List(request.PATH)
	if err != nil {
		return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
	}
	payl, err := msgpack.Marshal(lst)
	if err != nil {
		return response.Warning(err.Error()).Rnum(http.StatusInternalServerError).Build()
	}
	return response.Rnum(http.StatusOK).Payload(payl).Build()
}
