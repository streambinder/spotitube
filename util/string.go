package util

import (
	"strings"

	"github.com/gosimple/slug"
)

func Flatten(sentence string) string {
	return strings.ReplaceAll(slug.Make(sentence), "-", " ")
}

func UniqueFields(sentence string) (uniqueFieldsSentence string) {
	var (
		appearances  = make(map[string]bool)
		uniqueFields []string
	)

	for _, field := range strings.Fields(Flatten(sentence)) {
		if appearances[field] {
			continue
		}
		appearances[field] = true
		uniqueFields = append(uniqueFields, field)
	}

	return strings.Join(uniqueFields, " ")
}

func Excerpt(sentence string, args ...int) string {
	sentence = strings.ReplaceAll(strings.ReplaceAll(sentence, "\n", " "), "\r", " ")

	length := 10
	if len(args) > 0 {
		length = args[0]
	}

	if len(sentence) > length {
		return sentence[:length]
	}

	return sentence
}

func Pad(sentence string, args ...int) string {
	sentence = Excerpt(sentence, args...)
	length := 10
	if len(args) > 0 {
		length = args[0]
	}

	for len(sentence) < length {
		sentence += " "
	}

	return sentence
}
