package schema

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_CoordinateRender_Parallel(t *testing.T) {
	t.Parallel()

	r := setupTestRegistry(t)
	k := Key("domain_family_1_0_0")
	createSchemaFiles(t, r, schemaMap{
		k: `{"type": "object"}`,
	})

	s, err := r.GetSchemaByKey(k)
	require.NoError(t, err)

	ec := r.config.ProductionEnvConfig()

	// Use a WaitGroup to run multiple renders in parallel
	var wg sync.WaitGroup
	numParallel := 10
	wg.Add(numParallel)

	results := make([]RenderInfo, numParallel)
	errors := make([]error, numParallel)

	for i := 0; i < numParallel; i++ {
		go func(idx int) {
			defer wg.Done()
			ri, rErr := r.CoordinateRender(s, ec)
			results[idx] = ri
			errors[idx] = rErr
		}(i)
	}

	wg.Wait()

	// All should have succeeded and returned the same result
	for i := 0; i < numParallel; i++ {
		require.NoError(t, errors[i])
		assert.NotNil(t, results[i].Validator)
		if i > 0 {
			assert.Equal(t, results[0], results[i])
		}
	}
}
