package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testKey = "TEST_ENV_VAR"

func TestGet(t *testing.T) {
	os.Unsetenv(testKey)

	// Not set
	assert.Equal(t, "", Get(testKey))

	// Set
	os.Setenv(testKey, "hello")
	assert.Equal(t, "hello", Get(testKey))

	os.Unsetenv(testKey)
}

func TestGetOrDefault(t *testing.T) {
	os.Unsetenv(testKey)

	// Not set, returns default
	assert.Equal(t, "default", GetOrDefault(testKey, "default"))

	// Set, returns value
	os.Setenv(testKey, "value")
	assert.Equal(t, "value", GetOrDefault(testKey, "default"))

	os.Unsetenv(testKey)
}
