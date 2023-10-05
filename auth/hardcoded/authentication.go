// Integration with the legacy system (direct database access) not possible
// because legacy PostgreSQL listens on 127.0.0.1
// which is not accessible from within the docker container (except with host networking)
// -> using static json config file instead for testing
package hardcoded

import (
	"encoding/json"
	"fmt"

	"github.com/ProjectLighthouseCAU/beacon/config"
)

var (
	usersConfigJson  = config.GetString("USERS_CONFIG_JSON", "{}")
	adminsConfigJson = config.GetString("ADMINS_CONFIG_JSON", "{}")
)

func New() *AllowCustom {
	return &AllowCustom{
		Users:  ParseUserJson(),
		Admins: ParseAdminJson(),
	}
}

func ParseUserJson() (users map[string]string) {
	json.Unmarshal([]byte(usersConfigJson), &users)
	fmt.Println("Users: ", usersConfigJson, users)
	return
}

func ParseAdminJson() (admins map[string]bool) {
	json.Unmarshal([]byte(adminsConfigJson), &admins)
	fmt.Println("Admins: ", adminsConfigJson, admins)
	return
}
