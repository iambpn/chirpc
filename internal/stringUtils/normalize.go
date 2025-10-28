package stringUtils

import "strings"

// NormalizeStructFieldName by replacing generic brackets and . with underscores (_)
func NormalizeStructFieldName(fieldName string) string {
	fieldName = strings.ReplaceAll(fieldName, "[", "_")
	fieldName = strings.ReplaceAll(fieldName, "]", "_")
	fieldName = strings.ReplaceAll(fieldName, ".", "_")

	return fieldName
}
