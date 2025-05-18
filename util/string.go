package util

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/agnivade/levenshtein"
	"github.com/gosimple/slug"
)

var filenameIllegalCharacters = regexp.MustCompile(`[/\\?%*:|"<>]`)

func Flatten(sentence string) string {
	return strings.ReplaceAll(slug.Make(sentence), "-", " ")
}

func UniqueFields(sentence string) (uniqueFieldsSentence string) {
	var (
		appearances  = make(map[string]bool)
		uniqueFields []string
	)

	for _, field := range strings.Fields(Flatten(sentence)) {
		if appearances[field] || len(field) <= 3 {
			continue
		}
		appearances[field] = true
		uniqueFields = append(uniqueFields, field)
	}

	return strings.Join(uniqueFields, " ")
}

// consider only fields in the sentences which are not in common, ie:
// LBD("hello world", "earth hello") = LD("world", "eart")
func LevenshteinBoundedDistance(former, latter string) int {
	var uniqueFormer, uniqueLatter []string

	former, latter = UniqueFields(former), UniqueFields(latter)
	for _, field := range strings.Fields(former) {
		if !Contains(latter, field) {
			uniqueFormer = append(uniqueFormer, field)
		}
	}
	for _, field := range strings.Fields(latter) {
		if !Contains(former, field) {
			uniqueLatter = append(uniqueLatter, field)
		}
	}

	return levenshtein.ComputeDistance(
		strings.Join(uniqueFormer, " "),
		strings.Join(uniqueLatter, " "),
	)
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

func Contains(data string, parts ...string) bool {
	for _, part := range parts {
		partWord := regexp.MustCompile("\\b" + part + "\\b")
		if !partWord.MatchString(data) {
			return false
		}
	}
	return true
}

func LegalizeFilename(filename string) string {
	return filenameIllegalCharacters.ReplaceAllString(filename, "")
}

func FirstLine(text string) string {
	return strings.Split(text, "\n")[0]
}
