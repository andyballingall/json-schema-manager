package schema

import (
	"github.com/andyballingall/json-schema-manager/internal/validator"
)

// Spec defines a single schema test involving the rendered schema and a test JSON document which
// the schema should either validate (if a pass test doc) or not validate (if a fail test doc).
// Note that this would be called 'Test' except the file would be called 'test.go' and its test 'test_test.go'.
type Spec struct {
	Schema         *Schema     // The schema that is being tested with this spec
	TestInfo       TestInfo    // information about the test document itself
	TestDocType    TestDocType // the type of test document
	ForwardVersion *SemVer     // If this spec uses a test document from a future version, this is that version
	Err            error       // Once the spec has run, if it didn't give the expected outcome, the related error
}

// NewSpec sets up a new spec for execution.
func NewSpec(s *Schema, testInfo TestInfo, testDocType TestDocType, forwardVersion *SemVer) Spec {
	return Spec{
		Schema:         s,
		TestInfo:       testInfo,
		TestDocType:    testDocType,
		ForwardVersion: forwardVersion,
	}
}

func (s *Spec) Run(validator validator.Validator) error {
	u := s.TestInfo.Unmarshalled

	if s.TestDocType == TestDocTypePass {
		err := validator.Validate(u)
		if err != nil {
			s.Err = &PassTestFailedError{SchemaPath: s.Schema.Path(FilePath), TestDocPath: s.TestInfo.Path, Wrapped: err}
			return s.Err
		}
		return nil
	}

	// We expected it to fail and it passed:
	if err := validator.Validate(u); err == nil {
		s.Err = &FailTestPassedError{SchemaPath: s.Schema.Path(FilePath), TestDocPath: s.TestInfo.Path}
		return s.Err
	}

	// We expect it to fail, and it did:
	return nil
}

// ResultLabel returns a human-readable label for the result of the spec.
func (s *Spec) ResultLabel() string {
	if s.TestDocType == TestDocTypePass {
		if s.Err != nil {
			if s.ForwardVersion != nil {
				return "failed - future version " + s.ForwardVersion.String('.') + " introduced a breaking change!"
			}
			return "failed"
		}
		if s.ForwardVersion != nil {
			return "test from future version " + s.ForwardVersion.String('.') + " passed"
		}
		return "passed"
	}
	if s.Err != nil {
		return "passed, when expected fail"
	}
	return "failed, as expected"
}
