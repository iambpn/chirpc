package typescript

import (
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/coder/guts"
	"github.com/coder/guts/bindings"
)

const (
	structTagKey      = "tsKey"
	structTagType     = "tsType"
	structTagOptional = "tsOptional"
	structTagOmit     = "tsOmit"
)

// gutsConverter translates runtime types into guts' TypeScript AST. chirpc
// discovers endpoint types through reflection, so it cannot use guts' source
// package parser directly; using the same AST and TypeScript compiler-backed
// serializer preserves chirpc's runtime registration API.
type gutsConverter struct {
	renderer *bindings.Bindings

	declarations     map[reflect.Type]bindings.Node
	declarationNames map[string]reflect.Type
	declarationOrder []reflect.Type
	overrides        []typeOverride
}

type typeOverride struct {
	placeholder string
	value       string
}

func newGutsConverter() (*gutsConverter, error) {
	renderer, err := bindings.New()
	if err != nil {
		return nil, fmt.Errorf("initialize guts TypeScript renderer: %w", err)
	}

	return &gutsConverter{
		renderer:         renderer,
		declarations:     make(map[reflect.Type]bindings.Node),
		declarationNames: make(map[string]reflect.Type),
		declarationOrder: make([]reflect.Type, 0),
		overrides:        make([]typeOverride, 0),
	}, nil
}

func (c *gutsConverter) responseType(response reflect.Type) (string, error) {
	response = dereference(response)
	if response == nil || response.Kind() != reflect.Struct {
		return "", fmt.Errorf("response type must be a struct")
	}

	body, ok := response.FieldByName("Body")
	if !ok || body.PkgPath != "" {
		return "", fmt.Errorf("response type %s has no exported Body field", response)
	}

	expression, err := c.fieldType(body)
	if err != nil {
		return "", fmt.Errorf("convert response body: %w", err)
	}
	return c.serialize(expression)
}

func (c *gutsConverter) inlineStruct(typ reflect.Type) (string, error) {
	typ = dereference(typ)
	if typ == nil || typ.Kind() != reflect.Struct {
		return "", fmt.Errorf("inline type must be a struct")
	}

	expression, err := c.structExpression(typ)
	if err != nil {
		return "", err
	}
	return c.serialize(expression)
}

func (c *gutsConverter) declarationStrings() ([]string, error) {
	result := make([]string, 0, len(c.declarationOrder))
	for _, typ := range c.declarationOrder {
		declaration, ok := c.declarations[typ]
		if !ok {
			return nil, fmt.Errorf("missing declaration for %s", typ)
		}

		serialized, err := c.serialize(declaration)
		if err != nil {
			return nil, fmt.Errorf("serialize declaration %s: %w", typ, err)
		}
		result = append(result, serialized)
	}
	return result, nil
}

func (c *gutsConverter) typeExpression(typ reflect.Type) (bindings.ExpressionType, error) {
	if typ == nil {
		return keyword(bindings.KeywordUnknown), nil
	}

	if expression, ok := standardTypeMapping(typ); ok {
		return expression, nil
	}

	if typ.Name() != "" && typ.PkgPath() != "" {
		return c.namedType(typ)
	}

	return c.underlyingTypeExpression(typ)
}

func (c *gutsConverter) underlyingTypeExpression(typ reflect.Type) (bindings.ExpressionType, error) {
	switch typ.Kind() {
	case reflect.Bool:
		return keyword(bindings.KeywordBoolean), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return keyword(bindings.KeywordNumber), nil
	case reflect.String:
		return keyword(bindings.KeywordString), nil
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return keyword(bindings.KeywordString), nil
		}
		element, err := c.typeExpression(typ.Elem())
		if err != nil {
			return nil, fmt.Errorf("slice element: %w", err)
		}
		return bindings.Array(element), nil
	case reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			return keyword(bindings.KeywordString), nil
		}
		element, err := c.typeExpression(typ.Elem())
		if err != nil {
			return nil, fmt.Errorf("array element: %w", err)
		}
		return bindings.HomogeneousTuple(typ.Len(), element), nil
	case reflect.Map:
		key, err := c.typeExpression(typ.Key())
		if err != nil {
			return nil, fmt.Errorf("map key: %w", err)
		}
		value, err := c.typeExpression(typ.Elem())
		if err != nil {
			return nil, fmt.Errorf("map value: %w", err)
		}
		return bindings.Union(guts.RecordReference(key, value), &bindings.Null{}), nil
	case reflect.Struct:
		return c.structExpression(typ)
	case reflect.Pointer:
		element, err := c.typeExpression(typ.Elem())
		if err != nil {
			return nil, fmt.Errorf("pointer element: %w", err)
		}
		return nullable(element), nil
	case reflect.Interface:
		return keyword(bindings.KeywordUnknown), nil
	case reflect.Func, reflect.Chan, reflect.UnsafePointer, reflect.Invalid:
		return keyword(bindings.KeywordUnknown), nil
	default:
		return nil, fmt.Errorf("unsupported Go type %s", typ)
	}
}

