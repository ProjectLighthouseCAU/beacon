package heimdall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"sync"

	"golang.org/x/exp/maps"

	"github.com/ProjectLighthouseCAU/beacon/auth"
	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/ProjectLighthouseCAU/beacon/types"
	"github.com/ProjectLighthouseCAU/beacon/util"
	"github.com/redis/go-redis/v9"
)

type HeimdallAuth struct {
	dir directory.Directory
	rdb *redis.Client

	lock     sync.RWMutex
	authData map[string]AuthData // username -> token, roles
}

type AuthData struct {
	Token string
	Roles []string
}

func New(dir directory.Directory) *HeimdallAuth {
	auth := &HeimdallAuth{
		dir: dir,
		rdb: redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", config.HeimdallRedisHost, config.HeimdallRedisPort),
			Username: config.HeimdallRedisUser,
			Password: config.HeimdallRedisPassword,
			DB:       config.HeimdallRedisDBNumber,
		}),
		authData: make(map[string]AuthData),
	}
	go util.RunEvery(config.DatabaseQueryInterval, func() {
		err := auth.reload()
		if err != nil {
			log.Println(err)
			// TODO: delete authData when connection to redis failed
		}
	})
	return auth
}

func (a *HeimdallAuth) reload() error {
	// get list of keys
	keys, err := a.rdb.Keys(context.TODO(), "*").Result()
	if err != nil {
		return err
	}
	// get all hashes using a redis pipeline
	results, err := a.rdb.Pipelined(context.TODO(), func(p redis.Pipeliner) error {
		for _, key := range keys {
			p.HGetAll(context.TODO(), key)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// handle the results from redis and build the authData map of authenticated users
	newAuthData := make(map[string]AuthData)
	var errs []error
	for _, r := range results {
		res, ok := r.(*redis.MapStringStringCmd)
		if !ok {
			log.Println("interface conversion from redis.Cmder to *redis.MapStringStringCmd failed")
			continue
		}
		if len(res.Args()) < 2 {
			errs = append(errs, fmt.Errorf("redis args should have length 2 but has length %d", len(res.Args())))
			continue
		}
		key := res.Args()[1].(string)
		hash := res.Val()

		token, ok := hash["token"]
		if !ok {
			errs = append(errs, fmt.Errorf("redis hash with key %s does not exist or does not contain token", key))
			continue
		}
		rolesJson, ok := hash["roles"]
		if !ok {
			errs = append(errs, fmt.Errorf("redis hash with key %s does not exist or does not contain roles", key))
			continue
		}
		var roles []string
		err = json.Unmarshal([]byte(rolesJson), &roles)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		newAuthData[key] = AuthData{
			Token: token,
			Roles: roles,
		}
	}

	a.lock.Lock()
	defer a.lock.Unlock()
	prevUsers := maps.Keys(a.authData)
	currentUsers := maps.Keys(newAuthData)
	addedUsers, removedUsers := util.DiffSlices(prevUsers, currentUsers)

	a.authData = newAuthData

	// create resource for added user
	for _, addedUser := range addedUsers {
		a.dir.CreateResource([]string{"user", addedUser, "model"})
		a.dir.CreateResource([]string{"user", addedUser, "input"})
	}
	// delete resource for removed user
	for _, removedUser := range removedUsers {
		// a.dir.Delete([]string{"user", removedUser}) // TODO: remove old resources (deleted users, not just expired API token)
		delete(a.authData, removedUser)
	}

	return errors.Join(errs...)
}

func (a *HeimdallAuth) IsAuthorized(client *types.Client, request *types.Request) (bool, int) {
	username, ok := request.AUTH["USER"]
	if !ok {
		return false, http.StatusUnauthorized
	}
	token, ok := request.AUTH["TOKEN"]
	if !ok {
		return false, http.StatusUnauthorized
	}
	a.lock.RLock()
	defer a.lock.RUnlock()
	authData, ok := a.authData[username]
	if !ok {
		return false, http.StatusUnauthorized
	}
	if token != authData.Token {
		return false, http.StatusUnauthorized
	}

	// admin role can perform any action on any path
	if slices.Contains(authData.Roles, config.HeimdallAdminRolename) {
		return true, http.StatusOK
	}

	// deploy role can read and write to /metrics
	if slices.Contains(authData.Roles, config.HeimdallDeployRolename) {
		if len(request.PATH) > 0 && request.PATH[0] == "metrics" {
			return true, http.StatusOK
		}
	}

	// TODO: fine grained access control using casbin

	// allow users to read and write to /user/<own-username>/model and /user/<own-username>/input
	// allow users to read /user/<other-username>/model and /user/<other-username>/input
	if request.PATH[0] == "user" && (request.PATH[2] == "model" || request.PATH[2] == "input") && len(request.PATH) == 3 {
		if request.PATH[1] == username && auth.IsReadWriteOperation(request) {
			return true, http.StatusOK
		}
		if auth.IsReadOperation(request) {
			return true, http.StatusOK
		}
	}
	// allow users to read the current live resource contents
	if len(request.PATH) == 1 && request.PATH[0] == "live" && auth.IsReadOperation(request) {
		return true, http.StatusOK
	}

	return false, http.StatusForbidden
}
