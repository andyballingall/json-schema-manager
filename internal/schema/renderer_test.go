package schema

import (
	"errors"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderer_Render_Success(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	key := Key("domain_family_1_0_0")
	s := New(key, r)
	s.tmpl = loadTemplate(t, `{"id": "{{ ID }}", "type": "object"}`)

	ec := r.config.ProductionEnvConfig()
	renderer := NewRenderer(s, ec)

	rb, js, err := renderer.Render()
	require.NoError(t, err)
	// By default schemas are private, so it should use the privateUrlRoot
	assert.Contains(t, string(rb), "https://json-schemas.internal.myorg.io/domain_family_1_0_0.schema.json")
	assert.NotNil(t, js)
}

func TestRenderer_Render_InvalidJSON(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	// Create a schema that renders to invalid JSON (trailing comma)
	key := Key("domain_family_1_0_0")
	s := New(key, r)

	// But Render() doesn't check if the output is valid JSON until after execution.
	s.tmpl = loadTemplate(t, `{"id": "{{ ID }}", "invalid": }`)

	ec := r.config.ProductionEnvConfig()
	renderer := NewRenderer(s, ec)

	_, _, err := renderer.Render()
	require.Error(t, err)
	var target *InvalidJSONError
	require.ErrorAs(t, err, &target)
}

type errorCloner struct{}

func (e *errorCloner) Clone() (*template.Template, error) {
	return nil, errors.New("simulated clone failure")
}

func TestRenderer_Render_CloneErr(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	key := Key("domain_family_1_0_0")
	s := New(key, r)
	s.tmpl = &errorCloner{}

	ec := r.config.ProductionEnvConfig()
	renderer := NewRenderer(s, ec)

	_, _, err := renderer.Render()
	require.Error(t, err)
	assert.EqualError(t, err, "simulated clone failure")
}

func TestRenderer_Render_TemplateExecutionErr(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	key := Key("domain_family_1_0_0")
	s := New(key, r)
	// Passing an invalid key to JSM will cause an error during execution
	s.tmpl = loadTemplate(t, `{"id": "{{ JSM "invalid" }}"}`)

	ec := r.config.ProductionEnvConfig()
	renderer := NewRenderer(s, ec)

	_, _, err := renderer.Render()
	require.Error(t, err)
	var target *TemplateExecutionFailedError
	require.ErrorAs(t, err, &target)
}

func TestRenderer_JSM_Errs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		arg                string
		setup              func(t *testing.T, r *Registry)
		wantErrMsgContains string
	}{
		{
			name:               "invalid key format",
			arg:                "not-a-valid-key",
			wantErrMsgContains: "has an invalid <schema key>",
		},
		{
			name:               "schema not found",
			arg:                "domain_missing_1_0_0",
			wantErrMsgContains: "could not be loaded",
		},
		{
			name: "dependency render error",
			arg:  "domain_dep_1_0_0",
			setup: func(t *testing.T, r *Registry) {
				t.Helper()
				// Create a dependency schema that has a broken template
				depKey := Key("domain_dep_1_0_0")
				createSchemaFiles(t, r, schemaMap{
					depKey: `{"id": "{{ ID }"}`, // Missing closing brace
				})
			},
			wantErrMsgContains: "has a template syntax error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := setupTestRegistry(t)
			if tt.setup != nil {
				tt.setup(t, r)
			}

			s := New(Key("domain_root_1_0_0"), r)
			ec := r.config.ProductionEnvConfig()
			renderer := NewRenderer(s, ec)

			_, err := renderer.JSM(tt.arg)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantErrMsgContains)
		})
	}
}

func TestRenderer_ID(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)

	key := Key("domain_family_1_0_0")
	s := New(key, r)
	s.isPublic = true // Force public for production to get public URL

	ec := r.config.ProductionEnvConfig()
	renderer := NewRenderer(s, ec)

	id, err := renderer.ID()
	require.NoError(t, err)
	assert.Equal(t, ID("https://json-schemas.myorg.io/domain_family_1_0_0.schema.json"), id)
}

// loadTemplate is a helper to create a template without full Load() cycle.
func loadTemplate(t *testing.T, content string) *template.Template {
	t.Helper()
	tmpl, err := template.New("test").Funcs(template.FuncMap{
		"ID":  func() string { return "" },
		"JSM": func(_ string) string { return "" },
	}).Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}
