package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_KEY", "hello")
	defer os.Unsetenv("TEST_KEY")
	assert.Equal(t, "hello", getEnv("TEST_KEY", "default"))
	assert.Equal(t, "default", getEnv("MISSING_KEY", "default"))
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")
	assert.Equal(t, 42, getEnvInt("TEST_INT", 0))
	assert.Equal(t, 10, getEnvInt("MISSING_INT", 10))
	os.Setenv("BAD_INT", "abc")
	assert.Equal(t, 0, getEnvInt("BAD_INT", 0))
}

func TestRequireEnv(t *testing.T) {
	os.Setenv("REQ_KEY", "required")
	defer os.Unsetenv("REQ_KEY")
	assert.Equal(t, "required", requireEnv("REQ_KEY"))
}

func TestIsDev(t *testing.T) {
	_ = IsDev()
}
