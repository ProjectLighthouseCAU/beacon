package hardcoded

import (
	"net/http"
	"sync"

	"github.com/ProjectLighthouseCAU/beacon/auth"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

// AllowCustom is a custom implementation for authorization
type AllowCustom struct {
	Lock   sync.RWMutex
	Users  map[string]string // username -> token
	Admins map[string]bool   // usernames -> is admin flag
}

var _ auth.Auth = (*AllowCustom)(nil)

// IsAuthorized determines whether a request is authorized
func (a *AllowCustom) IsAuthorized(c *types.Client, req *types.Request) (bool, int) {
	username, ok := req.AUTH["USER"].(string)
	if !ok {
		return false, http.StatusUnauthorized
	}
	token, ok := req.AUTH["TOKEN"].(string)
	if !ok {
		return false, http.StatusUnauthorized
	}
	a.Lock.RLock()
	defer a.Lock.RUnlock()
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

	if req.PATH[0] == "user" && req.PATH[2] == "model" && len(req.PATH) == 3 {
		if req.PATH[1] == username {
			return true, http.StatusOK
		}
		if auth.IsReadOperation(req) {
			return true, http.StatusOK
		}
	}

	return false, http.StatusForbidden
}
