// Package tags reads the struct tags that decide how a Go struct field appears
// in the generated TypeScript, in decoded query strings, and in the required-field
// check of JSON bodies. All of them use these rules, so a field has the same name
// everywhere.
package tags

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	// Type replaces the TypeScript type of a field.
	Type = "tsType"
	// Optional marks a field as optional.
	Optional = "tsOptional"
	// Omit leaves a field out.
	Omit = "tsOmit"
)

// FieldName returns the name of a field, which is the key encoding/json uses:
// the json tag name, or the Go field name when the tag has no name.
func FieldName(field reflect.StructField) string {
	if name, _ := jsonTag(field); name != "" {
		return name
	}
	return field.Name
}

// CheckRemovedTags returns an error when a field uses a tag that chirpc no longer
// supports, so the change of name cannot go unnoticed.
func CheckRemovedTags(field reflect.StructField) error {
	if _, ok := field.Tag.Lookup("tsKey"); ok {
		return fmt.Errorf("field %s uses the tsKey tag, which is no longer supported, so use the json tag to rename it", field.Name)
	}
	return nil
}

// IsOptional reports whether a field is optional. It is optional when it has
// tsOptional:"true" or the json omitempty or omitzero option.
func IsOptional(field reflect.StructField) bool {
	return strings.EqualFold(field.Tag.Get(Optional), "true") ||
		HasJSONOption(field, "omitempty") || HasJSONOption(field, "omitzero")
}

// IsOmitted reports whether a field is left out, because of json:"-", tsOmit:"true",
// or typescript:"-".
func IsOmitted(field reflect.StructField) bool {
	name, _ := jsonTag(field)
	return name == "-" || strings.EqualFold(field.Tag.Get(Omit), "true") ||
		field.Tag.Get("typescript") == "-"
}

// IsEmbedded reports whether a field is an embedded struct whose fields are promoted,
// as encoding/json does when the embedded field has no json tag.
func IsEmbedded(field reflect.StructField) bool {
	return field.Anonymous && field.Tag.Get("json") == ""
}

// HasJSONOption reports whether the json tag of a field has the given option, such as "string".
func HasJSONOption(field reflect.StructField, option string) bool {
	_, options := jsonTag(field)
	for _, candidate := range options {
		if candidate == option {
			return true
		}
	}
	return false
}

// HasRequiredField reports whether the struct typ, or a struct embedded in it, has
// an exported field that is neither omitted nor optional.
func HasRequiredField(typ reflect.Type) bool {
	return hasRequiredField(typ, map[reflect.Type]bool{})
}

// hasRequiredField implements HasRequiredField. seen holds the structs already
// checked, to stop on recursive embedding.
func hasRequiredField(typ reflect.Type, seen map[reflect.Type]bool) bool {
	typ = Dereference(typ)
	if typ == nil || typ.Kind() != reflect.Struct || seen[typ] {
		return false
	}
	seen[typ] = true

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if IsOmitted(field) {
			continue
		}
		if IsEmbedded(field) {
			if hasRequiredField(field.Type, seen) {
				return true
			}
			continue
		}
		if field.PkgPath != "" {
			continue
		}
		if !IsOptional(field) {
			return true
		}
	}
	return false
}

// Dereference returns the type that typ points to, following all pointers.
func Dereference(typ reflect.Type) reflect.Type {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

// jsonTag returns the name and options of a field's json tag.
func jsonTag(field reflect.StructField) (string, []string) {
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return "", nil
	}
	parts := strings.Split(tag, ",")
	return parts[0], parts[1:]
}
