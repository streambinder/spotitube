package util

import (
	"fmt"
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

	length := 15
	if len(args) > 0 {
		length = args[0]
	}

	if len(sentence) > length && length > 0 {
		return sentence[:length]
	}

	return sentence
}

func Pad(sentence string, args ...int) string {
	sentence = Excerpt(sentence, args...)
	length := 15
	if len(args) > 0 {
		length = args[0]
	}

	for len(sentence) < length {
		sentence += " "
	}

	return sentence
}

func HumanizeBytes(bytes int) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%cB",
		float64(bytes)/float64(div), "kMGTPE"[exp])
}

func Fallback(data, fallback string) string {
	if len(data) == 0 {
		return fallback
	}
	return data
}

func ContainsEach(data string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(data, part) {
			return false
		}
	}
	return true
}
