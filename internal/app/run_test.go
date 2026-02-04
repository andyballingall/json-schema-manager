package app

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andyballingall/json-schema-manager/internal/config"
)

func TestRun(t *testing.T) {
	t.Parallel()

	// Setup a temporary registry
	regDir := t.TempDir()
	cfgData := `
environments:
  prod:
    privateUrlRoot: "https://json-schemas.internal.myorg.io/"
    publicUrlRoot: "https://json-schemas.myorg.io/"
    isProduction: true
`
	if err := os.WriteFile(
		filepath.Join(regDir, config.JsmRegistryConfigFile),
		[]byte(cfgData),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	t.Run("run help", func(t *testing.T) {
		t.Parallel()
		env := &mockEnvProvider{values: map[string]string{
			"JSM_REGISTRY_ROOT_DIR": regDir,
		}}
		err := Run(context.Background(), []string{"jsm", "--help"}, io.Discard, io.Discard, env)
		require.NoError(t, err)
	})

	t.Run("run invalid command", func(t *testing.T) {
		t.Parallel()
		env := &mockEnvProvider{values: map[string]string{
			"JSM_REGISTRY_ROOT_DIR": regDir,
		}}
		err := Run(context.Background(), []string{"jsm", "invalid-command"}, io.Discard, io.Discard, env)
		require.Error(t, err)
	})

	t.Run("run registry error", func(t *testing.T) {
		t.Parallel()
		env := &mockEnvProvider{values: map[string]string{
			"JSM_REGISTRY_ROOT_DIR": "/non/existent/path",
		}}
		err := Run(context.Background(), []string{"jsm", "validate", "some.schema.json"}, io.Discard, io.Discard, env)
		require.Error(t, err)
	})

	t.Run("run setupLogger error", func(t *testing.T) {
		t.Parallel()
		// Set log file to a directory to cause OpenFile to fail
		env := &mockEnvProvider{values: map[string]string{
			"JSM_REGISTRY_ROOT_DIR": regDir,
			"JSM_LOG_FILE":          regDir,
		}}

		// Use a command that triggers initialization
		err := Run(context.Background(), []string{"jsm", "validate", "missing_1_0_0"}, io.Discard, io.Discard, env)
		require.Error(t, err)
	})

	t.Run("run discovery failure", func(t *testing.T) {
		t.Parallel()
		// Registry root directory not set and not in a registry directory
		env := &mockEnvProvider{values: map[string]string{
			"JSM_REGISTRY_ROOT_DIR": "",
		}}
		// We'll pass tmpDir as a placeholder, but discovery should fail if it's not a registry
		// Actually, PathResolver.Abs(".") in NewRegistry will return CWD.
		// Since we want to avoid Chdir, we rely on the fact that JSM_REGISTRY_ROOT_DIR is empty
		// and the current CWD (project root) is not where the config is if we are elsewhere.
		// Wait, if we are in project root, it MIGHT find it.
		// Let's use a mock env provider that returns nothing.

		err := Run(context.Background(), []string{"jsm", "validate", "some_schema_1_0_0"}, io.Discard, io.Discard, env)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "json-schema-manager-config.yml missing")
	})

	t.Run("run command execution error", func(t *testing.T) {
		t.Parallel()
		env := &mockEnvProvider{values: map[string]string{
			"JSM_REGISTRY_ROOT_DIR": regDir,
		}}
		err := Run(context.Background(), []string{"jsm", "validate", "missing_1_0_0"}, io.Discard, io.Discard, env)
		require.Error(t, err)
	})

	t.Run("run with debug flag", func(t *testing.T) {
		t.Parallel()
		env := &mockEnvProvider{values: map[string]string{
			"JSM_REGISTRY_ROOT_DIR": regDir,
		}}
		err := Run(context.Background(), []string{"jsm", "--debug", "validate", "missing_1_0_0"}, io.Discard, io.Discard, env)
		require.Error(t, err)
	})

	t.Run("run distribution initialisation error", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		gitRoot := filepath.Join(tmpDir, "project")
		require.NoError(t, os.MkdirAll(gitRoot, 0o755))

		cmd := exec.Command("git", "init", gitRoot)
		require.NoError(t, cmd.Run())

		require.NoError(t, os.WriteFile(
			filepath.Join(gitRoot, config.JsmRegistryConfigFile),
			[]byte(cfgData),
			0o600,
		))
		env := &mockEnvProvider{values: map[string]string{
			"JSM_REGISTRY_ROOT_DIR": gitRoot,
		}}

		err := Run(context.Background(), []string{"jsm", "validate", "some_schema_1_0_0"}, io.Discard, io.Discard, env)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registry root cannot be the same as the git root")
	})
}
