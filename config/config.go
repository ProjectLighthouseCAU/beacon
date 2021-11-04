package config

import (
	"log"
	"os"
	"strconv"
	"time"
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
