package anchor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateToWidth(t *testing.T) {
	assert.Equal(t, "hello", truncateToWidth("hello", 10))
	assert.Equal(t, "hello", truncateToWidth("hello", 5))
	assert.Equal(t, "hell", truncateToWidth("hello", 4))
	assert.Equal(t, "", truncateToWidth("", 10))
	assert.Equal(t, "hello", truncateToWidth("hello", 0))
	assert.Equal(t, "hello", truncateToWidth("hello", -1))
	assert.Equal(t, "héll", truncateToWidth("héllo", 4))
}

func TestTruncateToWidthANSI(t *testing.T) {
	colored := "\x1b[31mhello\x1b[0m"
	assert.Equal(t, colored, truncateToWidth(colored, 10))
	assert.Equal(t, colored, truncateToWidth(colored, 5))
	assert.Equal(t, "\x1b[31mhell\x1b[0m", truncateToWidth(colored, 4))
	assert.Equal(t, "\x1b[31mhel", truncateToWidth("\x1b[31mhello", 3))
	assert.Equal(t, "\x1b[1m\x1b[31mhi\x1b[0m", truncateToWidth("\x1b[1m\x1b[31mhi\x1b[0m", 2))
}

func TestTerminalWidth(t *testing.T) {
	original := getTerminalSize
	defer func() { getTerminalSize = original }()

	getTerminalSize = func(int) (int, error) { return 120, nil }
	assert.Equal(t, 120, terminalWidth())

	getTerminalSize = func(int) (int, error) { return 0, errors.New("no tty") }
	assert.Equal(t, 0, terminalWidth())
}
