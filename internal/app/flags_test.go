package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatValue(t *testing.T) {
	t.Parallel()

	f := formatValue("text")
	assert.Equal(t, "text", f.String())
	assert.Equal(t, "<format>", f.Type())

	t.Run("valid values", func(t *testing.T) {
		t.Parallel()
		err := f.Set("json")
		require.NoError(t, err)
		assert.Equal(t, "json", f.String())

		err = f.Set("text")
		require.NoError(t, err)
		assert.Equal(t, "text", f.String())
	})

	t.Run("invalid value", func(t *testing.T) {
		t.Parallel()
		err := f.Set("invalid")
		require.Error(t, err)
		assert.EqualError(t, err, "must be 'text' or 'json'")
	})
}

func TestPathValue(t *testing.T) {
	t.Parallel()

	p := pathValue("")
	assert.Empty(t, p.String())
	assert.Equal(t, "<path>", p.Type())

	t.Run("set value", func(t *testing.T) {
		t.Parallel()
		err := p.Set("/some/path")
		require.NoError(t, err)
		assert.Equal(t, "/some/path", p.String())
	})
}
