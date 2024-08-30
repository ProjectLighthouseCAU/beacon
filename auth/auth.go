// Package auth combines authentication (with user/token) and authorization (permissions).
package auth

import (
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/types"
)

// Auth is the basic interface that an auth implementation must provide
type Auth interface {
	IsAuthorized(*types.Client, *types.Request) (bool, int)
}

// Helper function for determining if an operation is read-only
func IsReadOperation(req *types.Request) bool {
	return map[string]bool{
		"LIST":   true,
		"GET":    true,
		"STREAM": true,
		"STOP":   true,
	}[req.VERB]
}

// --- Combined Authorization Handlers ---

type andAuth struct {
	auth1, auth2 Auth
}

var _ Auth = (*andAuth)(nil)

func NewAndAuth(auth1 Auth, auth2 Auth) *andAuth {
	return &andAuth{auth1, auth2}
}

func (a *andAuth) IsAuthorized(c *types.Client, r *types.Request) (bool, int) {
	a1, _ := a.auth1.IsAuthorized(c, r)
	a2, _ := a.auth2.IsAuthorized(c, r)
	authorized := a1 && a2
	var code int
	if authorized {
		code = 200
	} else {
		code = 401
	}
	return authorized, code
}

type orAuth struct {
	auth1, auth2 Auth
}

var _ Auth = (*orAuth)(nil)

func NewOrAuth(auth1 Auth, auth2 Auth) *orAuth {
	return &orAuth{auth1, auth2}
}

func (a *orAuth) IsAuthorized(c *types.Client, r *types.Request) (bool, int) {
	a1, _ := a.auth1.IsAuthorized(c, r)
	a2, _ := a.auth2.IsAuthorized(c, r)
	authorized := a1 || a2
	var code int
	if authorized {
		code = http.StatusOK
	} else {
		code = http.StatusUnauthorized
	}
	return authorized, code
}

// --- Simple Authorization (AllowAll and AllowNone) ---

// AllowAll allows all requests
type allowAll struct{}

var _ Auth = (*allowAll)(nil)

func AllowAll() *allowAll {
	return &allowAll{}
}

// IsAuthorized determines whether a request is authorized
func (a *allowAll) IsAuthorized(c *types.Client, req *types.Request) (bool, int) {
	return true, http.StatusOK
}

// AllowNone allows no requests
type allowNone struct{}

var _ Auth = (*allowNone)(nil)

func AllowNone() *allowNone {
	return &allowNone{}
}

// IsAuthorized determines whether a request is authorized
func (a *allowNone) IsAuthorized(c *types.Client, req *types.Request) (bool, int) {
	return false, http.StatusUnauthorized
}
