package repo

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andyballingall/json-schema-manager/internal/config"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	git("init")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test User")
	git("commit", "--allow-empty", "-m", "initial commit")

	return dir
}

func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Environments: map[config.Env]*config.EnvConfig{
			"prod": {
				Env:                 "prod",
				IsProduction:        true,
				AllowSchemaMutation: false,
				PublicURLRoot:       "https://example.com",
				PrivateURLRoot:      "https://example.com",
			},
		},
	}
}

//nolint:paralleltest // os.Chdir is used
func TestCLIGitter_GetLatestAnchor(t *testing.T) {
	tmpDir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cfg := newTestConfig(t)
	g := NewCLIGitter(cfg)

	t.Run("no tags found - returns root commit", func(t *testing.T) {
		anchor, err := g.GetLatestAnchor("prod")
		require.NoError(t, err)

		// Get root commit hash to compare
		revCmd := exec.Command("git", "rev-list", "--max-parents=0", "HEAD")
		revOut, err := revCmd.Output()
		require.NoError(t, err)
		expected := Revision(strings.TrimSpace(string(revOut)))

		assert.Equal(t, expected, anchor)
	})

	t.Run("tag found", func(t *testing.T) {
		// Create a tag
		cmd := exec.Command("git", "tag", "jsm-deploy/prod/v1")
		require.NoError(t, cmd.Run())

		anchor, err := g.GetLatestAnchor("prod")
		require.NoError(t, err)
		assert.Equal(t, Revision("jsm-deploy/prod/v1"), anchor)
	})

	t.Run("error - invalid env", func(t *testing.T) {
		_, err := g.GetLatestAnchor("invalid-env")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not define environment")
	})

	t.Run("error - not a git repo", func(t *testing.T) {
		emptyDir := t.TempDir()
		require.NoError(t, os.Chdir(emptyDir))
		// No t.Cleanup(Chdir) needed here as it's already cleanup above for TestCLIGitter_GetLatestAnchor

		_, err := g.GetLatestAnchor("prod")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not find git history")
	})
}

