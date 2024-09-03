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
	"time"

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
			Addr:     config.GetString("REDIS_HOST", "127.0.0.1") + ":" + config.GetString("REDIS_PORT", "6379"),
			Username: config.GetString("REDIS_USER", ""),
			Password: config.GetString("REDIS_PASSWORD", ""),
			DB:       config.GetInt("REDIS_DB_NUMBER", 0),
		}),
		authData: make(map[string]AuthData),
	}
	go util.RunEvery(config.GetDuration("DB_QUERY_PERIOD", 1*time.Second), func() {
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
	}
	// delete resource for removed user
	for _, removedUser := range removedUsers {
		a.dir.Delete([]string{"user", removedUser})
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
	if slices.Contains(authData.Roles, config.GetString("HEIMDALL_ADMIN_ROLENAME", "admin")) {
		return true, http.StatusOK
	}
	// TODO: fine grained permission using casbin
	if request.PATH[0] == "user" && request.PATH[2] == "model" && len(request.PATH) == 3 {
		if request.PATH[1] == username {
			return true, http.StatusOK
		}
		if auth.IsReadOperation(request) {
			return true, http.StatusOK
		}
	}
	return false, http.StatusForbidden
}
