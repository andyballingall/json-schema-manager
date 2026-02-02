package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalPath(t *testing.T) {
	t.Parallel()

	t.Run("resolves absolute path", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "test")
		require.NoError(t, os.Mkdir(path, 0o755))

		canonical, err := CanonicalPath(path)
		require.NoError(t, err)
		assert.True(t, filepath.IsAbs(canonical))
		assert.Contains(t, canonical, "test")
	})

	t.Run("resolves symlinks", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		target := filepath.Join(dir, "target")
		require.NoError(t, os.Mkdir(target, 0o755))

		link := filepath.Join(dir, "link")
		require.NoError(t, os.Symlink(target, link))

		canonical, err := CanonicalPath(link)
		require.NoError(t, err)

		expected, _ := filepath.EvalSymlinks(target)
		assert.Equal(t, expected, canonical)
	})

	t.Run("returns error for non-existent path", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "non-existent")

		_, err := CanonicalPath(path)
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("returns error when absFunc fails", func(t *testing.T) { //nolint:paralleltest // modifies global state
		// Not parallel because it modifies global state
		original := absFunc
		defer func() { absFunc = original }()

		absFunc = func(_ string) (string, error) {
			return "", os.ErrPermission
		}

		_, err := CanonicalPath("some/path")
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("returns error when evalSymlinksFunc fails", func(t *testing.T) { //nolint:paralleltest // modifies global state
		// Not parallel because it modifies global state
		original := evalSymlinksFunc
		defer func() { evalSymlinksFunc = original }()

		evalSymlinksFunc = func(_ string) (string, error) {
			return "", os.ErrPermission
		}

		_, err := CanonicalPath("/some/abs/path")
		assert.ErrorIs(t, err, os.ErrPermission)
	})
}
