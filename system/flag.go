package system

import (
	"fmt"
	"strings"
)

// StringsFlag represents a flag to wrap an array of strings
type StringsFlag struct {
	Entries []string
}

// String is the string representation for StringsFlag object
func (f *StringsFlag) String() string {
	return fmt.Sprint(f.Entries)
}

// Set sets the value of a StringsFlag object
func (f *StringsFlag) Set(value string) error {
	f.Entries = append(f.Entries, strings.Split(value, ";")...)
	return nil
}

// IsSet returns true if flag has at least a value set
func (f *StringsFlag) IsSet() bool {
	return len(f.Entries) > 0
}
