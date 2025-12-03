package rpc

import (
	"errors"
	"fmt"
	"reflect"
)

// convertURLPattern converts a URL pattern with curly braces to a colon-prefixed slug format.
func convertURLPattern(input string) string {
	var result []rune
	braces := 0
	var buffer []rune

	for _, r := range input {
		switch r {
		case '{':
			if braces == 0 {
				buffer = buffer[:0] // reset buffer for a new slug section
			}
			braces++
			if braces > 1 {
				buffer = append(buffer, r)
			}
		case '}':
			braces--
			if braces == 0 {
				// flush buffered content as :slug...
				result = append(result, ':')
				result = append(result, buffer...)
			} else if braces > 0 {
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

// sliceToTsInf builds a TypeScript interface string mapping each slug to string,
// or returns 'never' if the slice is empty.
func sliceToTsInf(slice []string) string {
	if len(slice) == 0 {
		return "never"
	}

	inf := ""
	for _, s := range slice {
		inf += fmt.Sprintf(`"%s": string;`, s)
	}

	return fmt.Sprintf("{ %s }", inf)
}
