// Package jsonbody finds required fields that are missing from a JSON request body.
// encoding/json/v2 leaves a missing field at its zero value without an error, so this
// check runs after decoding. A field is required when it is neither omitted nor
// optional, which is the same rule the generated TypeScript uses.
package jsonbody

import (
	"encoding"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/iambpn/chirpc/internal/tags"
)

// Checker checks one Go type for missing required fields.
type Checker struct {
	root *typePlan
}

// planKind says how a typePlan looks inside a JSON value.
type planKind int

const (
	objectPlan planKind = iota // a struct, read from a JSON object
	listPlan                   // a slice or array, whose elements are checked
	mapPlan                    // a map, whose values are checked
)

// typePlan describes the checks for one Go type. A nil *typePlan means there is nothing to check.
type typePlan struct {
	kind   planKind
	fields []fieldPlan // for objectPlan
	elem   *typePlan   // for listPlan and mapPlan
}

// fieldPlan is one struct field, read from the JSON key key. When ignoreCase is set,
// the key is matched as the json case:ignore option does.
type fieldPlan struct {
	key        string
	ignoreCase bool
	required   bool
	plan       *typePlan
}

var (
	jsonUnmarshalerType     = reflect.TypeFor[json.Unmarshaler]()
	jsonUnmarshalerFromType = reflect.TypeFor[json.UnmarshalerFrom]()
	textUnmarshalerType     = reflect.TypeFor[encoding.TextUnmarshaler]()
)

// NewChecker prepares a checker for typ. Nested structs, and structs inside slices,
// arrays, and map values, are checked too. Types with their own JSON decoding, such
// as time.Time, are not looked into.
func NewChecker(typ reflect.Type) *Checker {
	builder := planBuilder{plans: map[reflect.Type]*typePlan{}}
	return &Checker{root: builder.plan(typ)}
}

// Check returns the missing required fields of the JSON document data, keyed by their
// path, such as "address.city" or "items[0].name". It returns nil when none are missing.
// data must already be valid JSON for the checker's type.
func (c *Checker) Check(data []byte) map[string][]string {
	if c.root == nil {
		return nil
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil
	}

	problems := map[string][]string{}
	c.root.check(value, "", problems)
	if len(problems) == 0 {
		return nil
	}
	return problems
}

// check adds the missing required fields in value, found at path, to problems.
// Values of the wrong JSON type are skipped, because decoding already reports them.
// JSON null is accepted, because encoding/json/v2 accepts it for any type.
func (p *typePlan) check(value any, path string, problems map[string][]string) {
	switch p.kind {
	case objectPlan:
		object, ok := value.(map[string]any)
		if !ok {
			return
		}
		for _, field := range p.fields {
			fieldValue, found := field.lookup(object)
			fieldPath := joinPath(path, field.key)
			if !found {
				if field.required {
					problems[fieldPath] = append(problems[fieldPath], "This field is required.")
				}
				continue
			}
			if field.plan != nil {
				field.plan.check(fieldValue, fieldPath, problems)
			}
		}
	case listPlan:
		list, ok := value.([]any)
		if !ok {
			return
		}
		for i, element := range list {
			p.elem.check(element, fmt.Sprintf("%s[%d]", path, i), problems)
		}
	case mapPlan:
		object, ok := value.(map[string]any)
		if !ok {
			return
		}
		for key, element := range object {
			p.elem.check(element, joinPath(path, key), problems)
		}
	}
}

// lookup finds the field's value in object the way encoding/json/v2 does. The key must
// match exactly, unless the field has case:ignore, which also ignores dashes and underscores.
func (f fieldPlan) lookup(object map[string]any) (any, bool) {
	if value, ok := object[f.key]; ok || !f.ignoreCase {
		return value, ok
	}
	key := foldName(f.key)
	for candidate, value := range object {
		if foldName(candidate) == key {
			return value, true
		}
	}
	return nil, false
}

// nameSeparators removes the dashes and underscores that case:ignore does not compare.
var nameSeparators = strings.NewReplacer("-", "", "_", "")

// foldName returns name in lower case without dashes and underscores.
func foldName(name string) string {
	return strings.ToLower(nameSeparators.Replace(name))
}

// Path turns pointer, a JSON Pointer into the JSON document data, into a path in the
// form Check uses, such as "items[0].name". A JSON Pointer cannot tell an array index
// from an object key, so data is read to find out which one each token is.
func Path(data []byte, pointer jsontext.Pointer) string {
	var value any
	_ = json.Unmarshal(data, &value)

	path := ""
	for token := range pointer.Tokens() {
		switch current := value.(type) {
		case []any:
			path += "[" + token + "]"
			if index, err := strconv.Atoi(token); err == nil && index >= 0 && index < len(current) {
				value = current[index]
			} else {
				value = nil
			}
		case map[string]any:
			path = joinPath(path, token)
			value = current[token]
		default:
			path = joinPath(path, token)
			value = nil
		}
	}
	return path
}

// joinPath adds key to a dotted path.
func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// planBuilder builds typePlans. plans holds the plans already built, which also
// stops the builder on recursive types.
type planBuilder struct {
	plans map[reflect.Type]*typePlan
}

// plan returns the checks for typ, or nil when there is nothing to check.
func (b *planBuilder) plan(typ reflect.Type) *typePlan {
	typ = tags.Dereference(typ)
	if typ == nil || HasCustomDecoding(typ) {
		return nil
	}
	if plan, ok := b.plans[typ]; ok {
		return plan
	}

	switch typ.Kind() {
	case reflect.Struct:
		plan := &typePlan{kind: objectPlan}
		b.plans[typ] = plan // stored before the fields, so a recursive field reuses it
		b.addFields(plan, typ)
		return plan
	case reflect.Slice, reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			return nil // byte slices and arrays are JSON strings
		}
		if elem := b.plan(typ.Elem()); elem != nil {
			return &typePlan{kind: listPlan, elem: elem}
		}
	case reflect.Map:
		if elem := b.plan(typ.Elem()); elem != nil {
			return &typePlan{kind: mapPlan, elem: elem}
		}
	}
	return nil
}

// addFields adds the fields of the struct typ to plan. The fields of embedded structs
// are added as if they were declared in typ, as encoding/json/v2 promotes them.
func (b *planBuilder) addFields(plan *typePlan, typ reflect.Type) {
	for field := range typ.Fields() {
		if tags.IsOmitted(field) {
			continue
		}
		if tags.IsEmbedded(field) {
			embedded := tags.Dereference(field.Type)
			if embedded.Kind() == reflect.Struct && !HasCustomDecoding(embedded) {
				b.addFields(plan, embedded)
			}
			continue
		}
		if field.PkgPath != "" {
			continue
		}

		plan.fields = append(plan.fields, fieldPlan{
			key:        tags.FieldName(field),
			ignoreCase: tags.IgnoresCase(field),
			required:   !tags.IsOptional(field),
			plan:       b.plan(field.Type),
		})
	}
}

// HasCustomDecoding reports whether encoding/json/v2 decodes typ with its own method.
func HasCustomDecoding(typ reflect.Type) bool {
	pointer := reflect.PointerTo(typ)
	return pointer.Implements(jsonUnmarshalerType) ||
		pointer.Implements(jsonUnmarshalerFromType) ||
		pointer.Implements(textUnmarshalerType)
}