func (c *gutsConverter) namedType(typ reflect.Type) (bindings.ExpressionType, error) {
	if expression, ok := standardTypeMapping(typ); ok {
		return expression, nil
	}
	if _, ok := c.declarations[typ]; ok {
		return bindings.Reference(bindings.Identifier{Name: c.typeName(typ)}), nil
	}

	name := c.typeName(typ)
	if owner, exists := c.declarationNames[name]; exists && owner != typ {
		return nil, fmt.Errorf("TypeScript name %q is shared by %s and %s", name, owner, typ)
	}
	c.declarationNames[name] = typ
	c.declarationOrder = append(c.declarationOrder, typ)

	identifier := bindings.Identifier{Name: name}
	if typ.Kind() == reflect.Struct {
		declaration := &bindings.Interface{Name: identifier}
		c.declarations[typ] = declaration // register before walking recursive fields

		fields, heritage, err := c.structMembers(typ)
		if err != nil {
			return nil, err
		}
		declaration.Fields = fields
		if len(heritage) > 0 {
			declaration.Heritage = []*bindings.HeritageClause{
				bindings.HeritageClauseExtends(heritage...),
			}
		}
	} else {
		declaration := &bindings.Alias{Name: identifier}
		c.declarations[typ] = declaration

		expression, err := c.underlyingTypeExpression(typ)
		if err != nil {
			return nil, err
		}
		declaration.Type = expression
	}

	return bindings.Reference(identifier), nil
}

func (c *gutsConverter) structExpression(typ reflect.Type) (bindings.ExpressionType, error) {
	fields, heritage, err := c.structMembers(typ)
	if err != nil {
		return nil, err
	}

	var literal bindings.ExpressionType = &bindings.TypeLiteralNode{Members: fields}
	if len(heritage) == 0 {
		return literal, nil
	}

	types := append(heritage, literal)
	return &bindings.TypeIntersection{Types: types}, nil
}

func (c *gutsConverter) structMembers(typ reflect.Type) ([]*bindings.PropertySignature, []bindings.ExpressionType, error) {
	fields := make([]*bindings.PropertySignature, 0, typ.NumField())
	heritage := make([]bindings.ExpressionType, 0)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if isFieldOmitted(field) {
			continue
		}

		if field.Anonymous && field.Tag.Get("json") == "" {
			embedded := dereference(field.Type)
			if embedded == nil || embedded.Kind() != reflect.Struct || embedded.Name() == "" {
				return nil, nil, fmt.Errorf("embedded field %s is not a named struct", field.Name)
			}
			expression, err := c.namedType(embedded)
			if err != nil {
				return nil, nil, fmt.Errorf("embedded field %s: %w", field.Name, err)
			}
			heritage = append(heritage, expression)
			continue
		}
		if field.PkgPath != "" {
			continue
		}

		expression, err := c.fieldType(field)
		if err != nil {
			return nil, nil, fmt.Errorf("field %s: %w", field.Name, err)
		}

		fields = append(fields, &bindings.PropertySignature{
			Name:          fieldName(field),
			QuestionToken: isFieldOptional(field),
			Type:          expression,
		})
	}

	return fields, heritage, nil
}

func (c *gutsConverter) fieldType(field reflect.StructField) (bindings.ExpressionType, error) {
	if override := field.Tag.Get(structTagType); override != "" {
		placeholder := fmt.Sprintf("__ChirpcTsOverride_%d__", len(c.overrides))
		c.overrides = append(c.overrides, typeOverride{placeholder: placeholder, value: override})
		return bindings.Reference(bindings.Identifier{Name: placeholder}), nil
	}

	expression, err := c.typeExpression(field.Type)
	if err != nil {
		return nil, err
	}
	if hasJSONOption(field, "string") {
		return jsonStringExpression(field.Type), nil
	}
	return expression, nil
}

func (c *gutsConverter) serialize(node bindings.Node) (string, error) {
	object, err := c.renderer.ToTypescriptNode(node)
	if err != nil {
		return "", fmt.Errorf("build TypeScript AST: %w", err)
	}

	result, err := c.renderer.SerializeToTypescript(object)
	if err != nil {
		return "", fmt.Errorf("serialize TypeScript AST: %w", err)
	}
	for _, override := range c.overrides {
		result = strings.ReplaceAll(result, override.placeholder, override.value)
	}
	return result, nil
}

// typeName returns the TypeScript name of a named Go type. Types outside package main
// are prefixed with their package name, so the same type name in two packages stays distinct.
func (c *gutsConverter) typeName(typ reflect.Type) string {
	pkg := typ.PkgPath()
	name := exportedIdentifier(typ.Name())
	if pkg == "" || strings.EqualFold(pkg, "main") {
		return name
	}
	return exportedIdentifier(packageName(pkg)) + "__" + name
}

