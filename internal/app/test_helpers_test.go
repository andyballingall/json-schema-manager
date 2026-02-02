package app

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/andyballingall/json-schema-manager/internal/config"
	"github.com/andyballingall/json-schema-manager/internal/repo"
	"github.com/andyballingall/json-schema-manager/internal/schema"
)

const testConfig = `
environments:
  prod:
    privateUrlRoot: "https://json-schemas.internal.myorg.io/"
    publicUrlRoot: "https://json-schemas.myorg.io/"
    isProduction: true
`

const simpleTestConfig = "environments: " +
	"{prod: {publicUrlRoot: 'https://p', privateUrlRoot: 'https://pr', isProduction: true}}"

type MockManager struct {
	mock.Mock
	registry *schema.Registry
}

func (m *MockManager) Registry() *schema.Registry {
	return m.registry
}

func (m *MockManager) ValidateSchema(ctx context.Context, target schema.ResolvedTarget, verbose bool,
	format string, useColour bool, continueOnError bool, testScope schema.TestScope, skipCompatible bool,
) error {
	args := m.Called(ctx, target, verbose, format, useColour, continueOnError, testScope, skipCompatible)
	return args.Error(0)
}

func (m *MockManager) CreateSchema(domainAndFamilyName string) (schema.Key, error) {
	args := m.Called(domainAndFamilyName)
	k, _ := args.Get(0).(schema.Key)
	return k, args.Error(1)
}

func (m *MockManager) CreateSchemaVersion(k schema.Key, rt schema.ReleaseType) (schema.Key, error) {
	args := m.Called(k, rt)
	kNew, _ := args.Get(0).(schema.Key)
	return kNew, args.Error(1)
}

func (m *MockManager) RenderSchema(ctx context.Context, target schema.ResolvedTarget, env config.Env) ([]byte, error) {
	args := m.Called(ctx, target, env)
	res, _ := args.Get(0).([]byte)
	return res, args.Error(1)
}

func (m *MockManager) CheckChanges(ctx context.Context, envName config.Env) error {
	args := m.Called(ctx, envName)
	return args.Error(0)
}

func (m *MockManager) TagDeployment(ctx context.Context, envName config.Env) error {
	args := m.Called(ctx, envName)
	return args.Error(0)
}

func (m *MockManager) BuildDist(ctx context.Context, envName config.Env, all bool) error {
	args := m.Called(ctx, envName, all)
	return args.Error(0)
}

// MockGitter is a test mock for the repo.Gitter interface.
type MockGitter struct {
	GetLatestAnchorFunc  func(env config.Env) (repo.Revision, error)
	TagDeploymentFunc    func(env config.Env) (string, error)
	GetSchemaChangesFunc func(anchor repo.Revision, sourceDir, suffix string) ([]repo.Change, error)
}

func (m *MockGitter) GetLatestAnchor(env config.Env) (repo.Revision, error) {
	if m.GetLatestAnchorFunc != nil {
		return m.GetLatestAnchorFunc(env)
	}
	return "HEAD", nil
}

func (m *MockGitter) TagDeploymentSuccess(env config.Env) (string, error) {
	if m.TagDeploymentFunc != nil {
		return m.TagDeploymentFunc(env)
	}
	return "jsm-deploy/prod/20260130-120000", nil
}

func (m *MockGitter) GetSchemaChanges(anchor repo.Revision, sourceDir, suffix string) ([]repo.Change, error) {
	if m.GetSchemaChangesFunc != nil {
		return m.GetSchemaChangesFunc(anchor, sourceDir, suffix)
	}
	return nil, nil
}
