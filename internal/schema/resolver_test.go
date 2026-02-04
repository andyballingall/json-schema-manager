package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetResolver(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	// Create a dummy schema file for path resolution tests
	k := Key("domain_family_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k: `{"type": "object"}`,
	})
	s, _ := r.GetSchemaByKey(k)
	schemaPath := s.Path(FilePath)
	schemaDir := s.Path(HomeDir)

	// Additional files for error cases
	badFile := filepath.Join(r.rootDirectory, "bad.txt")
	require.NoError(t, os.WriteFile(badFile, []byte(`{}`), 0o600))

	outsideFile := filepath.Join(t.TempDir(), "outside.schema.json")
	require.NoError(t, os.WriteFile(outsideFile, []byte(`{}`), 0o600))

	notASchemaFile := filepath.Join(t.TempDir(), "not-a-schema.txt")
	require.NoError(t, os.WriteFile(notASchemaFile, []byte(`{}`), 0o600))

	outsideDir := t.TempDir()

	tests := []struct {
		name        string
		arg         string
		setup       func(ar *TargetResolver)
		wantKey     Key
		wantScope   SearchScope
		wantErr     bool
		wantErrType interface{}
	}{
		{
			name: "explicit key",
			setup: func(ar *TargetResolver) {
				ar.SetKey("explicit_key_1_0_0")
			},
			wantKey: "explicit_key_1_0_0",
		},
		{
			name: "explicit ID",
			setup: func(ar *TargetResolver) {
				ar.SetID("https://example.com/explicit_id_1_0_0.schema.json")
			},
			wantKey: "explicit_id_1_0_0",
		},
		{
			name: "explicit scope",
			setup: func(ar *TargetResolver) {
				ar.SetScope("explicit/scope")
			},
			wantScope: "explicit/scope",
		},
		{
			name:    "guess ID from URL",
			arg:     "https://example.com/guess_id_1_0_0.schema.json",
			wantKey: "guess_id_1_0_0",
		},
		{
			name:    "guess Key from string with underscore",
			arg:     "guess_key_1_0_0",
			wantKey: "guess_key_1_0_0",
		},
		{
			name:      "guess Scope from string without underscore",
			arg:       "guess/scope",
			wantScope: "guess/scope",
		},
		{
			name:    "resolve file path",
			arg:     schemaPath,
			wantKey: k,
		},
		{
			name:      "resolve directory path",
			arg:       schemaDir,
			wantScope: "domain/family/1/0/0",
		},
		{
			name:      "resolve registry root directory path",
			arg:       r.rootDirectory,
			wantScope: "",
		},
		{
			name:        "error resolving file outside root",
			arg:         outsideFile,
			wantErr:     true,
			wantErrType: &LocationOutsideRootDirectoryError{},
		},
		{
			name:        "error resolving non-schema file",
			arg:         badFile,
			wantErr:     true,
			wantErrType: &NotASchemaFileError{},
		},
		{
			name:    "invalid URL",
			arg:     "https://:invalid",
			wantErr: true,
		},
		{
			name:        "URL with invalid suffix",
			arg:         "https://example.com/bad.txt",
			wantErr:     true,
			wantErrType: &NotASchemaFileError{},
		},
		{
			name: "explicit ID with invalid suffix",
			setup: func(ar *TargetResolver) {
				ar.SetID("https://example.com/bad.txt")
			},
			wantErr:     true,
			wantErrType: &NotASchemaFileError{},
		},
		{
			name: "explicit empty ID",
			setup: func(ar *TargetResolver) {
				ar.SetID("")
			},
			wantErr:     true,
			wantErrType: &NoSchemaTargetsError{},
		},
		{
			name: "invalid explicit ID",
			setup: func(ar *TargetResolver) {
				ar.SetID("https://:invalid")
			},
			wantErr: true,
		},
		{
			name: "empty explicit ID",
			setup: func(ar *TargetResolver) {
				ar.SetID("")
			},
			wantErr: true,
		},
		{
			name: "invalid explicit key",
			setup: func(ar *TargetResolver) {
				ar.SetKey("invalid_key")
			},
			wantErr: true,
		},
		{
			name: "invalid explicit scope",
			setup: func(ar *TargetResolver) {
				ar.SetScope("Invalid Scope!")
			},
			wantErr: true,
		},
		{
			name: "empty explicit scope",
			setup: func(ar *TargetResolver) {
				ar.SetScope("")
			},
			wantErr: true,
		},
		{
			name:    "invalid URL arg",
			arg:     "https://example.com/not-a-schema.txt",
			wantErr: true,
		},
		{
			name:    "invalid key-like arg",
			arg:     "not_a_key",
			wantErr: true,
		},
		{
			name:    "invalid scope-like arg",
			arg:     "Invalid Scope!",
			wantErr: true,
		},
		{
			name:        "resolve directory outside root",
			arg:         outsideDir,
			wantErr:     true,
			wantErrType: &LocationOutsideRootDirectoryError{},
		},
		{
			name:      "all schemas",
			arg:       "all",
			wantScope: "",
		},
		{
			name:        "no targets",
			arg:         "",
			wantErr:     true,
			wantErrType: &NoSchemaTargetsError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ar := NewTargetResolver(r, tt.arg)
			if tt.setup != nil {
				tt.setup(ar)
			}

			res, err := ar.Resolve()

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrType != nil {
					assert.IsType(t, tt.wantErrType, err)
				}
				return
			}

			require.NoError(t, err)
			if tt.wantKey != "" {
				assert.NotNil(t, res.Key)
				assert.Equal(t, tt.wantKey, *res.Key)
			} else {
				assert.Nil(t, res.Key)
			}

			if tt.wantScope != "" || (tt.name == "all schemas" || tt.name == "resolve registry root directory path") {
				assert.NotNil(t, res.Scope)
				assert.Equal(t, tt.wantScope, *res.Scope)
			} else {
				assert.Nil(t, res.Scope)
			}
		})
	}

	t.Run("resolvePathToScope canonicalPath error", func(t *testing.T) {
		t.Parallel()
		ar := NewTargetResolver(r, "")
		_, err := ar.resolvePathToScope("/non/existent/path")
		require.Error(t, err)
	})

	t.Run("resolvePathToScope out of bounds", func(t *testing.T) {
		t.Parallel()
		ar := NewTargetResolver(r, "")
		parentDir := filepath.Dir(r.rootDirectory)
		_, err := ar.resolvePathToScope(parentDir)
		require.Error(t, err)
		assert.IsType(t, &LocationOutsideRootDirectoryError{}, err)
	})
}
