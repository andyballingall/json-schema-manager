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

func TestRun(t *testing.T) { //nolint:paralleltest // uses os.Setenv
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

	// Set the environment variable for the test
	origEnv := os.Getenv("JSM_ROOT_DIRECTORY")
	defer os.Setenv("JSM_ROOT_DIRECTORY", origEnv)

	t.Run("run help", //nolint:paralleltest // uses os.Setenv
		func(t *testing.T) {
			os.Setenv("JSM_ROOT_DIRECTORY", regDir)
			err := Run(context.Background(), []string{"jsm", "--help"}, io.Discard, io.Discard)
			require.NoError(t, err)
		})

	t.Run("run invalid command", //nolint:paralleltest // uses os.Setenv
		func(t *testing.T) {
			os.Setenv("JSM_ROOT_DIRECTORY", regDir)
			err := Run(context.Background(), []string{"jsm", "invalid-command"}, io.Discard, io.Discard)
			require.Error(t, err)
		})

	t.Run("run registry error", //nolint:paralleltest // uses os.Setenv
		func(t *testing.T) {
			os.Setenv("JSM_ROOT_DIRECTORY", "/non/existent/path")
			// The error should now be returned from ExecuteContext (via PersistentPreRunE)
			err := Run(context.Background(), []string{"jsm", "validate", "some.schema.json"}, io.Discard, io.Discard)
			require.Error(t, err)
		})

	t.Run("run setupLogger error", //nolint:paralleltest // uses os.Setenv
		func(t *testing.T) {
			os.Setenv("JSM_ROOT_DIRECTORY", regDir)
			// Set log file to a directory to cause OpenFile to fail
			os.Setenv("JSM_LOG_FILE", regDir)
			defer os.Unsetenv("JSM_LOG_FILE")

			// Use a command that triggers initialization (validate with nonexistent schema)
			err := Run(context.Background(), []string{"jsm", "validate", "missing_1_0_0"}, io.Discard, io.Discard)
			require.Error(t, err) // Will error due to missing schema, but logger warning path should be covered
		})

	t.Run("run discovery failure", //nolint:paralleltest // uses os.Setenv
		func(t *testing.T) {
			os.Unsetenv("JSM_ROOT_DIRECTORY")
			// Move to a directory without config to ensure discovery fails
			origWd, _ := os.Getwd()
			tmpDir := t.TempDir()
			_ = os.Chdir(tmpDir)
			defer func() { _ = os.Chdir(origWd) }()

			// Use validate command instead of --help to trigger initialization
			err := Run(context.Background(), []string{"jsm", "validate", "some_schema_1_0_0"}, io.Discard, io.Discard)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "json-schema-manager-config.yml missing")
		})

	t.Run("run command execution error", //nolint:paralleltest // uses os.Setenv
		func(t *testing.T) {
			os.Setenv("JSM_ROOT_DIRECTORY", regDir)
			// validate with a key that doesn't exist should error
			err := Run(context.Background(), []string{"jsm", "validate", "missing_1_0_0"}, io.Discard, io.Discard)
			require.Error(t, err)
		})

	t.Run("run with debug flag", //nolint:paralleltest // uses os.Setenv
		func(t *testing.T) {
			os.Setenv("JSM_ROOT_DIRECTORY", regDir)
			// Run with debug flag to cover the debug branch in real initialization path
			err := Run(context.Background(), []string{"jsm", "--debug", "validate", "missing_1_0_0"}, io.Discard, io.Discard)
			// This will error because the schema doesn't exist, but the debug path will be covered
			require.Error(t, err)
		})

	t.Run("run distribution initialisation error", //nolint:paralleltest // uses os.Setenv
		func(t *testing.T) {
			// Registry root same as git root should error
			tmpDir := t.TempDir()
			gitRoot := filepath.Join(tmpDir, "project")
			require.NoError(t, os.MkdirAll(gitRoot, 0o755))

			// Init git
			cmd := exec.Command("git", "-C", gitRoot, "init")
			require.NoError(t, cmd.Run())

			require.NoError(t, os.WriteFile(
				filepath.Join(gitRoot, config.JsmRegistryConfigFile),
				[]byte(cfgData),
				0o600,
			))
			os.Setenv("JSM_ROOT_DIRECTORY", gitRoot)

			// Use validate command instead of --help to trigger initialization
			err := Run(context.Background(), []string{"jsm", "validate", "some_schema_1_0_0"}, io.Discard, io.Discard)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "registry root cannot be the same as the git root")
		})
}
