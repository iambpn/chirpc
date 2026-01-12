package tsGen

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/iambpn/chirpc/internal/stringUtils"
	"github.com/iambpn/chirpc/internal/tsGen/tsopts"
)

// getHeaderName generates a header name for a given type based on the provided TypeScript generation options.
// It dereferences pointer types, applies capitalization rules, and includes the package name if not in "main".
// The header name is used for TypeScript interface generation and can be customized via options.
func getHeaderName(valType reflect.Type, opt tsopts.TsGenOpts) string {
	// dereference pointer types
	for valType.Kind() == reflect.Ptr {
		valType = valType.Elem()
	}

	if !opt[tsopts.AddHeaderToInterface] {
		return ""
	}

	headerName := stringUtils.NormalizeStructFieldName(stringUtils.Capitalize(valType.Name()))
	if opt[tsopts.UnCapitalizeHeader] {
		headerName = stringUtils.Uncapitalize(valType.Name())
	}

	// get package path of the type
	packagePath := valType.PkgPath()

	if strings.ToLower(packagePath) != "main" {
		pathSlice := strings.Split(packagePath, "/")

		// use last segment of package path as package name
		packageName := stringUtils.Capitalize(pathSlice[len(pathSlice)-1])
		if opt[tsopts.UnCapitalizeHeader] {
			packageName = stringUtils.Uncapitalize(pathSlice[len(pathSlice)-1])
		}

		headerName = fmt.Sprintf("%s__%s", packageName, stringUtils.Capitalize(headerName))
	}

	return headerName
}

// getTsTypeTagValue returns the value of the "tsType" struct tag for the given field.
// If the tag is not present, it returns an empty string.
func getTsTypeTagValue(field reflect.StructField) string {
	if tagType, exists := field.Tag.Lookup(structTagType); exists {
		return tagType
	}

	return ""
}

// getTsKeyTagValue returns the key name for a struct field based on its tags.
// It first checks for a "json" tag and returns its value if present.
// If not, it checks for a custom "tsKey" tag and returns its value.
// If neither tag is present, it returns an empty string.
func getTsKeyTagValue(field reflect.StructField) string {
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

// isFieldOptional determines whether a struct field should be considered optional for TypeScript generation.
// It checks if the field has a "json" tag with "omitempty" or "omitzero", or a custom "tsOptional" tag set to "true".
// Returns true if the field is optional, otherwise false.
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

// isFieldOmitted determines whether a struct field should be excluded from TypeScript generation.
// It checks if the field has a "json" tag set to "-" or a "tsOmit" tag set to "true".
// Returns true if the field should be omitted, otherwise false.
func isFieldOmitted(field reflect.StructField) bool {
	// check for json tag first
	if jsonTag, exists := field.Tag.Lookup("json"); exists {
		if strings.TrimSpace(jsonTag) == "-" {
			return true
		}
	}

	// then check for tsIgnore tag
	if ignore, exists := field.Tag.Lookup(structTagOmit); exists {
		if strings.ToLower(ignore) == "true" {
			return true
		}
	}

	return false
}
