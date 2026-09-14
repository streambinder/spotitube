package anchor

import (
	"os"
	"strings"

	"golang.org/x/term"
)

func fitLine(data string) string {
	return truncateToWidth(data, terminalWidth())
}

func truncateToWidth(data string, width int) string {
	if width <= 0 {
		return data
	}
	var (
		out      strings.Builder
		visible  int
		inEscape bool
	)
	out.Grow(len(data))
	for _, r := range data {
		switch {
		case r == '\x1b':
			inEscape = true
			out.WriteRune(r)
		case inEscape:
			out.WriteRune(r)
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
				inEscape = false
			}
		case visible < width:
			out.WriteRune(r)
			visible++
		}
	}
	return out.String()
}

func terminalWidth() int {
	if width, err := getTerminalSize(int(os.Stdout.Fd())); err == nil {
		return width
	}
	return 0
}

var getTerminalSize = func(fd int) (int, error) {
	width, _, err := term.GetSize(fd)
	return width, err
}
