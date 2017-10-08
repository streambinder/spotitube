package spotitube

import (
	"os"
)

var (
	opt_interactive *bool = GetBoolPointer(false)
	opt_logfile     *bool = GetBoolPointer(false)
	opt_debug       *bool = GetBoolPointer(false)

	logger *Logger = NewLogger()

	opt_download_path string // TODO: fire this away
)

func IsDir(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	file_stat, err := file.Stat()
	if err != nil {
		return false
	}
	return file_stat.IsDir()
}

func MakeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}

func GetBoolPointer(value bool) *bool {
	return &value
}