//nolint:paralleltest // os.Chdir is used
func TestCLIGitter_GetSchemaChanges(t *testing.T) {
	tmpDir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cfg := newTestConfig(t)
	g := NewCLIGitter(cfg)

	// Setup: initial commit with a tag
	srcDir := "src/schemas"
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	f1 := filepath.Join(srcDir, "user.schema.json")
	require.NoError(t, os.WriteFile(f1, []byte("{}"), 0o600))

	require.NoError(t, exec.Command("git", "add", ".").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "first schema").Run())
	require.NoError(t, exec.Command("git", "tag", "jsm-deploy/prod/v1").Run())

	anchor := Revision("jsm-deploy/prod/v1")

	t.Run("modified file", func(t *testing.T) {
		require.NoError(t, os.WriteFile(f1, []byte(`{"type": "object"}`), 0o600))
		require.NoError(t, exec.Command("git", "add", ".").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "modify schema").Run())

		changes, err := g.GetSchemaChanges(anchor, srcDir, ".schema.json")
		require.NoError(t, err)
		require.Len(t, changes, 1)
		absF1, _ := filepath.Abs(f1)
		assert.Equal(t, absF1, changes[0].Path)
		assert.False(t, changes[0].IsNew)
	})

	t.Run("new file", func(t *testing.T) {
		f2 := filepath.Join(srcDir, "product.schema.json")
		require.NoError(t, os.WriteFile(f2, []byte("{}"), 0o600))
		require.NoError(t, exec.Command("git", "add", ".").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "add new schema").Run())

		changes, err := g.GetSchemaChanges(anchor, srcDir, ".schema.json")
		require.NoError(t, err)
		require.Len(t, changes, 2)

		absF2, _ := filepath.Abs(f2)
		var foundProduct bool
		for _, c := range changes {
			if c.Path == absF2 {
				assert.True(t, c.IsNew)
				foundProduct = true
			}
		}
		assert.True(t, foundProduct)
	})

	t.Run("ignore non-schema files", func(t *testing.T) {
		f3 := filepath.Join(srcDir, "README.md")
		require.NoError(t, os.WriteFile(f3, []byte("docs"), 0o600))
		require.NoError(t, exec.Command("git", "add", ".").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "add readme").Run())

		changes, err := g.GetSchemaChanges(anchor, srcDir, ".schema.json")
		require.NoError(t, err)
		require.Len(t, changes, 2)
	})

	t.Run("git diff error", func(t *testing.T) {
		_, err := g.GetSchemaChanges(Revision("invalid-anchor"), srcDir, ".schema.json")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git diff failed")
	})

	t.Run("no changes", func(t *testing.T) {
		// New repo with one commit and tag
		dir := setupTestRepo(t)
		require.NoError(t, os.Chdir(dir))
		// Note: no cleanup needed, we'll return to origDir after TestCLIGitter_GetSchemaChanges

		cmd := exec.Command("git", "tag", "jsm-deploy/prod/v2")
		require.NoError(t, cmd.Run())

		changes, err := g.GetSchemaChanges(Revision("jsm-deploy/prod/v2"), srcDir, ".schema.json")
		require.NoError(t, err)
		assert.Empty(t, changes)
	})

	t.Run("absPath error", func(t *testing.T) { //nolint:paralleltest // modifies global state
		origAbsPath := absPath
		absPath = func(_ string) (string, error) {
			return "", errors.New("absPath failure")
		}
		defer func() { absPath = origAbsPath }()

		_, err := g.GetSchemaChanges(Revision("HEAD"), "some/path", ".schema.json")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "absPath failure")
	})

	t.Run("subdirectory path resolution", func(t *testing.T) {
		// 1. Setup repo with registry in a subdirectory
		repoDir := setupTestRepo(t)
		registryDir := filepath.Join(repoDir, "my-registry")
		require.NoError(t, os.MkdirAll(registryDir, 0o755))

		// Get initial commit (anchor) before adding schema
		anchorOut, err := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD").Output()
		require.NoError(t, err)
		initialAnchor := Revision(strings.TrimSpace(string(anchorOut)))

		// 2. Add a schema and commit it
		schemaFile := filepath.Join(registryDir, "test.schema.json")
		require.NoError(t, os.WriteFile(schemaFile, []byte("{}"), 0o600))
		require.NoError(t, exec.Command("git", "-C", repoDir, "add", ".").Run())
		require.NoError(t, exec.Command("git", "-C", repoDir, "commit", "-m", "add schema").Run())

		// 3. Change workdir to registry
		origCWD, _ := os.Getwd()
		require.NoError(t, os.Chdir(registryDir))
		defer func() { _ = os.Chdir(origCWD) }()

		// 4. Call GetSchemaChanges with "." as sourceDir
		changes, err := g.GetSchemaChanges(initialAnchor, ".", ".schema.json")
		require.NoError(t, err)

		// 5. Verify paths are absolute or correctly resolvable
		// Current bug: returns "my-registry/test.schema.json" (repo-relative)
		// Expected: absolute path or correctly resolvable path from current CWD
		require.Len(t, changes, 1)

		// If it's absolute, this will pass. If it's the broken repo-relative path,
		// it will likely fail this check because it's looking for my-registry/my-registry/...
		_, statErr := os.Stat(changes[0].Path)
		assert.NoError(t, statErr, "Path %s should be resolvable from CWD %s", changes[0].Path, registryDir)
	})
	t.Run("getGitRoot error", func(t *testing.T) {
		emptyDir := t.TempDir()
		origCWD, _ := os.Getwd()
		require.NoError(t, os.Chdir(emptyDir))
		defer func() { _ = os.Chdir(origCWD) }()

		_, err := g.GetSchemaChanges(Revision("HEAD"), ".", ".schema.json")
		require.Error(t, err)
	})
}

