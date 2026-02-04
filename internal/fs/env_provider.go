package fs

import (
	"os"
)

// EnvProvider provides environment variable access.
type EnvProvider interface {
	// Get returns the value of the environment variable named by the key.
	Get(key string) string
}

// OSEnvProvider reads from the actual environment using os.Getenv.
type OSEnvProvider struct{}

// NewEnvProvider creates a new OSEnvProvider.
func NewEnvProvider() *OSEnvProvider {
	return &OSEnvProvider{}
}

// Get returns the value of the environment variable named by the key.
func (e *OSEnvProvider) Get(key string) string {
	return os.Getenv(key)
}