// packageName guesses the package name from an import path. It uses the last path
// element, or the one before it when the last is a major version such as "v2",
// which is how Go names packages in versioned module paths.
func packageName(importPath string) string {
	elements := strings.Split(importPath, "/")
	name := elements[len(elements)-1]
	if len(elements) > 1 && isMajorVersion(name) {
		name = elements[len(elements)-2]
	}
	return name
}

// isMajorVersion reports whether element looks like "v1", "v2", and so on.
func isMajorVersion(element string) bool {
	if len(element) < 2 || element[0] != 'v' {
		return false
	}
	for _, r := range element[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func fieldName(field reflect.StructField) string {
	if name := field.Tag.Get(structTagKey); name != "" {
		return name
	}
	if name, _ := jsonTag(field); name != "" {
		return name
	}
	return field.Name
}

func isFieldOptional(field reflect.StructField) bool {
	return strings.EqualFold(field.Tag.Get(structTagOptional), "true") ||
		hasJSONOption(field, "omitempty") || hasJSONOption(field, "omitzero")
}

// hasRequiredField reports whether the generated type for the struct typ has a required
// member. It follows the same rules as structMembers, including embedded structs.
// seen holds the embedded structs already checked, to stop on recursive embedding.
func hasRequiredField(typ reflect.Type, seen map[reflect.Type]bool) bool {
	typ = dereference(typ)
	if typ == nil || typ.Kind() != reflect.Struct || seen[typ] {
		return false
	}
	seen[typ] = true

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if isFieldOmitted(field) {
			continue
		}
		if field.Anonymous && field.Tag.Get("json") == "" {
			if hasRequiredField(field.Type, seen) {
				return true
			}
			continue
		}
		if field.PkgPath != "" {
			continue
		}
		if !isFieldOptional(field) {
			return true
		}
	}
	return false
}

func isFieldOmitted(field reflect.StructField) bool {
	name, _ := jsonTag(field)
	return name == "-" || strings.EqualFold(field.Tag.Get(structTagOmit), "true") ||
		field.Tag.Get("typescript") == "-"
}

func jsonTag(field reflect.StructField) (string, []string) {
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return "", nil
	}
	parts := strings.Split(tag, ",")
	return parts[0], parts[1:]
}

func hasJSONOption(field reflect.StructField, option string) bool {
	_, options := jsonTag(field)
	for _, candidate := range options {
		if candidate == option {
			return true
		}
	}
	return false
}

func jsonStringExpression(typ reflect.Type) bindings.ExpressionType {
	nullableType := false
	for typ.Kind() == reflect.Pointer {
		nullableType = true
		typ = typ.Elem()
	}
	expression := bindings.ExpressionType(keyword(bindings.KeywordString))
	if nullableType {
		expression = nullable(expression)
	}
	return expression
}

func nullable(expression bindings.ExpressionType) bindings.ExpressionType {
	if union, ok := expression.(*bindings.UnionType); ok {
		for _, member := range union.Types {
			if _, isNull := member.(*bindings.Null); isNull {
				return expression
			}
		}
	}
	return bindings.Union(expression, &bindings.Null{})
}

func keyword(value bindings.LiteralKeyword) *bindings.LiteralKeyword {
	return &value
}

func dereference(typ reflect.Type) reflect.Type {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

func standardTypeMapping(typ reflect.Type) (bindings.ExpressionType, bool) {
	if typ == reflect.TypeOf(time.Time{}) {
		return keyword(bindings.KeywordString), true
	}

	name := typ.PkgPath() + "." + typ.Name()
	switch name {
	case "time.Duration":
		return keyword(bindings.KeywordNumber), true
	case "database/sql.NullTime":
		return nullable(keyword(bindings.KeywordString)), true
	case "database/sql.NullString":
		return nullable(keyword(bindings.KeywordString)), true
	case "database/sql.NullBool":
		return nullable(keyword(bindings.KeywordBoolean)), true
	case "database/sql.NullInt64", "database/sql.NullInt32", "database/sql.NullInt16", "database/sql.NullFloat64":
		return nullable(keyword(bindings.KeywordNumber)), true
	case "github.com/google/uuid.UUID", "net/netip.Addr", "net/url.URL", "regexp.Regexp":
		return keyword(bindings.KeywordString), true
	case "github.com/google/uuid.NullUUID":
		return nullable(keyword(bindings.KeywordString)), true
	default:
		return nil, false
	}
}

func exportedIdentifier(value string) string {
	var result []rune
	upperNext := true
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$' {
			if len(result) == 0 && unicode.IsDigit(r) {
				result = append(result, '_')
			}
			if upperNext {
				r = unicode.ToUpper(r)
				upperNext = false
			}
			result = append(result, r)
			continue
		}
		upperNext = true
	}
	if len(result) == 0 {
		return "Anonymous"
	}
	return string(result)
}
