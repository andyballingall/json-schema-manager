package schema

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSpec(t *testing.T) {
	t.Parallel()
	s := &Schema{}
	ti := TestInfo{Path: "test.json"}
	spec := NewSpec(s, ti, TestDocTypePass, nil)

	assert.Equal(t, s, spec.Schema)
	assert.Equal(t, ti, spec.TestInfo)
	assert.Equal(t, TestDocTypePass, spec.TestDocType)
}

func TestSpec_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		testDocType TestDocType
		validateErr error
		wantErrType interface{}
	}{
		{
			name:        "Pass test doc succeeds",
			testDocType: TestDocTypePass,
			validateErr: nil,
			wantErrType: nil,
		},
		{
			name:        "Pass test doc fails",
			testDocType: TestDocTypePass,
			validateErr: errors.New("validation failed"),
			wantErrType: &PassTestFailedError{},
		},
		{
			name:        "Fail test doc succeeds (it fails validation as expected)",
			testDocType: TestDocTypeFail,
			validateErr: errors.New("validation failed"),
			wantErrType: nil,
		},
		{
			name:        "Fail test doc fails (it passes validation unexpectedly)",
			testDocType: TestDocTypeFail,
			validateErr: nil,
			wantErrType: &FailTestPassedError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := setupTestRegistry(t)
			s := New(Key("domain_family_1_0_0"), r)
			ti := TestInfo{
				Path:         "test.json",
				Unmarshalled: map[string]interface{}{},
			}
			spec := NewSpec(s, ti, tt.testDocType, nil)
			v := &mockValidator{Err: tt.validateErr}

			err := spec.Run(v)

			if tt.wantErrType != nil {
				require.Error(t, err)
				//nolint:testifylint // IsType is appropriate for table-driven tests with interface{}
				assert.IsType(t, tt.wantErrType, err)
				assert.Equal(t, err, spec.Err)
			} else {
				require.NoError(t, err)
				require.NoError(t, spec.Err)
			}
		})
	}
}

func TestSpec_ResultLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		testDocType    TestDocType
		err            error
		forwardVersion *SemVer
		want           string
	}{
		{
			name:        "Pass test doc - passed",
			testDocType: TestDocTypePass,
			err:         nil,
			want:        "passed",
		},
		{
			name:        "Pass test doc - failed",
			testDocType: TestDocTypePass,
			err:         errors.New("fail"),
			want:        "failed",
		},
		{
			name:        "Fail test doc - failed as expected",
			testDocType: TestDocTypeFail,
			err:         nil,
			want:        "failed, as expected",
		},
		{
			name:        "Fail test doc - passed unexpectedly",
			testDocType: TestDocTypeFail,
			err:         errors.New("pass"),
			want:        "passed, when expected fail",
		},
		{
			name:           "Pass test doc - breaking change (forward version)",
			testDocType:    TestDocTypePass,
			err:            errors.New("fail"),
			forwardVersion: &SemVer{1, 1, 0},
			want:           "failed - future version 1.1.0 introduced a breaking change!",
		},
		{
			name:           "Pass test doc - passed (forward version)",
			testDocType:    TestDocTypePass,
			err:            nil,
			forwardVersion: &SemVer{1, 1, 0},
			want:           "test from future version 1.1.0 passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			spec := Spec{
				TestDocType:    tt.testDocType,
				Err:            tt.err,
				ForwardVersion: tt.forwardVersion,
			}
			assert.Equal(t, tt.want, spec.ResultLabel())
		})
	}
}
