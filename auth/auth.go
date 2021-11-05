// Package auth combines authentication (with user/token) and authorization (permissions).
package auth

import (
	"net/http"

	"lighthouse.uni-kiel.de/lighthouse-server/types"
)

// Auth is the basic interface that an auth implementation must provide
type Auth interface {
	IsAuthorized(*types.Client, *types.Request) (bool, int)
}

// Helper function for determining if an operation is read-only
func isReadOperation(req *types.Request) bool {
	return map[string]bool{
		"LIST":   true,
		"GET":    true,
		"STREAM": true,
		"STOP":   true,
	}[req.VERB]
}

// --- Combined Authorization Handlers ---

type AndAuth struct {
	auth1, auth2 Auth
}

var _ Auth = (*AndAuth)(nil)

func (a *AndAuth) IsAuthorized(c *types.Client, r *types.Request) (bool, int) {
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

type OrAuth struct {
	auth1, auth2 Auth
}

var _ Auth = (*OrAuth)(nil)

func (a *OrAuth) IsAuthorized(c *types.Client, r *types.Request) (bool, int) {
	a1, _ := a.auth1.IsAuthorized(c, r)
	a2, _ := a.auth2.IsAuthorized(c, r)
	authorized := a1 || a2
	var code int
	if authorized {
		code = 200
	} else {
		code = 401
	}
	return authorized, code
}

// --- Simple Authorization (AllowAll and AllowNone) ---

// AllowAll allows all requests
type AllowAll struct{}

var _ Auth = (*AllowAll)(nil)

// IsAuthorized determines whether a request is authorized
func (a *AllowAll) IsAuthorized(c *types.Client, req *types.Request) (bool, int) {
	return true, http.StatusOK
}

// AllowNone allows no requests
type AllowNone struct{}

var _ Auth = (*AllowNone)(nil)

// IsAuthorized determines whether a request is authorized
func (a *AllowNone) IsAuthorized(c *types.Client, req *types.Request) (bool, int) {
	return false, http.StatusUnauthorized
}
