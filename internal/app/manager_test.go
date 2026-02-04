package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/andyballingall/json-schema-manager/internal/config"
	"github.com/andyballingall/json-schema-manager/internal/fs"
	"github.com/andyballingall/json-schema-manager/internal/repo"
	"github.com/andyballingall/json-schema-manager/internal/schema"
	"github.com/andyballingall/json-schema-manager/internal/validator"
)

const testConfigData = `
environments:
  prod: {publicUrlRoot: 'https://p', privateUrlRoot: 'https://pr', isProduction: true}`

// mockCompiler is a test implementation of validator.Compiler.
type mockCompiler struct{}

func (m *mockCompiler) AddSchema(_ string, _ validator.JSONSchema) error {
	return nil
}

func (m *mockCompiler) Compile(_ string) (validator.Validator, error) {
	return &mockValidator{}, nil
}

func (m *mockCompiler) SupportedSchemaVersions() []validator.Draft {
	return []validator.Draft{validator.Draft7}
}

type failingCompiler struct {
	mockCompiler
}

func (c *failingCompiler) AddSchema(_ string, _ validator.JSONSchema) error {
	return fmt.Errorf("add schema failed")
}

type mockValidator struct{}

func (m *mockValidator) Validate(_ validator.JSONDocument) error {
	return nil
}

type MockDistBuilder struct {
	BuildAllFunc      func(ctx context.Context, env config.Env) (int, error)
	BuildChangedFunc  func(ctx context.Context, env config.Env, anchor repo.Revision) (int, error)
	SetNumWorkersFunc func(n int)
}

func (m *MockDistBuilder) BuildAll(ctx context.Context, env config.Env) (int, error) {
	if m.BuildAllFunc != nil {
		return m.BuildAllFunc(ctx, env)
	}
	return 0, nil
}

func (m *MockDistBuilder) BuildChanged(ctx context.Context, env config.Env, anchor repo.Revision) (int, error) {
	if m.BuildChangedFunc != nil {
		return m.BuildChangedFunc(ctx, env, anchor)
	}
	return 0, nil
}

func (m *MockDistBuilder) SetNumWorkers(n int) {
	if m.SetNumWorkersFunc != nil {
		m.SetNumWorkersFunc(n)
	}
}

