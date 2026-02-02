package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		domain          []string
		familyName      string
		major           string
		minor           string
		patch           string
		wantErr         bool
		wantErrContains string
		errType         any
	}{
		{
			name:       "Valid - single domain",
			domain:     []string{"domain"},
			familyName: "family",
			major:      "1",
			minor:      "0",
			patch:      "0",
			wantErr:    false,
		},
		{
			name:       "Valid - multiple domains",
			domain:     []string{"domain", "subdomain"},
			familyName: "family",
			major:      "2",
			minor:      "3",
			patch:      "4",
			wantErr:    false,
		},
		{
			name:       "Valid - with hyphens and numbers",
			domain:     []string{"my-domain", "sub-2"},
			familyName: "my-family-123",
			major:      "10",
			minor:      "20",
			patch:      "30",
			wantErr:    false,
		},
		{
			name:            "Invalid - empty domain",
			domain:          []string{},
			familyName:      "family",
			major:           "1",
			minor:           "0",
			patch:           "0",
			wantErr:         true,
			wantErrContains: "At least one domain is required",
			errType:         &NoDomainError{},
		},
		{
			name:            "Invalid - uppercase in domain",
			domain:          []string{"Domain"},
			familyName:      "family",
			major:           "1",
			minor:           "0",
			patch:           "0",
			wantErr:         true,
			wantErrContains: "must contain only [a-z], [0-9], and '-'",
			errType:         &InvalidDomainError{},
		},
		{
			name:            "Invalid - special char in domain",
			domain:          []string{"domain@123"},
			familyName:      "family",
			major:           "1",
			minor:           "0",
			patch:           "0",
			wantErr:         true,
			wantErrContains: "must contain only [a-z], [0-9], and '-'",
			errType:         &InvalidDomainError{},
		},
		{
			name:            "Invalid - uppercase in family name",
			domain:          []string{"domain"},
			familyName:      "Family",
			major:           "1",
			minor:           "0",
			patch:           "0",
			wantErr:         true,
			wantErrContains: "must contain only [a-z], [0-9], and '-'",
			errType:         &InvalidFamilyNameError{},
		},
		{
			name:            "Invalid - major version is zero",
			domain:          []string{"domain"},
			familyName:      "family",
			major:           "0",
			minor:           "0",
			patch:           "0",
			wantErr:         true,
			wantErrContains: "is not a valid major version",
			errType:         &InvalidMajorVersionError{},
		},
		{
			name:            "Invalid - non-numeric major version",
			domain:          []string{"domain"},
			familyName:      "family",
			major:           "abc",
			minor:           "0",
			patch:           "0",
			wantErr:         true,
			wantErrContains: "is not a valid major version",
			errType:         &InvalidMajorVersionError{},
		},
		{
			name:            "Invalid - non-numeric minor version",
			domain:          []string{"domain"},
			familyName:      "family",
			major:           "1",
			minor:           "abc",
			patch:           "0",
			wantErr:         true,
			wantErrContains: "is not a valid minor version",
			errType:         &InvalidMinorVersionError{},
		},
		{
			name:            "Invalid - non-numeric patch version",
			domain:          []string{"domain"},
			familyName:      "family",
			major:           "1",
			minor:           "0",
			patch:           "abc",
			wantErr:         true,
			wantErrContains: "is not a valid patch version",
			errType:         &InvalidPatchVersionError{},
		},
		{
			name:            "Invalid - negative major version",
			domain:          []string{"domain"},
			familyName:      "family",
			major:           "-1",
			minor:           "0",
			patch:           "0",
			wantErr:         true,
			wantErrContains: "is not a valid major version",
			errType:         &InvalidMajorVersionError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			core, err := NewCore(tt.domain, tt.familyName, tt.major, tt.minor, tt.patch)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					require.ErrorAs(t, err, &tt.errType)
				}
				if tt.wantErrContains != "" {
					require.ErrorContains(t, err, tt.wantErrContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, core)

			// Verify the core was constructed correctly by checking the Key output
			key := core.Key()
			assert.Equal(t, tt.domain, key.Domain())
			assert.Equal(t, tt.familyName, key.FamilyName())
		})
	}
}

func TestNewCoreFromKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		key            Key
		wantDomain     []string
		wantFamilyName string
		wantMajor      uint64
		wantMinor      uint64
		wantPatch      uint64
	}{
		{
			name:           "Single domain",
			key:            Key("domain_family_1_0_0"),
			wantDomain:     []string{"domain"},
			wantFamilyName: "family",
			wantMajor:      1,
			wantMinor:      0,
			wantPatch:      0,
		},
		{
			name:           "Multiple domains",
			key:            Key("domain_subdomain_family_2_3_4"),
			wantDomain:     []string{"domain", "subdomain"},
			wantFamilyName: "family",
			wantMajor:      2,
			wantMinor:      3,
			wantPatch:      4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			core := NewCoreFromKey(tt.key)
			assert.NotNil(t, core)

			// The core should produce the same key when Key() is called
			assert.Equal(t, tt.key, core.Key())
		})
	}
}

func TestNewCoreFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		input           string
		sep             byte
		wantKey         Key
		wantErr         bool
		wantErrContains string
	}{
		{
			name:    "Valid - slash separator (CLI style)",
			input:   "domain/family/1/0/0",
			sep:     '/',
			wantKey: Key("domain_family_1_0_0"),
			wantErr: false,
		},
		{
			name:    "Valid - underscore separator",
			input:   "domain_family_1_0_0",
			sep:     '_',
			wantKey: Key("domain_family_1_0_0"),
			wantErr: false,
		},
		{
			name:    "Valid - multiple domains with slash",
			input:   "domain/subdomain/family/2/3/4",
			sep:     '/',
			wantKey: Key("domain_subdomain_family_2_3_4"),
			wantErr: false,
		},
		{
			name:            "Invalid - too few parts",
			input:           "domain/family/1/0",
			sep:             '/',
			wantErr:         true,
			wantErrContains: "is not a valid schema key",
		},
		{
			name:            "Invalid - only 4 parts",
			input:           "domain/1/0/0",
			sep:             '/',
			wantErr:         true,
			wantErrContains: "is not a valid schema key",
		},
		{
			name:            "Invalid - non-numeric version",
			input:           "domain/family/a/0/0",
			sep:             '/',
			wantErr:         true,
			wantErrContains: "is not a valid major version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			core, err := NewCoreFromString(tt.input, tt.sep)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContains != "" {
					require.ErrorContains(t, err, tt.wantErrContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, core)
			assert.Equal(t, tt.wantKey, core.Key())
		})
	}
}

func TestCoreKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		domain     []string
		familyName string
		major      string
		minor      string
		patch      string
		wantKey    Key
	}{
		{
			name:       "Single domain",
			domain:     []string{"domain"},
			familyName: "family",
			major:      "1",
			minor:      "0",
			patch:      "0",
			wantKey:    Key("domain_family_1_0_0"),
		},
		{
			name:       "Multiple domains",
			domain:     []string{"org", "team", "project"},
			familyName: "schema",
			major:      "2",
			minor:      "3",
			patch:      "4",
			wantKey:    Key("org_team_project_schema_2_3_4"),
		},
		{
			name:       "With hyphens",
			domain:     []string{"my-org", "my-team"},
			familyName: "my-schema",
			major:      "10",
			minor:      "20",
			patch:      "30",
			wantKey:    Key("my-org_my-team_my-schema_10_20_30"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			core, err := NewCore(tt.domain, tt.familyName, tt.major, tt.minor, tt.patch)
			require.NoError(t, err)

			assert.Equal(t, tt.wantKey, core.Key())
		})
	}
}

func TestCorePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		domain        []string
		familyName    string
		major         string
		minor         string
		patch         string
		pathType      PathType
		rootDirectory string
		wantPath      string
	}{
		{
			name:          "FamilyDir - single domain",
			domain:        []string{"domain"},
			familyName:    "family",
			major:         "1",
			minor:         "0",
			patch:         "0",
			pathType:      FamilyDir,
			rootDirectory: "/registry",
			wantPath:      "/registry/domain/family",
		},
		{
			name:          "FamilyDir - multiple domains",
			domain:        []string{"org", "team"},
			familyName:    "schema",
			major:         "1",
			minor:         "0",
			patch:         "0",
			pathType:      FamilyDir,
			rootDirectory: "/registry",
			wantPath:      "/registry/org/team/schema",
		},
		{
			name:          "HomeDir - single domain",
			domain:        []string{"domain"},
			familyName:    "family",
			major:         "1",
			minor:         "2",
			patch:         "3",
			pathType:      HomeDir,
			rootDirectory: "/registry",
			wantPath:      "/registry/domain/family/1/2/3",
		},
		{
			name:          "FilePath - single domain",
			domain:        []string{"domain"},
			familyName:    "family",
			major:         "1",
			minor:         "0",
			patch:         "0",
			pathType:      FilePath,
			rootDirectory: "/registry",
			wantPath:      "/registry/domain/family/1/0/0/domain_family_1_0_0.schema.json",
		},
		{
			name:          "FilePath - multiple domains",
			domain:        []string{"org", "team"},
			familyName:    "schema",
			major:         "2",
			minor:         "3",
			patch:         "4",
			pathType:      FilePath,
			rootDirectory: "/registry",
			wantPath:      "/registry/org/team/schema/2/3/4/org_team_schema_2_3_4.schema.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			core, err := NewCore(tt.domain, tt.familyName, tt.major, tt.minor, tt.patch)
			require.NoError(t, err)

			assert.Equal(t, tt.wantPath, core.Path(tt.pathType, tt.rootDirectory))
		})
	}
}
