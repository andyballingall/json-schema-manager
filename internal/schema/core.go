package schema

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var validCoreRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

// Core is the core information about a schema, representing all the dimensions within the JSON Schema Manager domain.
// A Key is a string which uniquely identifies the core information e.g. to use as a key in a map of schemas.
type Core struct {
	domain     []string // The domain namespace of the schema - may consist of multiple levels
	familyName string   // Describes the schema purpose. A family comprises all versions of the schema.
	version    SemVer   // The semantic version of a version of a schema in a family.
}

// NewCore creates a core from strings, applying validation tests.
func NewCore(domain []string, familyName, major, minor, patch string) (*Core, error) {
	if len(domain) == 0 {
		return nil, &NoDomainError{}
	}

	for _, d := range domain {
		if !validCoreRegex.MatchString(d) {
			return nil, &InvalidDomainError{d: domain}
		}
	}

	if !validCoreRegex.MatchString(familyName) {
		return nil, &InvalidFamilyNameError{fn: familyName}
	}

	version, err := NewSemVer(major, minor, patch)
	if err != nil {
		return nil, err
	}

	c := &Core{
		domain:     domain,
		familyName: familyName,
		version:    version,
	}
	return c, nil
}

// NewCoreFromKey creates a core from a Key. Note that a Key is assumed to have
// already been validated.
func NewCoreFromKey(k Key) *Core {
	return &Core{
		domain:     k.Domain(),
		familyName: k.FamilyName(),
		version:    k.Version(),
	}
}

// NewCoreFromString creates a Core from a string.
// The sep argument is the character used to separate the parts.
// E.g. "domain-a/subdomain-a/family-name/1/0/0" (this is used on the CLI).
// E.g. "domain-a_domain-b_family-name_1_0_0" (using KeySeparator).
func NewCoreFromString(s string, sep byte) (*Core, error) {
	parts := strings.Split(s, string(sep))
	nParts := len(parts)
	// we expect at least 5 parts: At least one domain, a family name, and 3 semantic version integers.
	if nParts < 5 {
		return nil, &InvalidKeyStringError{ks: s}
	}

	return NewCore(
		parts[:nParts-4],
		parts[nParts-4],
		parts[nParts-3],
		parts[nParts-2],
		parts[nParts-1],
	)
}

// Key converts the Core into the idiomatic Key which is used to identify a schema in the registry.
func (c *Core) Key() Key {
	var b strings.Builder
	for i, d := range c.domain {
		if i > 0 {
			b.WriteByte(KeySeparator)
		}
		b.WriteString(d)
	}
	b.WriteByte(KeySeparator)
	b.WriteString(c.familyName)
	b.WriteByte(KeySeparator)
	b.WriteString(strconv.FormatUint(c.version.Major(), 10))
	b.WriteByte(KeySeparator)
	b.WriteString(strconv.FormatUint(c.version.Minor(), 10))
	b.WriteByte(KeySeparator)
	b.WriteString(strconv.FormatUint(c.version.Patch(), 10))
	return Key(b.String())
}

// Path creates a path that relates to a schema with this core information.
func (c *Core) Path(pt PathType, rootDirectory string) string {
	nParts := len(c.domain)
	switch pt {
	case FamilyDir:
		nParts += 2
	case HomeDir:
		nParts += 5
	case FilePath:
		nParts += 6
	}

	parts := make([]string, 0, nParts)
	parts = append(parts, rootDirectory)
	parts = append(parts, c.domain...)
	parts = append(parts, c.familyName)

	if pt != FamilyDir {
		parts = append(parts,
			strconv.FormatUint(c.version.Major(), 10),
			strconv.FormatUint(c.version.Minor(), 10),
			strconv.FormatUint(c.version.Patch(), 10),
		)
	}
	if pt == FilePath {
		parts = append(parts, string(c.Key())+SchemaSuffix)
	}
	return filepath.Join(parts...)
}
