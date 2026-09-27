// Package jsonbody finds required fields that are missing from a JSON request body.
// encoding/json leaves a missing field at its zero value without an error, so this
// check runs after decoding. A field is required when it is neither omitted nor
// optional, which is the same rule the generated TypeScript uses.
package jsonbody

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
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

// fieldPlan is one struct field, read from the JSON key key.
type fieldPlan struct {
	key      string
	required bool
	plan     *typePlan
}

var (
	jsonUnmarshalerType = reflect.TypeFor[json.Unmarshaler]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
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

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
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
// JSON null is accepted, because encoding/json accepts it for any type.
func (p *typePlan) check(value any, path string, problems map[string][]string) {
	switch p.kind {
	case objectPlan:
		object, ok := value.(map[string]any)
		if !ok {
			return
		}
		for _, field := range p.fields {
			fieldValue, found := lookup(object, field.key)
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

// lookup finds key in object the way encoding/json does: an exact match first,
// then a case-insensitive match.
func lookup(object map[string]any, key string) (any, bool) {
	if value, ok := object[key]; ok {
		return value, true
	}
	for candidate, value := range object {
		if strings.EqualFold(candidate, key) {
			return value, true
		}
	}
	return nil, false
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
	if typ == nil || hasCustomDecoding(typ) {
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
// are added as if they were declared in typ, as encoding/json promotes them.
func (b *planBuilder) addFields(plan *typePlan, typ reflect.Type) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if tags.IsOmitted(field) {
			continue
		}
		if tags.IsEmbedded(field) {
			embedded := tags.Dereference(field.Type)
			if embedded.Kind() == reflect.Struct && !hasCustomDecoding(embedded) {
				b.addFields(plan, embedded)
			}
			continue
		}
		if field.PkgPath != "" {
			continue
		}

		plan.fields = append(plan.fields, fieldPlan{
			key:      tags.FieldName(field),
			required: !tags.IsOptional(field),
			plan:     b.plan(field.Type),
		})
	}
}

// hasCustomDecoding reports whether encoding/json decodes typ with its own method.
func hasCustomDecoding(typ reflect.Type) bool {
	pointer := reflect.PointerTo(typ)
	return pointer.Implements(jsonUnmarshalerType) || pointer.Implements(textUnmarshalerType)
}
