package config

import (
	"fmt"

	"github.com/andyballingall/json-schema-manager/internal/validator"
)

type MustHaveExactlyOneProductionEnvironmentError struct{}

func (e *MustHaveExactlyOneProductionEnvironmentError) Error() string {
	return "json-schema-manager-config.yml must have exactly one environment marked with isProduction: true"
}

type MissingConfigError struct {
	Path string
}

func (e *MissingConfigError) Error() string {
	return fmt.Sprintf("json-schema-manager-config.yml missing in: %s", e.Path)
}

type InvalidYAMLError struct {
	Wrapped error
}

func (e *InvalidYAMLError) Error() string {
	return fmt.Sprintf("json-schema-manager-config.yml is not a valid yaml document: %v", e.Wrapped)
}

type MissingPropertyError struct {
	Property string
}

func (e *MissingPropertyError) Error() string {
	return fmt.Sprintf("json-schema-manager-config.yml is missing required property: %s", e.Property)
}

type UnknownEnvironmentError struct {
	Env Env
}

func (e *UnknownEnvironmentError) Error() string {
	return fmt.Sprintf(
		"json-schema-manager-config.yml does not define environment '%s'",
		e.Env,
	)
}

type InvalidURLError struct {
	Wrapped  error
	Property string
	Value    string
}

func (e *InvalidURLError) Error() string {
	return fmt.Sprintf(
		"json-schema-manager-config.yml property %s has invalid URL '%s': %v",
		e.Property,
		e.Value,
		e.Wrapped,
	)
}

type InvalidDefaultJSONSchemaVersionError struct {
	Value     string
	Supported []validator.Draft
}

func (e *InvalidDefaultJSONSchemaVersionError) Error() string {
	return fmt.Sprintf(
		"json-schema-manager-config.yml property defaultJsonSchemaVersion has invalid value '%s'. Supported versions are: %v",
		e.Value,
		e.Supported,
	)
}
