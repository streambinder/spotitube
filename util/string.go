package util

import (
	"strings"

	"github.com/gosimple/slug"
)

func UniqueFields(sentence string) (uniqueFieldsSentence string) {
	var (
		appearances  = make(map[string]bool)
		uniqueFields []string
	)
	for _, field := range strings.Fields(strings.ReplaceAll(slug.Make(sentence), "-", " ")) {
		if appearances[field] {
			continue
		}
		appearances[field] = true
		uniqueFields = append(uniqueFields, field)
	}
	return strings.Join(uniqueFields, " ")
}
