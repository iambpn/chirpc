package tsGen

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/iambpn/chirpc/internal/stringUtils"
	"github.com/iambpn/chirpc/internal/tsGen/tsopts"
)

func getHeaderName(valType reflect.Type, opt tsopts.TsGenOpts) string {
	if !opt[tsopts.AddHeaderToInterface] {
		return ""
	}

	headerName := stringUtils.NormalizeStructFieldName(stringUtils.Capitalize(valType.Name()))
	if opt[tsopts.UnCapitalizeHeader] {
		headerName = stringUtils.Uncapitalize(valType.Name())
	}

	packagePath := valType.PkgPath()

	if strings.ToLower(packagePath) != "main" {
		pathSlice := strings.Split(packagePath, "/")

		packageName := stringUtils.Capitalize(pathSlice[len(pathSlice)-1])

		if opt[tsopts.UnCapitalizeHeader] {
			packageName = stringUtils.Uncapitalize(pathSlice[len(pathSlice)-1])
		}

		headerName = fmt.Sprintf("%s__%s", packageName, stringUtils.Capitalize(headerName))
	}

	return headerName
}

func getTagType(field reflect.StructField) string {
	if tagType, exists := field.Tag.Lookup(structTagType); exists {
		return tagType
	}

	return ""
}

func getTagKey(field reflect.StructField) string {
	// check for json tag first
	if jsonTagKey := getJsonTagValue(field); jsonTagKey != "" {
		return jsonTagKey
	}

	// then check for tsKey tag
	if key, exists := field.Tag.Lookup(structTagKey); exists {
		return key
	}

	return ""
}

// getJsonTagValue returns the value of the "json" struct tag for a given field.
// If the "json" tag is not present or has no value, it returns an empty string.
// For example, for a struct field defined as:
//
//	FieldName string `json:"field_name,omitempty"`
//
// This function will return "field_name".
func getJsonTagValue(field reflect.StructField) string {
	if jsonTag, exists := field.Tag.Lookup("json"); exists {
		splitTag := strings.Split(jsonTag, ",")
		if len(splitTag) > 0 && splitTag[0] != "" {
			return splitTag[0]
		}
	}

	return ""
}

// isJsonTagOptional checks if the "json" struct tag for a given field includes "omitempty" or "omitzero".
// It returns true if either option is present, indicating that the field is optional in JSON serialization.
func isJsonTagOptional(field reflect.StructField) bool {
	if jsonTag, exists := field.Tag.Lookup("json"); exists {
		if strings.Contains(jsonTag, ",omitempty") || strings.Contains(jsonTag, ",omitzero") {
			return true
		}
	}

	return false
}

func isFieldOptional(field reflect.StructField) bool {
	// check for json tag first
	if isJsonTagOptional(field) {
		return true
	}

	// then check for tsOptional tag
	if optional, exists := field.Tag.Lookup(structTagOptional); exists {
		if strings.ToLower(optional) == "true" {
			return true
		}
	}

	return false
}
