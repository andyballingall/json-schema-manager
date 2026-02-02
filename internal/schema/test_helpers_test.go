package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andyballingall/json-schema-manager/internal/config"
	"github.com/andyballingall/json-schema-manager/internal/validator"
)

// mockValidator is a test implementation of validator.Validator.
type mockValidator struct {
	Err          error
	ValidateFunc func(validator.JSONDocument) error
}

func (m *mockValidator) Validate(doc validator.JSONDocument) error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(doc)
	}
	return m.Err
}

// mockCompiler is a test implementation of validator.Compiler.
type mockCompiler struct {
	CompileFunc func(id string) (validator.Validator, error)
	Supported   []validator.Draft
}

func (m *mockCompiler) AddSchema(_ string, _ validator.JSONSchema) error {
	return nil
}

func (m *mockCompiler) Compile(id string) (validator.Validator, error) {
	if m.CompileFunc != nil {
		return m.CompileFunc(id)
	}
	return &mockValidator{}, nil
}

func (m *mockCompiler) SupportedSchemaVersions() []validator.Draft {
	if m.Supported != nil {
		return m.Supported
	}
	return []validator.Draft{validator.Draft7}
}

const testConfigData = `
environments:
  prod:
    privateUrlRoot: "https://json-schemas.internal.myorg.io/"
    publicUrlRoot: "https://json-schemas.myorg.io/"
    isProduction: true
`

func setupTestRegistry(t *testing.T) *Registry {
	t.Helper()
	regDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(regDir, config.JsmRegistryConfigFile),
		[]byte(testConfigData),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	compiler := &mockCompiler{}
	r, err := NewRegistry(regDir, compiler)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// schemaMap maps schema Keys to their JSON content for specifying schemas in tests.
type schemaMap map[Key]string

// createSchemaFiles is a helper func to create schema files in the temp directory for testing.
// The function takes a map where the key is the schema Key and the value is the content of the schema file.
func createSchemaFiles(t *testing.T, r *Registry, schemas schemaMap) {
	t.Helper()
	for key, content := range schemas {
		content = strings.ReplaceAll(content, "%%", "`")
		schemas[key] = content

		s := New(key, r)
		homeDir := s.Path(HomeDir)

		if err := os.MkdirAll(homeDir, 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(s.Path(FilePath), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
