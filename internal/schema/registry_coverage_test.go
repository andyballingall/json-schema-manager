package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoordinateRender_AlreadyCached(t *testing.T) {
	t.Parallel()
	r := setupTestRegistry(t)
	k := Key("domain_family_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k: `{"type": "object"}`,
	})
	s, err := r.GetSchemaByKey(k)
	require.NoError(t, err)

	ec := r.config.ProductionEnvConfig()

	// First render to populate cache
	firstInfo, err := s.Render(ec)
	require.NoError(t, err)

	// Now call CoordinateRender directly
	internalInfo, err := r.CoordinateRender(s, ec)
	require.NoError(t, err)

	assert.Equal(t, firstInfo, internalInfo)
}
