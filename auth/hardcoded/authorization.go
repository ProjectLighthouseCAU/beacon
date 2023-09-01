// TEMPORARY SOLUTION
package hardcoded

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/auth"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

// AllowCustom is a custom implementation for authorization
type AllowCustom struct {
	Users  map[string]string // username -> token
	Admins map[string]bool   // usernames -> is admin flag
}

var _ auth.Auth = (*AllowCustom)(nil)

// IsAuthorized determines whether a request is authorized
func (a *AllowCustom) IsAuthorized(c *types.Client, req *types.Request) (bool, int) {
	iUsername, ok := req.AUTH["USER"]
	if !ok {
		return false, http.StatusUnauthorized
	}
	username, ok := iUsername.(string)
	if !ok {
		return false, http.StatusUnauthorized
	}
	iToken, ok := req.AUTH["TOKEN"]
	if !ok {
		return false, http.StatusUnauthorized
	}
	token, ok := iToken.(string)
	if !ok {
		return false, http.StatusUnauthorized
	}
	correctToken, ok := a.Users[username]
	if !ok {
		return false, http.StatusUnauthorized
	}
	if token != correctToken {
		return false, http.StatusUnauthorized
	}

	isAdmin := a.Admins[username]
	if isAdmin {
		return true, http.StatusOK
	}

	if req.PATH[0] == "user" && req.PATH[2] == "model" {
		if req.PATH[1] == username {
			return true, http.StatusOK
		}
		if auth.IsReadOperation(req) {
			return true, http.StatusOK
		}
	}

	return false, http.StatusForbidden
}
