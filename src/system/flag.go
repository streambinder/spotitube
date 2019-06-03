package system

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathsArrayFlag : struct containing all the informations about a parsed PathsArrayFlag input flag
type PathsArrayFlag struct {
	Paths []string
}

// String : string representation for PathsArrayFlag object
func (flag *PathsArrayFlag) String() string {
	return fmt.Sprint(flag.Paths)
}

// Set : set value of a PathsArrayFlag object
func (flag *PathsArrayFlag) Set(value string) error {
	paths := strings.Split(value, ";")
	for _, path := range paths {
		pathAbs, pathErr := filepath.Abs(path)
		if pathErr != nil {
			return pathErr
		}
		flag.Paths = append(flag.Paths, pathAbs)
	}
	return nil
}
