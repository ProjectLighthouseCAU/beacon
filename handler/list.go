package handler

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/types"
	"github.com/vmihailenco/msgpack/v5"
)

func (handler *Handler) list(request *types.Request) *types.Response {
	response := types.NewResponse().Reid(request.REID)
	var lst any
	var err error
	nonrecursive, metaNonrecursiveExists := request.META["NONRECURSIVE"].(bool) // TODO: maybe inverse to keep backwards compatible
	if metaNonrecursiveExists && nonrecursive {
		lst, err = handler.directory.List(request.PATH)
		if err != nil {
			return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
		}
	} else {
		lst, err = handler.directory.ListRecursive(request.PATH)
		if err != nil {
			return response.Warning(err.Error()).Rnum(http.StatusNotFound).Build()
		}
	}
	payl, err := msgpack.Marshal(lst)
	if err != nil {
		return response.Warning(err.Error()).Rnum(http.StatusInternalServerError).Build()
	}
	return response.Rnum(http.StatusOK).Payload(payl).Build()
}
