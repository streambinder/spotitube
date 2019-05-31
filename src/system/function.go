package system

import (
	"unicode"
)

func isNonspacingMark(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}
