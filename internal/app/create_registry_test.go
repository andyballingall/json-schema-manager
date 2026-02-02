package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andyballingall/json-schema-manager/internal/config"
)

func TestNewCreateRegistryCmd(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		registryDir := filepath.Join(tmpDir, "my-registry")

		cmd := NewCreateRegistryCmd()
		cmd.SetArgs([]string{registryDir})

		err := cmd.Execute()
		require.NoError(t, err)

		// Verify directory exists
		info, err := os.Stat(registryDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Verify config file exists
		configPath := filepath.Join(registryDir, config.JsmRegistryConfigFile)
		data, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Equal(t, config.DefaultConfigContent, string(data))
	})

	t.Run("error - config file already exists", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, config.JsmRegistryConfigFile)
		err := os.WriteFile(configPath, []byte("existing"), 0o600)
		require.NoError(t, err)

		cmd := NewCreateRegistryCmd()
		cmd.SetArgs([]string{tmpDir})

		err = cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registry already exists")
	})

	t.Run("error - cannot create directory", func(t *testing.T) {
		t.Parallel()
		// On most systems, creating a directory in a non-existent path without MkdirAll -p equivalent would fail,
		// but MkdirAll handles it. So we need a real permission error or similar.
		// A simple way is to use a file where a directory should be.
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "some-file")
		err := os.WriteFile(filePath, []byte("not-a-dir"), 0o600)
		require.NoError(t, err)

		badDir := filepath.Join(filePath, "nested")

		cmd := NewCreateRegistryCmd()
		cmd.SetArgs([]string{badDir})

		err = cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create directory")
	})

	t.Run("error - missing argument", func(t *testing.T) {
		t.Parallel()
		cmd := NewCreateRegistryCmd()
		cmd.SetArgs([]string{})

		// Cobra will handle this and return an error before RunE
		err := cmd.Execute()
		require.Error(t, err)
	})

	t.Run("error - failed to write config file", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		registryDir := filepath.Join(tmpDir, "readonly-dir")
		err := os.Mkdir(registryDir, 0o555) // Read and execute but no write
		require.NoError(t, err)

		// Ensure cleanup if it fails (t.TempDir usually handles this but good to be safe)
		defer func() {
			_ = os.Chmod(registryDir, 0o755)
		}()

		cmd := NewCreateRegistryCmd()
		cmd.SetArgs([]string{registryDir})

		err = cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write configuration file")
	})
}

// TestRegistration verifies the command is registered in RootCmd.
func TestRootCmd_CreateRegistryRegistration(t *testing.T) {
	t.Parallel()
	lazy := &LazyManager{}
	rootCmd := NewRootCmd(lazy, nil, os.Stderr)

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == CreateRegistryCmdName {
			found = true
			break
		}
	}
	assert.True(t, found, CreateRegistryCmdName+" command should be registered")
}

// TestPersistentPreRunE_CreateRegistry_SkipsInitialisation verifies that create-registry skips registry init.
func TestPersistentPreRunE_CreateRegistry_SkipsInitialisation(t *testing.T) {
	t.Parallel()
	lazy := &LazyManager{}
	rootCmd := NewRootCmd(lazy, nil, os.Stderr)

	// Find the create-registry command
	var createRegistryCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == CreateRegistryCmdName {
			createRegistryCmd = cmd
			break
		}
	}
	assert.NotNil(t, createRegistryCmd)

	// Call PersistentPreRunE
	if rootCmd.PersistentPreRunE != nil {
		err := rootCmd.PersistentPreRunE(createRegistryCmd, []string{"some-path"})
		require.NoError(t, err)
	}

	// Verify that registry was NOT initialised (lazy manager remains empty)
	assert.False(t, lazy.HasInner())
}

//nolint:paralleltest // This test modifies a global variable
func TestAddEnvironmentVariableInstructionsForOS(t *testing.T) {
	dir := "/tmp/bg-registry"

	t.Run("windows", func(t *testing.T) { //nolint:paralleltest // shared global state
		got := addEnvironmentVariableInstructionsForOS(dir, "windows")
		assert.Contains(t, got, "setx")
		assert.Contains(t, got, "JSM_ROOT_DIRECTORY")
	})

	t.Run("darwin", func(t *testing.T) { //nolint:paralleltest // shared global state
		got := addEnvironmentVariableInstructionsForOS(dir, "darwin")
		assert.Contains(t, got, "echo")
		assert.Contains(t, got, "&& source")
		assert.Contains(t, got, ".zshrc")
		assert.Contains(t, got, "JSM_ROOT_DIRECTORY")
	})

	t.Run("linux", func(t *testing.T) { //nolint:paralleltest // shared global state
		got := addEnvironmentVariableInstructionsForOS(dir, "linux")
		assert.Contains(t, got, "echo")
		assert.Contains(t, got, "&& source")
		assert.Contains(t, got, ".bashrc")
		assert.Contains(t, got, "JSM_ROOT_DIRECTORY")
	})

	t.Run("abs-error", func(t *testing.T) { //nolint:paralleltest // shared global state
		oldAbs := absFunc
		defer func() { absFunc = oldAbs }()

		absFunc = func(_ string) (string, error) {
			return "", errors.New("mock-error")
		}

		got := addEnvironmentVariableInstructionsForOS(dir, "linux")
		assert.Contains(t, got, dir)
	})
}
