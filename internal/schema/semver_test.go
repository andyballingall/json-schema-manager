package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSemVer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		major           string
		minor           string
		patch           string
		want            SemVer
		wantErr         bool
		wantErrContains string
		errType         any
	}{
		{
			name:  "Valid",
			major: "1",
			minor: "2",
			patch: "3",
			want:  SemVer{1, 2, 3},
		},
		{
			name:            "Invalid Major",
			major:           "a",
			minor:           "0",
			patch:           "0",
			wantErr:         true,
			wantErrContains: "not a valid major version",
			errType:         &InvalidMajorVersionError{},
		},
		{
			name:            "Invalid Major - Zero",
			major:           "0",
			minor:           "0",
			patch:           "0",
			wantErr:         true,
			wantErrContains: "not a valid major version",
			errType:         &InvalidMajorVersionError{},
		},
		{
			name:            "Invalid Minor",
			major:           "1",
			minor:           "a",
			patch:           "0",
			wantErr:         true,
			wantErrContains: "not a valid minor version",
			errType:         &InvalidMinorVersionError{},
		},
		{
			name:            "Invalid Patch",
			major:           "1",
			minor:           "0",
			patch:           "a",
			wantErr:         true,
			wantErrContains: "not a valid patch version",
			errType:         &InvalidPatchVersionError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewSemVer(tt.major, tt.minor, tt.patch)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContains != "" {
					require.ErrorContains(t, err, tt.wantErrContains)
				}
				if tt.errType != nil {
					require.ErrorAs(t, err, &tt.errType)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSemVer_Accessors(t *testing.T) {
	t.Parallel()
	s := SemVer{1, 2, 3}
	assert.Equal(t, uint64(1), s.Major())
	assert.Equal(t, uint64(2), s.Minor())
	assert.Equal(t, uint64(3), s.Patch())
}

func TestSemVer_Set(t *testing.T) {
	t.Parallel()
	s := SemVer{1, 0, 0}
	s.Set(2, 3, 4)
	assert.Equal(t, SemVer{2, 3, 4}, s)
}

func TestSemVer_String(t *testing.T) {
	t.Parallel()
	s := SemVer{1, 2, 3}
	assert.Equal(t, "1.2.3", s.String('.'))
	assert.Equal(t, "1_2_3", s.String('_'))
}
