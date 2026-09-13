package sys

import (
	"fmt"
	"strconv"
	"strings"
)

func ColonMinutesToMillis(time string) (uint32, error) {
	parts := strings.Split(time, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid format")
	}

	minutes, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, err
	}

	// Split seconds and fractional part
	secParts := strings.Split(parts[1], ".")
	if len(secParts) != 2 {
		return 0, fmt.Errorf("invalid seconds format")
	}

	seconds, err := strconv.ParseUint(secParts[0], 10, 32)
	if err != nil {
		return 0, err
	}

	hundredths, err := strconv.ParseUint(secParts[1], 10, 32)
	if err != nil {
		return 0, err
	}

	//nolint:gosec
	return uint32((minutes * 60 * 1000) + (seconds * 1000) + (hundredths * 10)), nil
}

func MillisToColonMinutes(ms uint32) string {
	minutes := ms / 60000
	seconds := (ms % 60000) / 1000
	hundredths := (ms % 1000) / 10
	return fmt.Sprintf("%02d:%02d.%02d", minutes, seconds, hundredths)
}
