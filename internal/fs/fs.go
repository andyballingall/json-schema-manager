package fs

import (
	"path/filepath"
)

var (
	absFunc          = filepath.Abs
	evalSymlinksFunc = filepath.EvalSymlinks
)

// CanonicalPath returns the canonical, absolute path by resolving symlinks.
func CanonicalPath(path string) (string, error) {
	abs, err := absFunc(path)
	if err != nil {
		return "", err
	}

	cp, err := evalSymlinksFunc(abs)
	if err != nil {
		return "", err
	}
	return cp, nil
}
