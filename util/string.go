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

func Excerpt(sentence string, args ...bool) string {
	pad := false
	if len(args) > 0 {
		pad = args[0]
	}

	for len(sentence) < 10 {
		if !pad {
			return sentence
		}
		sentence += " "
	}
	return sentence[:10]
}
