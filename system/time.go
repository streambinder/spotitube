package system

import (
	"strconv"
	"strings"
)

// ColonDuration parses a duration formatted
// with colons and returns it in its seconds
// representation
func ColonDuration(duration string) (seconds int) {
	parts := strings.Split(duration, ":")
	for loop, i := 0, len(parts)-1; i >= 0; loop, i = loop+1, i-1 {
		part, err := strconv.Atoi(parts[i])
		if err != nil {
			continue
		}

		switch loop {
		case 0: // seconds
			seconds += part
			break
		case 1: // minutes
			seconds += part * 60
			break
		case 2: // hours
			seconds += part * 60 * 60
			break
		}
	}
	return
}
