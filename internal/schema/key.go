package schema

import (
	"strconv"
	"strings"
)

// KeySeparator is the separator used to separate the domains, family name, and semantic version in a Key
// (and therefore also the filename of a schema).
const KeySeparator byte = '_'

var KeySeparatorString = string(KeySeparator)

// Key is a string uniquely identifying a JSON Schema Manager schema in a schema cache in memory.
// The schema filename in the registry is the key suffixed by ".schema.json".
// Keys are always valid, and are used in contexts where many keys might be managed
// in lists or maps. Contrast with Core, which has explicit validation, and uses separate
// properties for each component.
type Key string

// NewKey creates a new Key from a string that is supposedly a valid key.
// If the string is not a valid key, an error is returned.
func NewKey(s string) (Key, error) {
	// check that the string is a valid key by tring to build a Core from it.
	if _, e := NewCoreFromString(s, KeySeparator); e != nil {
		return "", e
	}
	// We know it's a valid key, so just return it.
	return Key(s), nil
}

func (k Key) Domain() []string {
	parts := strings.Split(string(k), KeySeparatorString)
	return parts[:len(parts)-4]
}

func (k Key) FamilyName() string {
	parts := strings.Split(string(k), KeySeparatorString)
	return parts[len(parts)-4]
}

func (k Key) Major() uint64 {
	parts := strings.Split(string(k), KeySeparatorString)
	major, _ := strconv.ParseUint(parts[len(parts)-3], 10, 64)
	return major
}

func (k Key) Minor() uint64 {
	parts := strings.Split(string(k), KeySeparatorString)
	minor, _ := strconv.ParseUint(parts[len(parts)-2], 10, 64)
	return minor
}

func (k Key) Patch() uint64 {
	parts := strings.Split(string(k), KeySeparatorString)
	patch, _ := strconv.ParseUint(parts[len(parts)-1], 10, 64)
	return patch
}

func (k Key) Version() SemVer {
	return SemVer{k.Major(), k.Minor(), k.Patch()}
}
