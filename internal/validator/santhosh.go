package validator

import (
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// NewSanthoshCompiler returns a concrete implementation of Compiler.
// Using the santhosh-tekuri/jsonschema/v6 package.
func NewSanthoshCompiler() Compiler {
	return &santhoshCompiler{c: jsonschema.NewCompiler()}
}

// santhoshValidator wraps jsonschema.Schema to implement Validator.
type santhoshValidator struct {
	v *jsonschema.Schema
}

// Validate adapts jsonschema.Schema.Validate to match the Validator interface.
func (sv *santhoshValidator) Validate(doc JSONDocument) error {
	return sv.v.Validate(doc)
}

// santhoshCompiler wraps jsonschema.Compiler to implement Compiler.
type santhoshCompiler struct {
	mu sync.Mutex
	c  *jsonschema.Compiler
}

func (s *santhoshCompiler) AddSchema(id string, schemaData JSONSchema) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.c.AddResource(id, schemaData)
}

func (s *santhoshCompiler) Compile(id string) (Validator, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := s.c.Compile(id)
	if err != nil {
		return nil, err
	}
	return &santhoshValidator{v: v}, nil
}

func (s *santhoshCompiler) SupportedSchemaVersions() []Draft {
	return []Draft{
		Draft4,
		Draft6,
		Draft7,
		Draft2019_09,
		Draft2020_12,
	}
}
