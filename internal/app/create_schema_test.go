package app

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andyballingall/json-schema-manager/internal/schema"
)

func TestCreateSchemaCmd(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*MockManager, *cobra.Command) {
		t.Helper()
		reg := setupTestRegistry(t)
		mgr := &MockManager{
			registry: reg,
		}
		cmd := NewCreateSchemaCmd(mgr)
		return mgr, cmd
	}

	t.Run("successful execution", func(t *testing.T) {
		t.Parallel()
		mgr, cmd := setup(t)
		domainAndFamily := "test-domain/test-family"
		key := schema.Key("test-domain_test-family_1_0_0")

		// Create the schema file in the correct nested directory structure
		s := schema.New(key, mgr.Registry())
		require.NoError(t, os.MkdirAll(s.Path(schema.HomeDir), 0o755))
		require.NoError(t, os.WriteFile(s.Path(schema.FilePath), []byte("{}"), 0o600))

		mgr.On("CreateSchema", domainAndFamily).Return(key, nil).Once()

		cmd.SetArgs([]string{domainAndFamily})
		err := cmd.ExecuteContext(context.Background())
		require.NoError(t, err)
		mgr.AssertExpectations(t)
	})

	t.Run("missing args errors", func(t *testing.T) {
		t.Parallel()
		_, cmd := setup(t)
		cmd.SetArgs([]string{})
		err := cmd.ExecuteContext(context.Background())
		require.Error(t, err)
	})

	t.Run("too many args errors", func(t *testing.T) {
		t.Parallel()
		_, cmd := setup(t)
		cmd.SetArgs([]string{"arg1", "arg2"})
		err := cmd.ExecuteContext(context.Background())
		require.Error(t, err)
	})

	t.Run("manager error propagates", func(t *testing.T) {
		t.Parallel()
		mgr, cmd := setup(t)
		domainAndFamily := "test-domain/test-family"
		mgr.On("CreateSchema", domainAndFamily).Return(schema.Key(""), errors.New("manager error")).Once()

		cmd.SetArgs([]string{domainAndFamily})
		err := cmd.ExecuteContext(context.Background())
		require.Error(t, err)
		assert.Equal(t, "manager error", err.Error())
		mgr.AssertExpectations(t)
	})
}
