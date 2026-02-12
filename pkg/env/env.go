package env

import (
	"os"
)

// Get returns the value of the environment variable.
// Returns empty string if the variable is not set.
func Get(key string) string {
	return os.Getenv(key)
}

// GetOrDefault returns the value of the environment variable.
// If the variable is not set, it returns the default value.
func GetOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
