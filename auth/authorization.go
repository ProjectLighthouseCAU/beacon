// WORK IN PROGRESS
package auth

import (
	"net/http"

	"lighthouse.uni-kiel.de/lighthouse-server/types"
)

type JWTAuth struct{}

var _ Auth = (*JWTAuth)(nil)

func (a *JWTAuth) IsAuthorized(c *types.Client, r *types.Request) (bool, int) {
	if c.Claims == nil {
		// TODO: try to parse the jwt again (in case endpoint authentication is disabled)
		return false, http.StatusUnauthorized
	}
	// TODO: verify request operation and path by lookup in jwt claims

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

	return false, -1
}
