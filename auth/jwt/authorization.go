// WORK IN PROGRESS
package jwt

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/auth"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

type JWTAuth struct{}

var _ auth.Auth = (*JWTAuth)(nil)

func (a *JWTAuth) IsAuthorized(c *types.Client, r *types.Request) (bool, int) {
	// TODO: verify request operation and path by lookup in jwt claims
	// TODO: or make request to auth-service to check the access control rules
	switch r.VERB {
	case "POST": // create permission on path
	case "DELETE": // delete permission on path
	case "LIST": // read permission on path
	case "GET": // read permission on path
	case "PUT": // write permission on path
	case "STREAM": // read permission on path
	case "STOP": // no permission
	case "LINK": // write permission on path AND read permission on payload-encoded path
	case "UNLINK": // write permission on path
		// TODO: consider revoked permission -> links still remain!
	}

	return false, http.StatusNotImplemented
}
