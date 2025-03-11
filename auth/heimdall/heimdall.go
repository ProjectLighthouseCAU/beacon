package heimdall

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/auth"
	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/ProjectLighthouseCAU/beacon/types"
)

type HeimdallAuth struct {
	client *http.Client
}

// Message that is sent to notify subscribers on changes to one of these authentication related values
type AuthUpdateMessage struct {
	Username  string    `json:"username"`   // unique username associated with this token
	Token     string    `json:"api_token"`  // the actual API token
	ExpiresAt time.Time `json:"expires_at"` // expiration date of this token
	Permanent bool      `json:"permanent"`  // permanent token (ignore expires_at)
	Roles     []string  `json:"roles"`      // roles associated with this token
}

// Message that is sent to notify subscribers (e.g. Beacon) when a new user is created or a user is removed
type UsersUpdateMessage struct {
	Username string `json:"username"`
	Removed  bool   `json:"removed"`
}

var (
	errKeepAliveMessage = errors.New("received keep alive message")
)

func New(dir directory.Directory) *HeimdallAuth {
	auth := HeimdallAuth{
		client: http.DefaultClient,
	}
	go func() {
		for {
			err := auth.directoryUpdater(dir)
			if err != nil {
				log.Println(err)
			}
			log.Println("Directory updater lost connection, retrying in 3 seconds...")
			time.Sleep(3 * time.Second)
		}
	}()
	return &auth
}

func (a *HeimdallAuth) directoryUpdater(dir directory.Directory) error {
	req, err := http.NewRequest("GET", config.HeimdallUsernamesURL, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", config.BeaconToken)
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	log.Println("Directory updater connected!")
	reader := bufio.NewReader(resp.Body)
	for {
		msg, err := a.readUsersUpdateMessage(reader)
		if err != nil {
			if err == errKeepAliveMessage {
				continue
			}
			return err
		}
		log.Printf("Received: %+v\n", msg)
		if msg.Removed {
			_ = dir.Delete([]string{"user", msg.Username})
			continue
		}
		_ = dir.CreateResource([]string{"user", msg.Username, "model"})
		_ = dir.CreateResource([]string{"user", msg.Username, "input"})
	}
}

func (a *HeimdallAuth) getAuthEntry(client *types.Client, username, token string) (*types.AuthCacheEntry, error) {
	// check cache
	entry := client.LookupAuthCache(username)
	if entry != nil { // cache hit
		return entry, nil
	}
	// cache miss

	// query the auth endpoint
	req, err := http.NewRequest("GET", config.HeimdallAuthenticateURL+"/"+username, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", config.BeaconToken)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	reader := bufio.NewReader(resp.Body)
	// the first response (AuthUpdateMessage) is guaranteed to arrive directly upon request
	msg, err := a.readAuthUpdateMessage(reader)
	if err != nil {
		return nil, err
	}
	if config.VerboseLogging {
		log.Printf("[HeimdallAuth] Received first AuthUpdateMessage for user %s: %+v\n", username, msg)
	}

	entry = &types.AuthCacheEntry{
		Token:     msg.Token,
		ExpiresAt: msg.ExpiresAt,
		Permanent: msg.Permanent,
		Roles:     msg.Roles,
	}
	client.SetAuthCacheEntry(msg.Username, entry)

	// start a new goroutine for receiving update messages and updating the cache
	ctx, cancel := context.WithCancel(context.Background())
	client.AddAuthCacheUpdaterCancelFunc(username, cancel) // for stopping the goroutine on client disconnect
	onStop := func() {
		resp.Body.Close()
		client.RemoveAuthCacheUpdaterCancelFunc(username)
		cancel()
	}
	go a.cacheEntryUpdater(ctx, onStop, reader, username, client)

	return entry, nil
}

func (a *HeimdallAuth) cacheEntryUpdater(ctx context.Context, onStop func(), reader *bufio.Reader, username string, client *types.Client) {
	// delete the cache entry if the updater exits (connection closed or error)
	defer func() {
		client.DeleteAuthCacheEntry(username)
		onStop()
	}()
	for {
		if ctx.Err() != nil { // updater canceled
			return
		}
		msg, err := a.readAuthUpdateMessage(reader)
		if err != nil {
			if err == errKeepAliveMessage {
				continue
			}
			if err != io.EOF {
				log.Printf("Error in Heimdall auth cache entry updater for user %s: %v - closed auth update connection and deleted auth cache entry\n", username, err)
			}
			return
		}
		if config.VerboseLogging {
			log.Printf("[HeimdallAuth] Received AuthUpdateMessage for user %s: %+v\n", username, msg)
		}
		client.SetAuthCacheEntry(username, &types.AuthCacheEntry{
			Token:     msg.Token,
			ExpiresAt: msg.ExpiresAt,
			Permanent: msg.Permanent,
			Roles:     msg.Roles,
		})
	}
}

func readFullLine(reader *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		lineFragment, isPrefix, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}
		line = append(line, lineFragment...)
		if !isPrefix {
			break
		}
	}
	return line, nil
}

func (a *HeimdallAuth) readAuthUpdateMessage(reader *bufio.Reader) (*AuthUpdateMessage, error) {
	line, err := readFullLine(reader)
	if err != nil {
		return nil, err
	}
	// special case: keep alive messages contain just "\r\n", we therefore skip empty lines
	// TODO: decide if we want to send "null\r\n" or just "\r\n"
	if len(line) == 0 || len(line) == 4 && string(line) == "null" {
		return nil, errKeepAliveMessage
	}
	var msg AuthUpdateMessage
	err = json.Unmarshal(line, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (a *HeimdallAuth) readUsersUpdateMessage(reader *bufio.Reader) (*UsersUpdateMessage, error) {
	line, err := readFullLine(reader)
	if err != nil {
		return nil, err
	}
	// special case: keep alive messages contain just "\r\n", we therefore skip empty lines
	// TODO: decide if we want to send "null\r\n" or just "\r\n"
	if len(line) == 0 || len(line) == 4 && string(line) == "null" {
		return nil, errKeepAliveMessage
	}
	var msg UsersUpdateMessage
	err = json.Unmarshal(line, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
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
	entry, err := a.getAuthEntry(client, username, token)
	if err != nil {
		return false, http.StatusUnauthorized
	}

	if token != entry.Token || (!entry.Permanent && entry.ExpiresAt.Before(time.Now())) {
		return false, http.StatusUnauthorized
	}

	if config.VerboseLogging {
		log.Printf("[HeimdallAuth] Authenticated user %s with entry: %+v\n", username, entry)
	}

	// admin role can perform any action on any path
	if slices.Contains(entry.Roles, config.HeimdallAdminRolename) {
		return true, http.StatusOK
	}

	// deploy role can read and write to /metrics
	if slices.Contains(entry.Roles, config.HeimdallDeployRolename) {
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