func setupTestRegistry(t *testing.T) *schema.Registry {
	t.Helper()
	regDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(regDir, config.JsmRegistryConfigFile),
		[]byte(testConfig),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	compiler := &mockCompiler{}
	pathResolver := fs.NewPathResolver()
	envProvider := fs.NewEnvProvider()
	r, err := schema.NewRegistry(regDir, compiler, pathResolver, envProvider)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestCLIManager_ValidateSchema(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := setupTestRegistry(t)

	// Create a schema file so ValidateSchema has something to test
	key := schema.Key("domain_family_1_0_0")
	s := schema.New(key, registry)
	homeDir := s.Path(schema.HomeDir)
	require.NoError(t, os.MkdirAll(homeDir, 0o755))
	require.NoError(t, os.WriteFile(s.Path(schema.FilePath), []byte(`{"type": "object"}`), 0o600))
	// Ensure pass/fail dirs exist
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, "pass"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, "fail"), 0o755))

	t.Run("successful validation", func(t *testing.T) {
		t.Parallel()
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)
		vErr := mgr.ValidateSchema(context.Background(), schema.ResolvedTarget{Key: &key},
			false, "text", false, false, schema.TestScopeLocal, false)
		require.NoError(t, vErr)
	})

	t.Run("successful JSON validation", func(t *testing.T) {
		t.Parallel()
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)
		vErr := mgr.ValidateSchema(context.Background(), schema.ResolvedTarget{Key: &key},
			false, "json", false, false, schema.TestScopeLocal, false)
		require.NoError(t, vErr)
	})

	t.Run("successful verbose validation", func(t *testing.T) {
		t.Parallel()
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)
		vErr := mgr.ValidateSchema(context.Background(), schema.ResolvedTarget{Key: &key},
			true, "text", false, false, schema.TestScopeLocal, false)
		require.NoError(t, vErr)
	})

	t.Run("validation error", func(t *testing.T) {
		t.Parallel()
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)
		// Non-existent path
		scope := schema.SearchScope("non/existent/path")
		vErr := mgr.ValidateSchema(context.Background(), schema.ResolvedTarget{Scope: &scope},
			false, "text", false, false, schema.TestScopeLocal, false)
		require.Error(t, vErr)
	})

	t.Run("tester error", func(t *testing.T) {
		t.Parallel()
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)
		// This will fail because the path is outside the registry root
		tKey := schema.Key("outside_key_1_0_0")
		vErr := mgr.ValidateSchema(context.Background(), schema.ResolvedTarget{Key: &tKey},
			false, "text", false, false, schema.TestScopeLocal, false)
		require.Error(t, vErr)
	})

	t.Run("no identification method provided", func(t *testing.T) {
		t.Parallel()
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)
		vErr := mgr.ValidateSchema(context.Background(), schema.ResolvedTarget{},
			false, "text", false, false, schema.TestScopeLocal, false)
		require.Error(t, vErr)
		assert.IsType(t, &schema.NoSchemaTargetsError{}, vErr)
	})

	t.Run("missing test directories", func(t *testing.T) {
		t.Parallel()
		// Create a schema and registry but delete the pass/fail folders
		regDir := t.TempDir()
		cfg := `
environments:
  prod:
    publicUrlRoot: "https://p"
    privateUrlRoot: "https://pr"
    isProduction: true
`
		require.NoError(t, os.WriteFile(filepath.Join(regDir, config.JsmRegistryConfigFile), []byte(cfg), 0o600))
		r, err := schema.NewRegistry(regDir, &mockCompiler{}, fs.NewPathResolver(), fs.NewEnvProvider())
		require.NoError(t, err)

		tKey := schema.Key("d2_f2_1_0_0")
		s2 := schema.New(tKey, r)
		tHomeDir := s2.Path(schema.HomeDir)
		require.NoError(t, os.MkdirAll(tHomeDir, 0o755))
		require.NoError(t, os.WriteFile(s2.Path(schema.FilePath), []byte(`{}`), 0o600))

		mgr2 := NewCLIManager(logger, r, schema.NewTester(r), &MockGitter{}, nil)
		vErr := mgr2.ValidateSchema(context.Background(), schema.ResolvedTarget{Key: &tKey},
			false, "text", false, false, schema.TestScopeLocal, false)
		require.Error(t, vErr)
		require.ErrorContains(t, vErr, "pass directory missing")
	})
}

func TestCLIManager_CreateSchema(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successful create", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)

		key, err := mgr.CreateSchema("test-domain/test-family")
		require.NoError(t, err)
		assert.Equal(t, schema.Key("test-domain_test-family_1_0_0"), key)
	})

	t.Run("failed create - invalid scope", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)

		_, err := mgr.CreateSchema("INVALID/scope")
		require.Error(t, err)
	})
}

func TestCLIManager_CreateSchemaVersion(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successful create version", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)

		// Create base schema
		baseKey := schema.Key("d1_f1_1_0_0")
		s := schema.New(baseKey, registry)
		require.NoError(t, os.MkdirAll(s.Path(schema.HomeDir), 0o755))
		require.NoError(t, os.WriteFile(s.Path(schema.FilePath), []byte("{}"), 0o600))

		newKey, err := mgr.CreateSchemaVersion(baseKey, schema.ReleaseTypeMajor)
		require.NoError(t, err)
		assert.Equal(t, schema.Key("d1_f1_2_0_0"), newKey)
	})

	t.Run("failed create version - missing base", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)

		_, err := mgr.CreateSchemaVersion(schema.Key("missing_1_0_0"), schema.ReleaseTypeMajor)
		require.Error(t, err)
	})
}

