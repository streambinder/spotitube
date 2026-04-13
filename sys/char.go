package sys

var alphaRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func RandomAlpha() rune {
	return alphaRunes[RandomInt(len(alphaRunes))]
}
