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

func getTagKey(field reflect.StructField, opt tsopts.TsGenOpts) string {
	fieldKey := field.Name
	if key, exists := field.Tag.Lookup(structTagKey); exists {
		fieldKey = key
	}

	return stringUtils.ShouldToLower(fieldKey, opt[tsopts.ToLowercase])
}

func isFieldOptional(field reflect.StructField) bool {
	if optional, exists := field.Tag.Lookup(structTagOptional); exists {
		if strings.ToLower(optional) == "true" {
			return true
		}
	}

	return false
}