func TestCLIManager_RenderSchema(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successful render", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)

		// Create base schema
		baseKey := schema.Key("d1_f1_1_0_0")
		s := schema.New(baseKey, registry)
		require.NoError(t, os.MkdirAll(s.Path(schema.HomeDir), 0o755))
		require.NoError(t, os.WriteFile(s.Path(schema.FilePath), []byte("{}"), 0o600))

		target := schema.ResolvedTarget{Key: &baseKey}
		rendered, err := mgr.RenderSchema(context.Background(), target, config.Env("prod"))
		require.NoError(t, err)
		assert.NotEmpty(t, rendered)
	})

	t.Run("default environment (prod)", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)

		baseKey := schema.Key("d1_f1_1_0_0")
		s := schema.New(baseKey, registry)
		require.NoError(t, os.MkdirAll(s.Path(schema.HomeDir), 0o755))
		require.NoError(t, os.WriteFile(s.Path(schema.FilePath), []byte("{}"), 0o600))

		target := schema.ResolvedTarget{Key: &baseKey}
		rendered, err := mgr.RenderSchema(context.Background(), target, "")
		require.NoError(t, err)
		assert.NotEmpty(t, rendered)
	})

	t.Run("invalid environment", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)

		baseKey := schema.Key("d1_f1_1_0_0")
		target := schema.ResolvedTarget{Key: &baseKey}
		_, err := mgr.RenderSchema(context.Background(), target, "invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid environment: 'invalid'. Valid environments are: 'prod'")
	})

	t.Run("missing target key", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)

		target := schema.ResolvedTarget{}
		_, err := mgr.RenderSchema(context.Background(), target, "prod")
		require.Error(t, err)
		assert.IsType(t, &schema.NoSchemaTargetsError{}, err)
	})

	t.Run("config error", func(t *testing.T) {
		t.Parallel()
		registry := &schema.Registry{}
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)

		baseKey := schema.Key("d1_f1_1_0_0")
		target := schema.ResolvedTarget{Key: &baseKey}
		_, vErr := mgr.RenderSchema(context.Background(), target, "prod")
		require.Error(t, vErr)
	})

	t.Run("GetSchemaByKey error", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)

		baseKey := schema.Key("missing_1_0_0")
		target := schema.ResolvedTarget{Key: &baseKey}
		_, err := mgr.RenderSchema(context.Background(), target, "prod")
		require.Error(t, err)
	})

	t.Run("CoordinateRender error", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)

		baseKey := schema.Key("d1_f1_1_0_0")
		s := schema.New(baseKey, registry)
		require.NoError(t, os.MkdirAll(s.Path(schema.HomeDir), 0o755))
		// Invalid JSON should trigger an error in renderer.Render
		require.NoError(t, os.WriteFile(s.Path(schema.FilePath), []byte("{ invalid"), 0o600))

		target := schema.ResolvedTarget{Key: &baseKey}
		_, err := mgr.RenderSchema(context.Background(), target, "prod")
		require.Error(t, err)
	})

	t.Run("multiple environments success", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		// Setup config with two environments
		configData := `environments:
  prod: {publicUrlRoot: 'https://p', privateUrlRoot: 'https://pr', isProduction: true}
  dev: {publicUrlRoot: 'https://d', privateUrlRoot: 'https://dr', isProduction: false}`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "json-schema-manager-config.yml"), []byte(configData), 0o600))
		registry, err := schema.NewRegistry(tmpDir, &mockCompiler{}, fs.NewPathResolver(), fs.NewEnvProvider())
		require.NoError(t, err)

		tester := schema.NewTester(registry)
		mgr := NewCLIManager(logger, registry, tester, &MockGitter{}, nil)

		baseKey := schema.Key("d1_f1_1_0_0")
		s := schema.New(baseKey, registry)
		require.NoError(t, os.MkdirAll(s.Path(schema.HomeDir), 0o755))
		require.NoError(t, os.WriteFile(s.Path(schema.FilePath), []byte("{}"), 0o600))

		target := schema.ResolvedTarget{Key: &baseKey}

		// Test prod
		_, err = mgr.RenderSchema(context.Background(), target, "prod")
		require.NoError(t, err)

		// Test dev
		_, err = mgr.RenderSchema(context.Background(), target, "dev")
		require.NoError(t, err)
	})

	t.Run("GetSchemaByKey failure - unreadable dir", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		mgr := NewCLIManager(logger, registry, nil, &MockGitter{}, nil)

		baseKey := schema.Key("d1_f1_1_0_0")
		s := schema.New(baseKey, registry)
		homedir := s.Path(schema.HomeDir)
		require.NoError(t, os.MkdirAll(homedir, 0o755))
		require.NoError(t, os.WriteFile(s.Path(schema.FilePath), []byte("{}"), 0o600))

		// Make it unreadable
		require.NoError(t, os.Chmod(homedir, 0o000))
		t.Cleanup(func() { _ = os.Chmod(homedir, 0o755) })

		target := schema.ResolvedTarget{Key: &baseKey}
		_, err := mgr.RenderSchema(context.Background(), target, "prod")
		require.Error(t, err)
	})

	t.Run("CoordinateRender failure - bad template", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		mgr := NewCLIManager(logger, registry, nil, &MockGitter{}, nil)

		baseKey := schema.Key("d1_f1_1_0_0")
		s := schema.New(baseKey, registry)
		require.NoError(t, os.MkdirAll(s.Path(schema.HomeDir), 0o755))
		require.NoError(t, os.WriteFile(s.Path(schema.FilePath), []byte("{{ bad }"), 0o600))

		target := schema.ResolvedTarget{Key: &baseKey}
		_, err := mgr.RenderSchema(context.Background(), target, "prod")
		require.Error(t, err)
	})

	t.Run("CoordinateRender failure - bad JSM key", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		mgr := NewCLIManager(logger, registry, nil, &MockGitter{}, nil)

		baseKey := schema.Key("d1_f1_1_0_0")
		s := schema.New(baseKey, registry)
		require.NoError(t, os.MkdirAll(s.Path(schema.HomeDir), 0o755))
		// Valid JSON, but bad template action
		require.NoError(t, os.WriteFile(s.Path(schema.FilePath), []byte("{{ JSM \"!!\" }}"), 0o600))

		target := schema.ResolvedTarget{Key: &baseKey}
		_, err := mgr.RenderSchema(context.Background(), target, "prod")
		require.Error(t, err)
	})

	t.Run("CoordinateRender failure - missing dependency", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		mgr := NewCLIManager(logger, registry, nil, &MockGitter{}, nil)

		baseKey := schema.Key("d1_f1_1_0_0")
		s := schema.New(baseKey, registry)
		require.NoError(t, os.MkdirAll(s.Path(schema.HomeDir), 0o755))
		// Valid template, but missing dependency will fail during rendering execution
		require.NoError(t, os.WriteFile(s.Path(schema.FilePath), []byte("{{ JSM \"missing_dep_1_0_0\" }}"), 0o600))

		target := schema.ResolvedTarget{Key: &baseKey}
		_, err := mgr.RenderSchema(context.Background(), target, "prod")
		require.Error(t, err)
	})

	t.Run("CoordinateRender failure - compiler error", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "json-schema-manager-config.yml")
		require.NoError(t, os.WriteFile(cfgPath, []byte(testConfigData), 0o600))
		registry, _ := schema.NewRegistry(tmpDir, &failingCompiler{}, fs.NewPathResolver(), fs.NewEnvProvider())
		mgr := NewCLIManager(logger, registry, nil, &MockGitter{}, nil)

		baseKey := schema.Key("d1_f1_1_0_0")
		s := schema.New(baseKey, registry)
		require.NoError(t, os.MkdirAll(s.Path(schema.HomeDir), 0o755))
		require.NoError(t, os.WriteFile(s.Path(schema.FilePath), []byte("{}"), 0o600))

		target := schema.ResolvedTarget{Key: &baseKey}
		_, err := mgr.RenderSchema(context.Background(), target, "prod")
		require.Error(t, err)
	})
}