//nolint:paralleltest // os.Chdir is used
func TestCLIGitter_getGitRoot(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repoDir := setupTestRepo(t)
		origCWD, _ := os.Getwd()
		require.NoError(t, os.Chdir(repoDir))
		defer func() { _ = os.Chdir(origCWD) }()

		g := NewCLIGitter(newTestConfig(t))
		root, err := g.getGitRoot()
		require.NoError(t, err)

		expected, _ := filepath.EvalSymlinks(repoDir)
		actual, _ := filepath.EvalSymlinks(root)
		assert.Equal(t, expected, actual)
	})

	t.Run("error - not a git repo", func(t *testing.T) {
		emptyDir := t.TempDir()
		origCWD, _ := os.Getwd()
		require.NoError(t, os.Chdir(emptyDir))
		defer func() { _ = os.Chdir(origCWD) }()

		g := NewCLIGitter(newTestConfig(t))
		_, err := g.getGitRoot()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find git root")
	})
}

// TestCLIGitter_TagDeploymentSuccess verifies the tag creation and push logic.
//
//nolint:paralleltest // os.Chdir is used
func TestCLIGitter_TagDeploymentSuccess(t *testing.T) {
	tmpDir := setupTestRepo(t)
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cfg := newTestConfig(t)
	g := NewCLIGitter(cfg)

	t.Run("success without remote", func(t *testing.T) {
		// This should fail on push, but tagName should be returned
		tagName, err := g.TagDeploymentSuccess("prod")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to push git tag to origin")
		assert.NotEmpty(t, tagName)
		assert.Contains(t, tagName, "jsm-deploy/prod/")

		// Verify tag exists locally
		cmd := exec.Command("git", "rev-parse", tagName)
		require.NoError(t, cmd.Run())
	})

	t.Run("success with remote", func(t *testing.T) {
		// Setup a "remote"
		remoteDir := t.TempDir()
		// Init remote as shared/bare repo to allow pushing
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

		repoDir := t.TempDir()
		require.NoError(t, exec.Command("git", "init", repoDir).Run())
		require.NoError(t, os.Chdir(repoDir))
		require.NoError(t, exec.Command("git", "config", "user.email", "t@t.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "t").Run())
		require.NoError(t, exec.Command("git", "commit", "--allow-empty", "-m", "init").Run())
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())

		g2 := NewCLIGitter(cfg)
		tagName, err := g2.TagDeploymentSuccess("prod")
		require.NoError(t, err)
		assert.Contains(t, tagName, "jsm-deploy/prod/")

		// Verify tag exists on "remote"
		cmd := exec.Command("git", "-C", remoteDir, "rev-parse", tagName)
		require.NoError(t, cmd.Run())

		// Cleanup: go back to tmpDir
		require.NoError(t, os.Chdir(tmpDir))
	})

	t.Run("tag failure", func(t *testing.T) {
		dir := t.TempDir()
		binDir := filepath.Join(dir, "bin")
		require.NoError(t, os.Mkdir(binDir, 0o755))

		realGit, _ := exec.LookPath("git")
		// Script that fails only if 'tag' is an argument
		gitScript := fmt.Sprintf(`#!/bin/sh
for arg in "$@"; do
	if [ "$arg" = "tag" ]; then
		exit 1
	fi
done
exec %s "$@"
`, realGit)
		//nolint:gosec // need executable permission for mock git
		require.NoError(t, os.WriteFile(filepath.Join(binDir, "git"), []byte(gitScript), 0o755))

		origPath := os.Getenv("PATH")
		_ = os.Setenv("PATH", binDir+":"+origPath)
		defer func() { _ = os.Setenv("PATH", origPath) }()

		_, err := g.TagDeploymentSuccess("prod")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create git tag")
	})

	t.Run("error - invalid env", func(t *testing.T) {
		_, err := g.TagDeploymentSuccess("invalid-env")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not define environment")
	})
}

func TestCLIGitter_tagPrefix(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)
	g := NewCLIGitter(cfg)

	prefix := g.tagPrefix("prod")
	assert.Equal(t, "jsm-deploy/prod", prefix)

	prefix = g.tagPrefix("staging")
	assert.Equal(t, "jsm-deploy/staging", prefix)
}
