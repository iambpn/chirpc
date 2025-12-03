package stringUtils

import "strings"

// NormalizeStructFieldName normalizes struct field names by replacing generic brackets and dots with underscores (_).
func NormalizeStructFieldName(fieldName string) string {
	fieldName = strings.ReplaceAll(fieldName, "[", "_")
	fieldName = strings.ReplaceAll(fieldName, "]", "_")
	fieldName = strings.ReplaceAll(fieldName, ".", "_")

	return fieldName
}
