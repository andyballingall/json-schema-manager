package app

import (
	"fmt"
)

// formatValue implements pflag.Value to provide a custom type name in help text
// and validation for output formats.
type formatValue string

func (f *formatValue) String() string {
	return string(*f)
}

func (f *formatValue) Set(v string) error {
	if v != "json" && v != "text" {
		return fmt.Errorf("must be 'text' or 'json'")
	}
	*f = formatValue(v)
	return nil
}

func (f *formatValue) Type() string {
	return "<format>"
}

// pathValue implements pflag.Value to provide a custom type name in help text.
type pathValue string

func (p *pathValue) String() string {
	return string(*p)
}

func (p *pathValue) Set(v string) error {
	*p = pathValue(v)
	return nil
}

func (p *pathValue) Type() string {
	return "<path>"
}