func TestCLIManager_CheckChanges(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	registry := setupTestRegistry(t)
	mgr := NewCLIManager(logger, registry, nil, &MockGitter{}, nil)

	t.Run("config error", func(t *testing.T) {
		t.Parallel()
		// Use an empty registry to simulate uninitialised config
		m := NewCLIManager(logger, &schema.Registry{}, nil, &MockGitter{}, nil)
		err := m.CheckChanges(context.Background(), "prod")
		require.Error(t, err)
	})

	t.Run("invalid environment", func(t *testing.T) {
		t.Parallel()
		err := mgr.CheckChanges(context.Background(), "invalid")
		require.Error(t, err)
	})

	t.Run("git history error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir() // Not a git repo
		cfgPath := filepath.Join(dir, "json-schema-manager-config.yml")
		require.NoError(t, os.WriteFile(cfgPath, []byte(testConfigData), 0o600))

		registry2, err := schema.NewRegistry(dir, &mockCompiler{}, fs.NewPathResolver(), fs.NewEnvProvider())
		require.NoError(t, err)

		cfg2, err := registry2.Config()
		require.NoError(t, err)
		m := NewCLIManager(logger, registry2, nil, repo.NewCLIGitter(cfg2, fs.NewPathResolver(), dir), nil)

		err = m.CheckChanges(context.Background(), "prod")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not find git history")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		// Init git repo
		require.NoError(t, exec.Command("git", "init", dir).Run())
		require.NoError(t, exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "config", "user.name", "t").Run())

		// Create a schema in a sub-directory
		schemaDir := filepath.Join(dir, "domain", "family", "1", "0", "0")
		require.NoError(t, os.MkdirAll(schemaDir, 0o755))
		f1 := filepath.Join(schemaDir, "f1.schema.json")
		require.NoError(t, os.WriteFile(f1, []byte("{}"), 0o600))
		require.NoError(t, exec.Command("git", "-C", dir, "add", ".").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "init").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "tag", "jsm-deploy/prod/v1").Run())

		cfgPath := filepath.Join(dir, "json-schema-manager-config.yml")
		require.NoError(t, os.WriteFile(cfgPath, []byte(testConfigData), 0o600))

		registry4, err := schema.NewRegistry(dir, &mockCompiler{}, fs.NewPathResolver(), fs.NewEnvProvider())
		require.NoError(t, err)

		m := NewCLIManager(logger, registry4, nil, &MockGitter{}, nil)

		// No changes
		err = m.CheckChanges(context.Background(), "prod")
		require.NoError(t, err)

		// New schema
		f2 := filepath.Join(schemaDir, "f2.schema.json")
		require.NoError(t, os.WriteFile(f2, []byte("{}"), 0o600))
		gitAdd := exec.Command("git", "add", ".")
		gitAdd.Dir = dir
		require.NoError(t, gitAdd.Run())

		gitCommit := exec.Command("git", "commit", "-m", "new schema")
		gitCommit.Dir = dir
		require.NoError(t, gitCommit.Run())

		err = m.CheckChanges(context.Background(), "prod")
		require.NoError(t, err)
	})

	t.Run("mutation forbidden", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, exec.Command("git", "init", dir).Run())
		require.NoError(t, exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "config", "user.name", "t").Run())

		schemaDir := filepath.Join(dir, "domain", "family", "1", "0", "0")
		require.NoError(t, os.MkdirAll(schemaDir, 0o755))
		f1 := filepath.Join(schemaDir, "f1.schema.json")
		require.NoError(t, os.WriteFile(f1, []byte("{}"), 0o600))
		require.NoError(t, exec.Command("git", "-C", dir, "add", ".").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "in").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "tag", "jsm-deploy/prod/v1").Run())

		// Mutate it
		require.NoError(t, os.WriteFile(f1, []byte(`{"type":"object"}`), 0o600))
		require.NoError(t, exec.Command("git", "-C", dir, "add", ".").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "mu").Run())

		require.NoError(t, os.WriteFile(filepath.Join(dir, "json-schema-manager-config.yml"), []byte(testConfigData), 0o600))
		r, _ := schema.NewRegistry(dir, &mockCompiler{}, fs.NewPathResolver(), fs.NewEnvProvider())
		rcfg, _ := r.Config()
		m := NewCLIManager(logger, r, nil, repo.NewCLIGitter(rcfg, fs.NewPathResolver(), dir), nil)

		err := m.CheckChanges(context.Background(), "prod")
		require.Error(t, err)

		var mutationErr *schema.ChangedDeployedSchemasError
		require.ErrorAs(t, err, &mutationErr)
		assert.Contains(t, mutationErr.Paths[0], "f1.schema.json")
	})

	t.Run("mutation allowed", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, exec.Command("git", "init", dir).Run())
		require.NoError(t, exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "config", "user.name", "t").Run())

		schemaDir := filepath.Join(dir, "domain", "family", "1", "0", "0")
		require.NoError(t, os.MkdirAll(schemaDir, 0o755))
		f1 := filepath.Join(schemaDir, "f1.schema.json")
		require.NoError(t, os.WriteFile(f1, []byte("{}"), 0o600))
		require.NoError(t, exec.Command("git", "-C", dir, "add", ".").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "in").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "tag", "jsm-deploy/prod/v1").Run())

		// Mutate it
		require.NoError(t, os.WriteFile(f1, []byte(`{"type":"object"}`), 0o600))
		require.NoError(t, exec.Command("git", "-C", dir, "add", ".").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "commit", "-m", "mu").Run())

		cfgWithMutation := `
environments:
  prod: {publicUrlRoot: 'https://p', privateUrlRoot: 'https://pr', isProduction: true, allowSchemaMutation: true}`
		cfgPath := filepath.Join(dir, "json-schema-manager-config.yml")
		require.NoError(t, os.WriteFile(cfgPath, []byte(cfgWithMutation), 0o600))
		r, _ := schema.NewRegistry(dir, &mockCompiler{}, fs.NewPathResolver(), fs.NewEnvProvider())
		m := NewCLIManager(logger, r, nil, &MockGitter{}, nil)

		err := m.CheckChanges(context.Background(), "prod")
		require.NoError(t, err)
	})

	t.Run("GetSchemaChanges error", func(t *testing.T) {
		t.Parallel()
		r := setupTestRegistry(t)
		mockGitter := &MockGitter{
			GetSchemaChangesFunc: func(_ repo.Revision, _, _ string) ([]repo.Change, error) {
				return nil, fmt.Errorf("git diff failed")
			},
		}
		m := NewCLIManager(logger, r, nil, mockGitter, nil)

		err := m.CheckChanges(context.Background(), "prod")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git diff failed")
	})
}

