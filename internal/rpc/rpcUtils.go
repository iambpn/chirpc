package rpc

import (
	"errors"
	"reflect"
	"strings"
)

// ColonPattern converts a URL pattern with curly braces to a colon-prefixed slug format.
// A chi regex after the slug name is dropped, so "/{id:[0-9]+}" becomes "/:id".
func ColonPattern(input string) string {
	var result []rune
	braces := 0
	var buffer []rune

	for _, r := range input {
		switch {
		case r == '{':
			if braces == 0 {
				buffer = buffer[:0] // reset buffer for a new slug section
			}
			braces++
			if braces > 1 {
				buffer = append(buffer, r)
			}
		case r == '}' && braces > 0:
			braces--
			if braces == 0 {
				// flush buffered content as :slug...
				result = append(result, ':')
				result = append(result, []rune(slugName(string(buffer)))...)
			} else {
				buffer = append(buffer, r)
			}
		default:
			if braces > 0 {
				buffer = append(buffer, r)
			} else {
				result = append(result, r)
			}
		}
	}

	return string(result)
}

// parseURLSlugs returns the names of the chi path parameters in url, in the order they appear.
// Braces inside a parameter, as in "{code:[a-z]{3}}", belong to that parameter.
func parseURLSlugs(url string) []string {
	slugs := []string{}
	braces := 0
	var buffer []rune

	for _, r := range url {
		switch {
		case r == '{':
			if braces > 0 {
				buffer = append(buffer, r)
			}
			braces++
		case r == '}' && braces > 0:
			braces--
			if braces == 0 {
				slugs = append(slugs, slugName(string(buffer)))
				buffer = buffer[:0]
			} else {
				buffer = append(buffer, r)
			}
		case braces > 0:
			buffer = append(buffer, r)
		}
	}

	return slugs
}

// slugName returns the parameter name from a chi slug, without the optional ":regex" part.
func slugName(slug string) string {
	name, _, _ := strings.Cut(slug, ":")
	return name
}

// mergePaths combines a base path and relative path into a single path,
// handling trailing and leading slashes appropriately.
func mergePaths(basePath, relativePath string) string {
	if basePath == "" {
		return relativePath
	}
	if relativePath == "" {
		return basePath
	}

	hasBaseSlash := basePath[len(basePath)-1] == '/'
	hasRelativeSlash := relativePath[0] == '/'

	switch {
	case hasBaseSlash && hasRelativeSlash:
		return basePath + relativePath[1:]
	case !hasBaseSlash && !hasRelativeSlash:
		return basePath + "/" + relativePath
	default:
		return basePath + relativePath
	}
}

// extractReturnType returns the first (non-pointer) return type of a function.
// It errors if the input is not a function or has no return values.
func extractReturnType(typeVal reflect.Type) (reflect.Type, error) {
	if typeVal.Kind() == reflect.Pointer {
		typeVal = typeVal.Elem()
	}

	// get the return type of the function
	if typeVal.Kind() != reflect.Func {
		return nil, errors.New("provided value is not a function")
	}

	// check if function has at least one return value
	if typeVal.NumOut() < 1 {
		return nil, errors.New("function must have at least one return value")
	}

	// get First return value of the function
	retType := typeVal.Out(0)

	// convert retType to its underlying type if it's a pointer
	if retType.Kind() == reflect.Pointer {
		retType = retType.Elem()
	}

	return retType, nil
}

// BuildGoToTsSchema constructs a HandlerSchema from the given HTTP method, URL, and handler function.
// It extracts the return type from the handler and initializes the schema.
// It does not add the schema to any collection for type generation.
// Returns an error if the handler is not a valid function or lacks a return type.
func BuildGoToTsSchema(method, url string, fnVal any) (*HandlerSchema, error) {
	typeVal := reflect.TypeOf(fnVal)

	retType, err := extractReturnType(typeVal)
	if err != nil {
		return nil, err
	}

	schema := NewHandlerSchema(method, url, retType)

	return schema, nil
}
