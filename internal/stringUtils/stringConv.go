package stringUtils

import (
	"strings"
)

func ShouldToLower(s string, shouldConvert bool) string {
	if shouldConvert {
		return strings.ToLower(s)
	}

	return s
}

func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func Uncapitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
