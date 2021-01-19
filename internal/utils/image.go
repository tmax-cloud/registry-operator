package utils

import (
	"regexp"
	"strings"
)

func Matched(pattern string, image string) bool {
	pattern = strings.ReplaceAll(pattern, "?", "[\\w\\d:./_-]")
	pattern = strings.ReplaceAll(pattern, "*", "[\\w\\d:./_-]*")
	pattern = "^" + pattern

	if pattern[len(pattern)-1] != '*' {
		pattern += "$"
	}

	regex := regexp.MustCompile(pattern)

	return regex.MatchString(image)
}
