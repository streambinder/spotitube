package sys

func RandomAlpha() rune {
	return rune('a' - 1 + RandomInt(26))
}
