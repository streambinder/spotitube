package system

import (
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Asciify : transform eventually unicoded string to ASCII
func Asciify(dirty string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isNonspacingMark), norm.NFC)
	clean, _, _ := transform.String(t, dirty)
	return clean
}

func isNonspacingMark(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}
