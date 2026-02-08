package schema

import (
	"fmt"
)

type NoDomainError struct{}

func (e *NoDomainError) Error() string {
	return "At least one domain is required"
}

type InvalidDomainError struct {
	d []string
}

func (e *InvalidDomainError) Error() string {
	return fmt.Sprintf("domain %s must contain only [a-z], [0-9], and '-'", e.d)
}

type InvalidFamilyNameError struct {
	fn string
}

func (e *InvalidFamilyNameError) Error() string {
	return fmt.Sprintf("family name %s must contain only [a-z], [0-9], and '-'", e.fn)
}

type InvalidSchemaKeyCharactersError struct {
	Key Key
}

func (e *InvalidSchemaKeyCharactersError) Error() string {
	return fmt.Sprintf("%s contains invalid characters - must contain only [a-z], [0-9],'-', and '_'", e.Key)
}

type InvalidMajorVersionError struct {
	v string
}

func (e *InvalidMajorVersionError) Error() string {
	return fmt.Sprintf("%s is not a valid major version", e.v)
}

type InvalidMinorVersionError struct {
	v string
}

func (e *InvalidMinorVersionError) Error() string {
	return fmt.Sprintf("%s is not a valid minor version", e.v)
}

type InvalidPatchVersionError struct {
	v string
}

func (e *InvalidPatchVersionError) Error() string {
	return fmt.Sprintf("%s is not a valid patch version", e.v)
}

type InvalidKeyStringError struct {
	ks string
}

func (e *InvalidKeyStringError) Error() string {
	return fmt.Sprintf("%s is not a valid schema key. Example: 'domain_subdomain_family_1_0_0'", e.ks)
}

type LocationOutsideRootDirectoryError struct {
	Location      string
	RootDirectory string
}

func (e *LocationOutsideRootDirectoryError) Error() string {
	return fmt.Sprintf("location %s is outside root directory %s", e.Location, e.RootDirectory)
}

type NotASchemaFileError struct {
	Path string
}

func (e *NotASchemaFileError) Error() string {
	return fmt.Sprintf("%s is not a JSON schema file", e.Path)
}

type AlreadyExistsError struct {
	K Key
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("schema %s already exists", e.K)
}

type InvalidJSONError struct {
	Path    string
	Wrapped error
}

func (e InvalidJSONError) Error() string {
	return fmt.Sprintf("%s is not valid JSON: %s", e.Path, e.Wrapped)
}

type InvalidJSONSchemaError struct {
	Path    string
	Wrapped error
}

func (e InvalidJSONSchemaError) Error() string {
	return fmt.Sprintf("%s is not a valid JSON Schema: %s", e.Path, e.Wrapped)
}

type CannotReadXPublicError struct {
	Path string
}

func (e CannotReadXPublicError) Error() string {
	return fmt.Sprintf("%s contains invalid x-public property - if present, it must be a boolean", e.Path)
}

type TemplateFormatInvalidError struct {
	Path    string
	Wrapped error
}

func (e TemplateFormatInvalidError) Error() string {
	return fmt.Sprintf("%s has a template syntax error. Check {{ID }} and {{JSM `<schema key>`}} for "+
		"validity. Error: %v", e.Path, e.Wrapped)
}

type TemplateExecutionFailedError struct {
	Path    string
	Wrapped error
}

func (e TemplateExecutionFailedError) Error() string {
	return fmt.Sprintf("%s cannot be rendered. Check {{ID }} and {{JSM `<schema key>`}} for validity. "+
		"Error: %v", e.Path, e.Wrapped)
}

type RegistryInitError struct {
	Path string
	Err  error
}

func (e *RegistryInitError) Error() string {
	return fmt.Sprintf("Registry could not be initialised. Path: %s, Error: %s", e.Path, e.Err)
}

type RegistryRootNotFolderError struct {
	Path string
}

func (e *RegistryRootNotFolderError) Error() string {
	return fmt.Sprintf("Registry could not be initialised. Path: %s is not a directory", e.Path)
}

type JSMArgInvalidKeyError struct {
	Arg string
}

func (e *JSMArgInvalidKeyError) Error() string {
	return fmt.Sprintf("A $ref to a JSM schema ({{ JSM `<schema key>` }}) has an invalid <schema key>: %s", e.Arg)
}

