package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

var (
	// websocket
	WebsocketHost            string = GetString("WEBSOCKET_HOST", "127.0.0.1")
	WebsocketPort            int    = GetInt("WEBSOCKET_PORT", 3000)
	WebsocketReadBufferSize  int    = GetInt("WEBSOCKET_READ_BUFFER_SIZE", 0)
	WebsocketWriteBufferSize int    = GetInt("WEBSOCKET_WRITE_BUFFER_SIZE", 0)
	WebsocketReadLimit       int    = GetInt("WEBSOCKET_READ_LIMIT", 2048)

	// snapshot
	SnapshotPath     string        = GetString("SNAPSHOT_PATH", "./snapshot.beacon")
	SnapshotInterval time.Duration = GetDuration("SNAPSHOT_INTERVAL", 1*time.Second)

	// auth
	Auth string = GetString("AUTH", "allow_none") // valid values: hardcoded, legacy, heimdall, allow_all, allow_none
	// hardcoded
	UsersConfigJson  string = GetString("USERS_CONFIG_JSON", "{}")
	AdminsConfigJson string = GetString("ADMINS_CONFIG_JSON", "{}")
	// heimdall
	HeimdallAdminRolename   string = GetString("HEIMDALL_ADMIN_ROLENAME", "admin")
	HeimdallDeployRolename  string = GetString("HEIMDALL_DEPLOY_ROLENAME", "deploy")
	HeimdallAuthenticateURL string = GetString("HEIMDALL_AUTHENTICATE_URL", "https://lighthouse.uni-kiel.de/api/internal/authenticate")
	HeimdallUsernamesURL    string = GetString("HEIMDALL_USERNAMES_URL", "https://lighthouse.uni-kiel.de/api/internal/users")
	BeaconUsername          string = GetString("BEACON_USERNAME", "")
	BeaconToken             string = GetString("BEACON_TOKEN", "")
	ContainerName           string = GetString("CONTAINER_NAME", "beacon")

	// legacy
	LegacyDatabaseHost     string        = GetString("DB_HOST", "localhost")
	LegacyDatabasePort     int           = GetInt("DB_PORT", 5432)
	LegacyDatabaseUser     string        = GetString("DB_USER", "postgres")
	LegacyDatabasePassword string        = GetString("DB_PASSWORD", "postgres")
	LegacyDatabaseName     string        = GetString("DB_NAME", "LHP")
	DatabaseQueryInterval  time.Duration = GetDuration("DB_QUERY_PERIOD", 1*time.Second)

	// TLS certificates (for https client)
	CaCertificatesFilePath string = GetString("CA_CERTIFICATES_FILE_PATH", "/etc/ssl/certs/ca-certificates.crt")

	// logging
	VerboseLogging bool = GetBool("VERBOSE_LOGGING", false)

	// resource
	ResourceImplementation string = GetString("RESOURCE_IMPL", "brokerless") // valid values: broker, brokerless
	// stream
	ResourceStreamChannelSize int = GetInt("RESOURCE_STREAM_CHANNEL_SIZE", 10)
	// broker-specific
	ResourceInputChannelSize   int = GetInt("RESOURCE_PUT_CHANNEL_SIZE", 10)
	ResourceControlChannelSize int = GetInt("RESOURCE_CONTROL_CHANNEL_SIZE", 10)

	// webinterface (very hacked together)
	WebinterfaceHost  = GetString("WEBINTERFACE_HOST", "127.0.0.1")
	WebinterfaceRoute = GetString("WEBINTERFACE_ROUTE", "/")
	WebinterfacePort  = GetInt("WEBINTERFACE_PORT", 3001)
)

func GetString(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func GetInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		i, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("Found Config %s=%s, but could not parse it (int required)", key, value)
			return defaultValue
		}
		return i
	}
	return defaultValue
}

func GetBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		b, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Found Config %s=%s, but could not parse it (bool required)", key, value)
			return defaultValue
		}
		return b
	}
	return defaultValue
}

func GetDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		d, err := time.ParseDuration(value)
		if err != nil {
			log.Printf("Found Config %s=%s, but could not parse it (duration required, e.g. \"1s\")", key, value)
			return defaultValue
		}
		return d
	}
	return defaultValue
}