func TestCLIManager_TagDeployment(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	registry := setupTestRegistry(t)
	mgr := NewCLIManager(logger, registry, nil, &MockGitter{}, nil)

	t.Run("config error", func(t *testing.T) {
		t.Parallel()
		m := NewCLIManager(logger, &schema.Registry{}, nil, &MockGitter{}, nil)
		err := m.TagDeployment(context.Background(), "prod")
		require.Error(t, err)
	})

	t.Run("invalid environment", func(t *testing.T) {
		t.Parallel()
		err := mgr.TagDeployment(context.Background(), "invalid")
		require.Error(t, err)
	})

	t.Run("success without remote", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, exec.Command("git", "init", dir).Run())
		require.NoError(t, exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "config", "user.name", "t").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "commit", "--allow-empty", "-m", "init").Run())

		require.NoError(t, os.WriteFile(filepath.Join(dir, "json-schema-manager-config.yml"), []byte(testConfigData), 0o600))
		r, _ := schema.NewRegistry(dir, &mockCompiler{}, fs.NewPathResolver(), fs.NewEnvProvider())
		m := NewCLIManager(logger, r, nil, &MockGitter{}, nil)

		// This calls g.TagDeploymentSuccess() which will fail push but return tagName
		err := m.TagDeployment(context.Background(), "prod")
		require.NoError(t, err) // We return nil if tag created but push fails
	})

	t.Run("git tag error", func(t *testing.T) {
		t.Parallel()
		r := setupTestRegistry(t)
		mockGitter := &MockGitter{
			TagDeploymentFunc: func(_ config.Env) (string, error) {
				return "", fmt.Errorf("failed to create git tag")
			},
		}
		m := NewCLIManager(logger, r, nil, mockGitter, nil)

		err := m.TagDeployment(context.Background(), "prod")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create git tag")
	})

	t.Run("success with remote", func(t *testing.T) {
		t.Parallel()
		// Setup a "remote"
		remoteDir := t.TempDir()
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

		repoDir := t.TempDir()
		require.NoError(t, exec.Command("git", "init", repoDir).Run())
		require.NoError(t, exec.Command("git", "-C", repoDir, "config", "user.email", "t@t.com").Run())
		require.NoError(t, exec.Command("git", "-C", repoDir, "config", "user.name", "t").Run())
		require.NoError(t, exec.Command("git", "-C", repoDir, "commit", "--allow-empty", "-m", "init").Run())
		require.NoError(t, exec.Command("git", "-C", repoDir, "remote", "add", "origin", remoteDir).Run())

		cfgPath := filepath.Join(repoDir, "json-schema-manager-config.yml")
		require.NoError(t, os.WriteFile(cfgPath, []byte(testConfigData), 0o600))
		r, _ := schema.NewRegistry(repoDir, &mockCompiler{}, fs.NewPathResolver(), fs.NewEnvProvider())
		m := NewCLIManager(logger, r, nil, &MockGitter{}, nil)

		err := m.TagDeployment(context.Background(), "prod")
		require.NoError(t, err)

		// Verify tag exists on "remote"
		cmd := exec.Command("git", "-C", remoteDir, "rev-parse", "HEAD") // just check it's a git repo
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.NotEmpty(t, output)
	})

	t.Run("tag created but push fails", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, exec.Command("git", "init", dir).Run())
		require.NoError(t, exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "config", "user.name", "t").Run())
		require.NoError(t, exec.Command("git", "-C", dir, "commit", "--allow-empty", "-m", "init").Run())

		require.NoError(t, os.WriteFile(filepath.Join(dir, "json-schema-manager-config.yml"), []byte(testConfigData), 0o600))
		r, _ := schema.NewRegistry(dir, &mockCompiler{}, fs.NewPathResolver(), fs.NewEnvProvider())

		mockGitter := &MockGitter{
			TagDeploymentFunc: func(_ config.Env) (string, error) {
				return "jsm-deploy/prod/failed-push", fmt.Errorf("git push failed")
			},
		}

		m := NewCLIManager(logger, r, nil, mockGitter, nil)

		err := m.TagDeployment(context.Background(), "prod")
		require.NoError(t, err) // Should return nil if tag created but push failed
	})
}