type JSMArgNotFoundError struct {
	Key     Key
	Wrapped error
}

func (e *JSMArgNotFoundError) Error() string {
	return fmt.Sprintf("A $ref to a JSM schema ({{ JSM `%s` }}) could not be loaded. Error: %s", e.Key, e.Wrapped)
}

type NotFoundError struct {
	Path string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("schema not found: %s", e.Path)
}

type InvalidSearchScopeError struct {
	spec string
}

func (e *InvalidSearchScopeError) Error() string {
	return fmt.Sprintf("`%s` is not a valid JSM Search Scope. Valid Example: 'domain/family'", e.spec)
}

type InvalidSchemaFilenameError struct {
	Path string
}

func (e *InvalidSchemaFilenameError) Error() string {
	return fmt.Sprintf("schema file %s has an invalid filename structure", e.Path)
}

type FailTestPassedError struct {
	SchemaPath  string
	TestDocPath string
}

func (e FailTestPassedError) Error() string {
	return fmt.Sprintf("Fail Test document %s did not fail validation as expected for schema %s:",
		e.TestDocPath, e.SchemaPath)
}

type PassTestFailedError struct {
	SchemaPath  string
	TestDocPath string
	Wrapped     error
}

func (e PassTestFailedError) Error() string {
	return fmt.Sprintf("Pass Test document %s did not pass validation as expected for schema %s, error: %s",
		e.TestDocPath, e.SchemaPath, e.Wrapped)
}

type CannotReadTestDocumentError struct {
	Path string
}

func (e CannotReadTestDocumentError) Error() string {
	return fmt.Sprintf("test document %s could not be read", e.Path)
}

type InvalidTestDocumentError struct {
	Path string
}

func (e InvalidTestDocumentError) Error() string {
	return fmt.Sprintf("test document %s is not valid JSON", e.Path)
}

type TestDirMissingConfigError struct {
	Path string
	Type TestDocType
}

func (e *TestDirMissingConfigError) Error() string {
	return fmt.Sprintf("%s directory missing: %s", e.Type, e.Path)
}

type InvalidTestDocumentDirectoryError struct {
	Path string
}

func (e *InvalidTestDocumentDirectoryError) Error() string {
	return fmt.Sprintf("test document must be in a 'pass' or 'fail' directory: %s", e.Path)
}

type NoSchemaTargetsError struct{}

func (e *NoSchemaTargetsError) Error() string {
	return "must specify a schema to validate via positional argument or flags (--key, --id, --search-scope)"
}

type InvalidCreateSchemaArgError struct {
	Arg string
}

func (e *InvalidCreateSchemaArgError) Error() string {
	return fmt.Sprintf("Invalid create schema argument: %s - "+
		"you must provide a chain of one or more domains and a family name, "+
		"separated by '/' e.g. 'domain/subdomain/family'", e.Arg)
}

type InvalidTestScopeError struct {
	Scope string
}

func (e *InvalidTestScopeError) Error() string {
	return fmt.Sprintf("Invalid test scope: %s - valid scopes are: %s, %s, %s, %s, %s",
		e.Scope, TestScopeLocal, TestScopePass, TestScopeFail, TestScopeConsumerBreaking, TestScopeAll)
}

type InvalidTargetArgumentError struct {
	Arg string
}

func (e *InvalidTargetArgumentError) Error() string {
	return fmt.Sprintf("Invalid schema target: %s", e.Arg)
}

type TargetArgumentTargetsMultipleSchemasError struct {
	Arg string
}

func (e *TargetArgumentTargetsMultipleSchemasError) Error() string {
	return fmt.Sprintf("Schema target %s targets multiple schemas", e.Arg)
}

type InvalidReleaseTypeError struct {
	Value string
}

func (e *InvalidReleaseTypeError) Error() string {
	return fmt.Sprintf("Invalid release type: %s - valid release types are: %s, %s, %s",
		e.Value, ReleaseTypeMajor, ReleaseTypeMinor, ReleaseTypePatch)
}

type NoTargetArgumentError struct{}

func (e *NoTargetArgumentError) Error() string {
	return "No target has been provided"
}

type ChangedDeployedSchemasError struct {
	Paths []string
}

func (e *ChangedDeployedSchemasError) Error() string {
	return fmt.Sprintf("cannot modify deployed schemas: %v", e.Paths)
}
