package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSchemaID = "http://example.com/schema.json"

func TestNewSanthoshCompiler(t *testing.T) {
	t.Parallel()
	c := NewSanthoshCompiler()
	assert.NotNil(t, c)
}

func TestSanthoshCompiler_AddSchema(t *testing.T) {
	t.Parallel()
	c := NewSanthoshCompiler()
	id := testSchemaID
	data := map[string]interface{}{
		"$id":  testSchemaID,
		"type": "object",
	}

	err := c.AddSchema(id, data)
	require.NoError(t, err)
}

func TestSanthoshCompiler_Compile(t *testing.T) {
	t.Parallel()
	t.Run("successful compile", func(t *testing.T) {
		t.Parallel()
		c := NewSanthoshCompiler()
		id := testSchemaID
		data := map[string]interface{}{
			"$id":  testSchemaID,
			"type": "object",
		}

		_ = c.AddSchema(id, data)
		v, err := c.Compile(id)
		require.NoError(t, err)
		assert.NotNil(t, v)
	})

	t.Run("compile missing schema", func(t *testing.T) {
		t.Parallel()
		c := NewSanthoshCompiler()
		id := "http://example.com/missing.json"

		v, err := c.Compile(id)
		require.Error(t, err)
		assert.Nil(t, v)
	})

	t.Run("compile invalid schema", func(t *testing.T) {
		t.Parallel()
		c := NewSanthoshCompiler()
		id := "http://example.com/invalid.json"
		data := map[string]interface{}{
			"type": 123, // type must be string or array
		}

		_ = c.AddSchema(id, data)
		v, err := c.Compile(id)
		require.Error(t, err)
		assert.Nil(t, v)
	})
}

func TestSanthoshCompiler_SupportedSchemaVersions(t *testing.T) {
	t.Parallel()
	c := NewSanthoshCompiler()
	versions := c.SupportedSchemaVersions()
	assert.Len(t, versions, 5)
	assert.Contains(t, versions, Draft4)
	assert.Contains(t, versions, Draft6)
	assert.Contains(t, versions, Draft7)
	assert.Contains(t, versions, Draft2019_09)
	assert.Contains(t, versions, Draft2020_12)
}

func TestSanthoshValidator_Validate(t *testing.T) {
	t.Parallel()
	c := NewSanthoshCompiler()
	id := testSchemaID
	data := map[string]interface{}{
		"$id":  testSchemaID,
		"type": "object",
		"properties": map[string]interface{}{
			"foo": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []interface{}{"foo"},
	}

	_ = c.AddSchema(id, data)
	v, _ := c.Compile(id)

	t.Run("valid document", func(t *testing.T) {
		t.Parallel()
		doc := map[string]interface{}{
			"foo": "bar",
		}
		err := v.Validate(doc)
		require.NoError(t, err)
	})

	t.Run("invalid document", func(t *testing.T) {
		t.Parallel()
		doc := map[string]interface{}{
			"foo": 123,
		}
		err := v.Validate(doc)
		require.Error(t, err)
	})

	t.Run("missing required field", func(t *testing.T) {
		t.Parallel()
		doc := map[string]interface{}{}
		err := v.Validate(doc)
		require.Error(t, err)
	})
}