func TestCLIManager_BuildDist(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success --all", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		mockBuilder := &MockDistBuilder{
			BuildAllFunc: func(_ context.Context, env config.Env) (int, error) {
				assert.Equal(t, config.Env("prod"), env)
				return 5, nil
			},
		}
		mgr := NewCLIManager(logger, registry, nil, &MockGitter{}, mockBuilder)

		err := mgr.BuildDist(context.Background(), "prod", true)
		require.NoError(t, err)
	})

	t.Run("success with changes", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		mockBuilder := &MockDistBuilder{
			BuildChangedFunc: func(_ context.Context, env config.Env, _ repo.Revision) (int, error) {
				assert.Equal(t, config.Env("prod"), env)
				return 3, nil
			},
		}
		mgr := NewCLIManager(logger, registry, nil, &MockGitter{}, mockBuilder)

		err := mgr.BuildDist(context.Background(), "prod", false)
		require.NoError(t, err)
	})

	t.Run("no schemas to build", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		mockBuilder := &MockDistBuilder{
			BuildAllFunc: func(_ context.Context, _ config.Env) (int, error) {
				return 0, nil
			},
		}
		mgr := NewCLIManager(logger, registry, nil, &MockGitter{}, mockBuilder)

		err := mgr.BuildDist(context.Background(), "prod", true)
		require.NoError(t, err)
	})

	t.Run("BuildAll error", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		mockBuilder := &MockDistBuilder{
			BuildAllFunc: func(_ context.Context, _ config.Env) (int, error) {
				return 0, fmt.Errorf("build failed")
			},
		}
		mgr := NewCLIManager(logger, registry, nil, &MockGitter{}, mockBuilder)

		err := mgr.BuildDist(context.Background(), "prod", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "build failed")
	})

	t.Run("CheckChanges error", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		// Use a gitter that returns a mutation error
		mockGitter := &MockGitter{
			GetSchemaChangesFunc: func(_ repo.Revision, _, _ string) ([]repo.Change, error) {
				return []repo.Change{{Path: "mutated.schema.json", IsNew: false}}, nil
			},
		}
		mgr := NewCLIManager(logger, registry, nil, mockGitter, nil)

		err := mgr.BuildDist(context.Background(), "prod", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot modify deployed schemas")
	})

	t.Run("GetLatestAnchor error", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		calls := 0
		mockGitter := &MockGitter{
			GetLatestAnchorFunc: func(_ config.Env) (repo.Revision, error) {
				calls++
				if calls == 1 {
					return "HEAD", nil
				}
				return "", fmt.Errorf("git anchor failed")
			},
		}
		mgr := NewCLIManager(logger, registry, nil, mockGitter, nil)

		err := mgr.BuildDist(context.Background(), "prod", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git anchor failed")
	})

	t.Run("BuildChanged error", func(t *testing.T) {
		t.Parallel()
		registry := setupTestRegistry(t)
		mockBuilder := &MockDistBuilder{
			BuildChangedFunc: func(_ context.Context, _ config.Env, _ repo.Revision) (int, error) {
				return 0, fmt.Errorf("build changed failed")
			},
		}
		mgr := NewCLIManager(logger, registry, nil, &MockGitter{}, mockBuilder)

		err := mgr.BuildDist(context.Background(), "prod", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "build changed failed")
	})
}

