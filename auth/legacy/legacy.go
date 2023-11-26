package legacy

import (
	"fmt"
	"slices"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/auth/hardcoded"
	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"golang.org/x/exp/maps"
)

func New(dir directory.Directory) *hardcoded.AllowCustom {
	db, err := sqlx.Connect("postgres",
		fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			config.GetString("DB_HOST", "localhost"),
			config.GetInt("DB_PORT", 5432),
			config.GetString("DB_USER", "postgres"),
			config.GetString("DB_PASSWORD", "postgres"),
			config.GetString("DB_NAME", "LHP")))
	if err != nil {
		panic(err)
	}
	a := hardcoded.AllowCustom{
		Users:  make(map[string]string),
		Admins: make(map[string]bool),
	}
	go runEvery(config.GetDuration("DB_QUERY_PERIOD", 1*time.Second), func() {
		queryDb(db, &a, dir)
	})
	return &a
}

type User struct {
	Username string `db:"username"`
	Token    string `db:"token"`
}

type Admin struct {
	Username string `db:"username"`
}

const (
	userQuery = `SELECT username, token
FROM webmultiplexer.api_tokens
WHERE permanent
OR issued >= LOCALTIMESTAMP - INTERVAL '2 day 30 minutes'`

	adminQuery = `SELECT webmultiplexer.users.username
FROM webmultiplexer.user_groups
FULL OUTER JOIN webmultiplexer.users
ON user_groups.username = users.username
WHERE users.is_admin
OR user_groups.groupname = 'admin'`
)

func runEvery(t time.Duration, f func()) {
	ticker := time.NewTicker(t)
	for range ticker.C {
		f()
	}
}

func queryDb(db *sqlx.DB, a *hardcoded.AllowCustom, dir directory.Directory) {
	users := []User{}
	admins := []string{}

	err := db.Select(&users, userQuery)
	if err != nil {
		panic(err)
	}
	err = db.Select(&admins, adminQuery)
	if err != nil {
		panic(err)
	}

	a.Lock.Lock()
	defer a.Lock.Unlock()

	// get difference of user map and query result
	addedUsers, removedUsers := diffSlices[string](
		maps.Keys(a.Users),
		mapSlice(func(s User) string { return s.Username }, users))

	// update user map
	for _, user := range users {
		a.Users[user.Username] = user.Token
	}

	// create resource for added user
	for _, addedUser := range addedUsers {
		dir.CreateResource([]string{"user", addedUser, "model"})
	}

	// delete resource for removed user
	for _, removedUser := range removedUsers {
		dir.Delete([]string{"user", removedUser})
		delete(a.Users, removedUser)
	}
	// add new admins
	addedAdmins, removedAdmins := diffSlices[string](maps.Keys(a.Admins), admins)
	for _, admin := range addedAdmins {
		a.Admins[admin] = true
	}
	// remove revoked admins
	for _, admin := range removedAdmins {
		delete(a.Admins, admin)
	}
}

func diffSlices[T comparable](prev []T, next []T) (added, removed []T) {
	for _, p := range prev {
		if !slices.Contains[[]T](next, p) {
			removed = append(removed, p)
		}
	}
	for _, n := range next {
		if !slices.Contains[[]T](prev, n) {
			added = append(added, n)
		}
	}
	return
}

func mapSlice[T, U any](mapFunc func(T) U, slice []T) (result []U) {
	for _, elem := range slice {
		result = append(result, mapFunc(elem))
	}
	return
}
