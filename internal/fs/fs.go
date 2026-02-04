package fs

// defaultResolver is used by the package-level CanonicalPath function.
var defaultResolver = NewPathResolver()

// CanonicalPath returns the canonical, absolute path by resolving symlinks.
// This is a convenience function that uses the default StandardPathResolver.
func CanonicalPath(path string) (string, error) {
	return defaultResolver.CanonicalPath(path)
}

// Abs returns the absolute path.
// This is a convenience function that uses the default StandardPathResolver.
func Abs(path string) (string, error) {
	return defaultResolver.Abs(path)
}