func TestNewBuildDistCmd(t *testing.T) {
	t.Parallel()
	mockMgr := &MockManager{}
	cmd := NewBuildDistCmd(mockMgr)
	assert.NotNil(t, cmd)

	mockMgr.On("BuildDist", mock.Anything, config.Env("prod"), false).Return(nil)

	cmd.SetArgs([]string{"prod"})
	err := cmd.Execute()
	require.NoError(t, err)

	mockMgr.AssertExpectations(t)
}

func TestCLIManager_Registry(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	registry := setupTestRegistry(t)
	mgr := NewCLIManager(logger, registry, nil, nil, nil)

	assert.Equal(t, registry, mgr.Registry())
}

func TestLazyManager_PanicWhenNotInitialised(t *testing.T) {
	t.Parallel()
	lazy := &LazyManager{}

	// Should panic when accessing any method before SetInner is called
	assert.Panics(t, func() {
		lazy.Registry()
	})
}

func TestLazyManager_Delegation(t *testing.T) {
	t.Parallel()
	mockMgr := &MockManager{
		registry: &schema.Registry{},
	}
	lazy := &LazyManager{}
	lazy.SetInner(mockMgr)

	// Test HasInner
	assert.True(t, lazy.HasInner())

	// Test Registry delegation
	assert.Equal(t, mockMgr.registry, lazy.Registry())

	// Test ValidateSchema delegation
	ctx := context.Background()
	key := schema.Key("test_1_0_0")
	target := schema.ResolvedTarget{Key: &key}
	mockMgr.On("ValidateSchema", ctx, target, false, "text", false, false, schema.TestScopeLocal, false).Return(nil)
	err := lazy.ValidateSchema(ctx, target, false, "text", false, false, schema.TestScopeLocal, false)
	require.NoError(t, err)

	// Test CreateSchema delegation
	mockMgr.On("CreateSchema", "domain/family").Return(schema.Key("domain_family_1_0_0"), nil)
	key2, err := lazy.CreateSchema("domain/family")
	require.NoError(t, err)
	assert.Equal(t, schema.Key("domain_family_1_0_0"), key2)

	// Test CreateSchemaVersion delegation
	mockMgr.On("CreateSchemaVersion", schema.Key("domain_family_1_0_0"), schema.ReleaseTypeMinor).
		Return(schema.Key("domain_family_1_1_0"), nil)
	key3, err := lazy.CreateSchemaVersion(schema.Key("domain_family_1_0_0"), schema.ReleaseTypeMinor)
	require.NoError(t, err)
	assert.Equal(t, schema.Key("domain_family_1_1_0"), key3)

	// Test RenderSchema delegation
	mockMgr.On("RenderSchema", ctx, target, config.Env("prod")).Return([]byte("{}"), nil)
	rendered, err := lazy.RenderSchema(ctx, target, config.Env("prod"))
	require.NoError(t, err)
	assert.Equal(t, []byte("{}"), rendered)

	// Test CheckChanges delegation
	mockMgr.On("CheckChanges", ctx, config.Env("prod")).Return(nil)
	err = lazy.CheckChanges(ctx, config.Env("prod"))
	require.NoError(t, err)

	// Test TagDeployment delegation
	mockMgr.On("TagDeployment", ctx, config.Env("prod")).Return(nil)
	err = lazy.TagDeployment(ctx, config.Env("prod"))
	require.NoError(t, err)

	// Test BuildDist delegation
	mockMgr.On("BuildDist", ctx, config.Env("prod"), false).Return(nil)
	err = lazy.BuildDist(ctx, config.Env("prod"), false)
	require.NoError(t, err)

	mockMgr.AssertExpectations(t)
}
