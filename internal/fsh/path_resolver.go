package fsh

import (
	"path/filepath"
)

// PathResolver provides path resolution operations.
type PathResolver interface {
	// CanonicalPath returns the canonical, absolute path by resolving symlinks.
	CanonicalPath(path string) (string, error)
	// Abs returns the absolute path.
	Abs(path string) (string, error)
	// GetUintSubdirectories returns a slice of uint64s corresponding to subdirectories
	// of the given path that are compatible with uint64, sorted in ascending order.
	GetUintSubdirectories(dirPath string) ([]uint64, error)
}

// StandardPathResolver is the default implementation using standard library functions.
type StandardPathResolver struct{}

// NewPathResolver creates a new StandardPathResolver.
func NewPathResolver() *StandardPathResolver {
	return &StandardPathResolver{}
}

// CanonicalPath returns the canonical, absolute path by resolving symlinks.
func (r *StandardPathResolver) CanonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return filepath.EvalSymlinks(abs)
}

// Abs returns the absolute path.
func (r *StandardPathResolver) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

// GetUintSubdirectories returns a slice of uint64s corresponding to subdirectories
// of the given path that are compatible with uint64, sorted in ascending order.
func (r *StandardPathResolver) GetUintSubdirectories(dirPath string) ([]uint64, error) {
	return GetUintSubdirectories(dirPath)
}
