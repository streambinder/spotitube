package system

import (
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Asciify transforms eventually unicoded string to ASCII
func Asciify(dirty string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r)
	}), norm.NFC)
	clean, _, _ := transform.String(t, dirty)
	return clean
}
