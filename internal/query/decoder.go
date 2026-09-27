// Package query decodes URL query parameters into a struct. It reads the same
// names (the json tag name, or the Go field name) and optional rules as the
// generated TypeScript, so a client that follows the ApiSchema sends the keys that
// the decoder expects. Keys are matched exactly, including case.
package query

import (
	"encoding"
	"fmt"
	"net/url"
	"reflect"
	"strconv"

	"github.com/iambpn/chirpc/internal/tags"
)

// Decoder fills one struct type from URL query values.
type Decoder struct {
	fields []field
}

// field is one struct field that the decoder fills.
type field struct {
	index    []int
	key      string
	required bool
	set      func(target reflect.Value, values []string) error
}

var textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()

// NewDecoder prepares a decoder for the struct type typ. It returns an error when a
// field has a type that cannot be read from a query string, such as a nested struct.
//
// Supported field types are strings, booleans, numbers, types that implement
// encoding.TextUnmarshaler (such as time.Time), pointers to these, and slices of these.
// A slice reads every value of a repeated key, as in "?tag=a&tag=b".
func NewDecoder(typ reflect.Type) (*Decoder, error) {
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("the query type must be a struct, but got %s", typ)
	}

	decoder := &Decoder{}
	if err := decoder.addFields(typ, nil, map[string]string{}); err != nil {
		return nil, err
	}
	return decoder, nil
}

// addFields adds the fields of typ, whose position in the root struct is index.
// keys maps each query key to the Go field that uses it, to report duplicate keys.
func (d *Decoder) addFields(typ reflect.Type, index []int, keys map[string]string) error {
	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		fieldIndex := append(append([]int{}, index...), i)

		if err := tags.CheckRemovedTags(structField); err != nil {
			return err
		}
		if tags.IsOmitted(structField) {
			continue
		}
		if tags.IsEmbedded(structField) {
			if structField.Type.Kind() != reflect.Struct {
				return fmt.Errorf("embedded field %s must be a struct, not a pointer or other type", structField.Name)
			}
			if err := d.addFields(structField.Type, fieldIndex, keys); err != nil {
				return err
			}
			continue
		}
		if structField.PkgPath != "" {
			continue
		}

		set, err := newSetter(structField.Type)
		if err != nil {
			return fmt.Errorf("field %s: %w", structField.Name, err)
		}

		key := tags.FieldName(structField)
		if other, ok := keys[key]; ok {
			return fmt.Errorf("fields %s and %s both use the query key %q", other, structField.Name, key)
		}
		keys[key] = structField.Name

		d.fields = append(d.fields, field{
			index:    fieldIndex,
			key:      key,
			required: !tags.IsOptional(structField),
			set:      set,
		})
	}
	return nil
}

// Decode fills target, an addressable struct value of the decoder's type, from values.
// It returns the problems for each query key, or nil when there are none.
func (d *Decoder) Decode(values url.Values, target reflect.Value) map[string][]string {
	var problems map[string][]string
	addProblem := func(key, message string) {
		if problems == nil {
			problems = map[string][]string{}
		}
		problems[key] = append(problems[key], message)
	}

	for _, f := range d.fields {
		fieldValues := values[f.key]
		if len(fieldValues) == 0 {
			if f.required {
				addProblem(f.key, "This field is required.")
			}
			continue
		}

		if err := f.set(target.FieldByIndex(f.index), fieldValues); err != nil {
			addProblem(f.key, err.Error())
		}
	}
	return problems
}

// newSetter returns a function that stores query values in a field of type typ.
func newSetter(typ reflect.Type) (func(reflect.Value, []string) error, error) {
	if typ.Kind() == reflect.Slice && !implementsTextUnmarshaler(typ) {
		parse, err := newParser(typ.Elem())
		if err != nil {
			return nil, err
		}
		return func(target reflect.Value, values []string) error {
			slice := reflect.MakeSlice(typ, len(values), len(values))
			for i, value := range values {
				if err := parse(slice.Index(i), value); err != nil {
					return err
				}
			}
			target.Set(slice)
			return nil
		}, nil
	}

	parse, err := newParser(typ)
	if err != nil {
		return nil, err
	}
	// A single value uses the first occurrence of the key, like url.Values.Get.
	return func(target reflect.Value, values []string) error {
		return parse(target, values[0])
	}, nil
}

// newParser returns a function that parses one query value into an addressable value of type typ.
func newParser(typ reflect.Type) (func(reflect.Value, string) error, error) {
	if implementsTextUnmarshaler(typ) {
		return func(target reflect.Value, value string) error {
			if err := target.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(value)); err != nil {
				return fmt.Errorf("The value %q is not valid.", value)
			}
			return nil
		}, nil
	}

	switch typ.Kind() {
	case reflect.Pointer:
		parse, err := newParser(typ.Elem())
		if err != nil {
			return nil, err
		}
		return func(target reflect.Value, value string) error {
			element := reflect.New(typ.Elem())
			if err := parse(element.Elem(), value); err != nil {
				return err
			}
			target.Set(element)
			return nil
		}, nil
	case reflect.String:
		return func(target reflect.Value, value string) error {
			target.SetString(value)
			return nil
		}, nil
	case reflect.Bool:
		return func(target reflect.Value, value string) error {
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("The value %q is not a valid boolean.", value)
			}
			target.SetBool(parsed)
			return nil
		}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(target reflect.Value, value string) error {
			parsed, err := strconv.ParseInt(value, 10, typ.Bits())
			if err != nil {
				return fmt.Errorf("The value %q is not a valid whole number.", value)
			}
			target.SetInt(parsed)
			return nil
		}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return func(target reflect.Value, value string) error {
			parsed, err := strconv.ParseUint(value, 10, typ.Bits())
			if err != nil {
				return fmt.Errorf("The value %q is not a valid non-negative whole number.", value)
			}
			target.SetUint(parsed)
			return nil
		}, nil
	case reflect.Float32, reflect.Float64:
		return func(target reflect.Value, value string) error {
			parsed, err := strconv.ParseFloat(value, typ.Bits())
			if err != nil {
				return fmt.Errorf("The value %q is not a valid number.", value)
			}
			target.SetFloat(parsed)
			return nil
		}, nil
	default:
		return nil, fmt.Errorf("type %s cannot be read from a query string", typ)
	}
}

// implementsTextUnmarshaler reports whether a pointer to typ implements encoding.TextUnmarshaler.
func implementsTextUnmarshaler(typ reflect.Type) bool {
	return reflect.PointerTo(typ).Implements(textUnmarshalerType)
}
