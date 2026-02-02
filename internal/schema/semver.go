package schema

import (
	"fmt"
	"strconv"
)

// SemVer represents the semantic version of a schema.
type SemVer [3]uint64

func NewSemVer(major, minor, patch string) (SemVer, error) {
	vMaj, err := strconv.ParseUint(major, 10, 64)
	if err != nil || vMaj == 0 {
		return SemVer{}, &InvalidMajorVersionError{v: major}
	}

	vMin, err := strconv.ParseUint(minor, 10, 64)
	if err != nil {
		return SemVer{}, &InvalidMinorVersionError{v: minor}
	}

	vPat, err := strconv.ParseUint(patch, 10, 64)
	if err != nil {
		return SemVer{}, &InvalidPatchVersionError{v: patch}
	}

	return SemVer{vMaj, vMin, vPat}, nil
}

func (s SemVer) Major() uint64 {
	return s[0]
}

func (s SemVer) Minor() uint64 {
	return s[1]
}

func (s SemVer) Patch() uint64 {
	return s[2]
}

func (s *SemVer) Set(major, minor, patch uint64) {
	s[0] = major
	s[1] = minor
	s[2] = patch
}

// String returns the version as a string of digits separated by the sep byte.
func (s SemVer) String(sep byte) string {
	return fmt.Sprintf("%d%c%d%c%d", s[0], sep, s[1], sep, s[2])
}
