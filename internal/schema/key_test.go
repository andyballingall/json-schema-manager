package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKeyAccessors tests the Key type accessor methods that extract
// components from a well-formed key string.
func TestKeyAccessors(t *testing.T) {
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
		{
			name:           "With hyphens",
			key:            Key("my-domain_my-family_1_0_0"),
			wantDomain:     []string{"my-domain"},
			wantFamilyName: "my-family",
			wantMajor:      1,
			wantMinor:      0,
			wantPatch:      0,
		},
		{
			name:           "With numbers in names",
			key:            Key("domain123_family456_1_0_0"),
			wantDomain:     []string{"domain123"},
			wantFamilyName: "family456",
			wantMajor:      1,
			wantMinor:      0,
			wantPatch:      0,
		},
		{
			name:           "Deep nesting",
			key:            Key("domain_sub1_sub2_sub3_family_1_0_0"),
			wantDomain:     []string{"domain", "sub1", "sub2", "sub3"},
			wantFamilyName: "family",
			wantMajor:      1,
			wantMinor:      0,
			wantPatch:      0,
		},
		{
			name:           "Large version numbers",
			key:            Key("domain_family_100_200_300"),
			wantDomain:     []string{"domain"},
			wantFamilyName: "family",
			wantMajor:      100,
			wantMinor:      200,
			wantPatch:      300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantDomain, tt.key.Domain(), "Domain()")
			assert.Equal(t, tt.wantFamilyName, tt.key.FamilyName(), "FamilyName()")
			assert.Equal(t, tt.wantMajor, tt.key.Major(), "Major()")
			assert.Equal(t, tt.wantMinor, tt.key.Minor(), "Minor()")
			assert.Equal(t, tt.wantPatch, tt.key.Patch(), "Patch()")
		})
	}
}

// TestKeySeparator tests that the KeySeparator constant is correctly set.
func TestKeySeparator(t *testing.T) {
	t.Parallel()
	assert.Equal(t, KeySeparator, byte('_'))
}

func TestNewKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid key",
			input:   "domain_family_1_0_0",
			wantErr: false,
		},
		{
			name:    "valid multi-level domain",
			input:   "domain_sub_family_1_0_0",
			wantErr: false,
		},
		{
			name:    "invalid key - too few parts",
			input:   "domain_family_1_0",
			wantErr: true,
		},
		{
			name:    "invalid key - empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid key - wrong format",
			input:   "domain-family-1-0-0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k, err := NewKey(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, k)
			} else {
				require.NoError(t, err)
				assert.Equal(t, Key(tt.input), k)
			}
		})
	}
}
