package hardcoded

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/ProjectLighthouseCAU/beacon/auth"
	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

// AllowCustom is a custom implementation for authorization
type AllowCustom struct {
	Lock   sync.RWMutex
	Users  map[string]string // username -> token
	Admins map[string]bool   // usernames -> is admin flag
}

var _ auth.Auth = (*AllowCustom)(nil)

func New() *AllowCustom {
	return &AllowCustom{
		Users:  parseUserJson(),
		Admins: parseAdminJson(),
	}
}

func parseUserJson() (users map[string]string) {
	err := json.Unmarshal([]byte(config.UsersConfigJson), &users)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("Users: ", config.UsersConfigJson, users)
	return
}

func parseAdminJson() (admins map[string]bool) {
	err := json.Unmarshal([]byte(config.AdminsConfigJson), &admins)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("Admins: ", config.AdminsConfigJson, admins)
	return
}

// IsAuthorized determines whether a request is authorized
func (a *AllowCustom) IsAuthorized(c *types.Client, req *types.Request) (bool, int) {
	username, ok := req.AUTH["USER"]
	if !ok {
		return false, http.StatusUnauthorized
	}
	token, ok := req.AUTH["TOKEN"]
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
		if req.PATH[1] == username && auth.IsReadWriteOperation(req) {
			return true, http.StatusOK
		}
		if auth.IsReadOperation(req) {
			return true, http.StatusOK
		}
	}

	return false, http.StatusForbidden
}
