package sys

func RandomAlpha() rune {
	return rune('a' + RandomInt(26))
}
