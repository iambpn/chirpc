package stringUtils

import (
	"strings"
)

// ShouldToLower returns the lowercase version of the input string s if shouldConvert is true.
// Otherwise, it returns s unchanged.
func ShouldToLower(s string, shouldConvert bool) string {
	if shouldConvert {
		return strings.ToLower(s)
	}

	return s
}

// Capitalize returns the input string s with the first character converted to uppercase.
// If s is empty, it returns s unchanged.
func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Uncapitalize returns the input string s with the first character converted to lowercase.
// If s is empty, it returns s unchanged.
func Uncapitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
