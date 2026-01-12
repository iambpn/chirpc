package tsGen

/*
Convert Go structs to TypeScript interfaces.
*/

import (
	"errors"
	"fmt"
	"reflect"

	orderedmap "github.com/elliotchance/orderedmap/v3"
	"github.com/iambpn/chirpc/internal/stringUtils"
	"github.com/iambpn/chirpc/internal/tsGen/tsInterface"
	"github.com/iambpn/chirpc/internal/tsGen/tsopts"
)

// GenericAny is an alias for any type, used for generic purposes.
type GenericAny any

// TsGen is responsible for converting Go structs to TypeScript interfaces.
type TsGen struct {
	opt     tsopts.TsGenOpts
	builder *tsInterface.TsInterfaceBuilder
}

// structTagKey is the struct tag used to override the TypeScript property name.
const structTagKey = "tsKey"

// structTagType is the struct tag used to override the TypeScript type.
const structTagType = "tsType"

// structTagOptional is the struct tag used to specify if the TypeScript property is optional.
const structTagOptional = "tsOptional"

// structTagOmit is the struct tag used to ignore a field in TypeScript generation.
const structTagOmit = "tsOmit"

// GetType returns the TypeScript type string for a given Go struct field.
func (t *TsGen) GetType(field reflect.StructField, opts ...tsopts.TsGenOpts) string {
	allOpts := append([]tsopts.TsGenOpts{t.opt}, opts...)
	mergedOpt := tsopts.MergeOpts(allOpts...)

	switch field.Type.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Slice, reflect.Array:
		elemType := t.GetType(reflect.StructField{Type: field.Type.Elem()}, mergedOpt)
		return fmt.Sprintf("(%s)[]", elemType)
	case reflect.Map:
		keyType := t.GetType(reflect.StructField{Type: field.Type.Key()}, mergedOpt)
		valueType := t.GetType(reflect.StructField{Type: field.Type.Elem()}, mergedOpt)
		return fmt.Sprintf("{ [key: %s]: %s }", keyType, valueType)
	case reflect.Struct:
		return t.handleNestedStruct(field, mergedOpt)
	case reflect.Pointer:
		elemType := t.GetType(reflect.StructField{Type: field.Type.Elem()}, mergedOpt)
		return elemType + " | null"
	case reflect.Func:
		return "Function"
	case reflect.Interface, reflect.Chan, reflect.Complex64, reflect.Complex128:
		return "any"
	case reflect.Invalid:
		return "unknown"
	default:
		return "unknown"
	}
}

// AddValue registers a Go value's type for TypeScript interface generation.
func (t *TsGen) AddValue(val any, opts ...tsopts.TsGenOpts) error {
	allOpts := append([]tsopts.TsGenOpts{t.opt}, opts...)
	mergedOpt := tsopts.MergeOpts(allOpts...)

	valType := reflect.TypeOf(val)

	if valType.Kind() == reflect.Pointer {
		valType = valType.Elem()
	}

	err := t.registerStruct(valType, "", mergedOpt)
	if err != nil {
		return err
	}

	return nil
}

// AddValueWithName registers a Go value's type with a custom TypeScript interface name.
func (t *TsGen) AddValueWithName(val any, headerName string, opts ...tsopts.TsGenOpts) error {
	allOpts := append([]tsopts.TsGenOpts{t.opt}, opts...)
	mergedOpt := tsopts.MergeOpts(allOpts...)

	valType := reflect.TypeOf(val)

	if valType.Kind() == reflect.Pointer {
		valType = valType.Elem()
	}

	err := t.registerStruct(valType, headerName, mergedOpt)
	if err != nil {
		return err
	}

	return nil
}

// AddType registers a Go type for TypeScript interface generation.
func (t *TsGen) AddType(valType reflect.Type, opts ...tsopts.TsGenOpts) error {
	allOpts := append([]tsopts.TsGenOpts{t.opt}, opts...)
	mergedOpt := tsopts.MergeOpts(allOpts...)

	err := t.registerStruct(valType, "", mergedOpt)
	if err != nil {
		return err
	}

	return nil
}

// AddTypeWithName registers a Go type with a custom TypeScript interface name.
func (t *TsGen) AddTypeWithName(valType reflect.Type, headerName string, opts ...tsopts.TsGenOpts) error {
	allOpts := append([]tsopts.TsGenOpts{t.opt}, opts...)
	mergedOpt := tsopts.MergeOpts(allOpts...)

	err := t.registerStruct(valType, headerName, mergedOpt)
	if err != nil {
		return err
	}

	return nil
}

// String returns the generated TypeScript interfaces as a string.
func (t *TsGen) String() string {
	return t.builder.String()
}

// GetRegisteredTypes returns all registered TypeScript interfaces.
func (t *TsGen) GetRegisteredTypes() *orderedmap.OrderedMap[string, *tsInterface.TsInterface] {
	return t.builder.GetTypes()
}

// registerStruct registers a Go struct type for TypeScript interface generation.
func (t *TsGen) registerStruct(valType reflect.Type, headerName string, opt tsopts.TsGenOpts) error {
	if valType.Kind() != reflect.Struct {
		return errors.New("cannot build TS type for non-struct types")
	}

	tsInf := t.buildTsStruct(valType, headerName, opt)
	t.builder.RegisterType(tsInf)

	return nil
}

// buildTsStruct builds a TypeScript interface from a Go struct type.
func (t *TsGen) buildTsStruct(valType reflect.Type, headerName string, opt tsopts.TsGenOpts) *tsInterface.TsInterface {
	if headerName == "" {
		headerName = getHeaderName(valType, opt)
	}
	// check if the type has already been generated
	if tsInf, exists := t.builder.GetType(headerName); exists {
		return tsInf
	}

	tsInf := tsInterface.New(false)
	tsInf.SetPrimary(true)
	tsInf.AddInterfaceName(headerName)

	for i := 0; i < valType.NumField(); i++ {
		structField := valType.Field(i)

		// check if the field is not exported
		// pkgPath is non-empty for unexported fields
		if structField.PkgPath != "" {
			continue
		}

		// if field is marked to be omitted skip it
		if isFieldOmitted(structField) {
			continue
		}

		tsType := getTsTypeTagValue(structField)
		if tsType == "" {
			tsType = t.GetType(structField, opt)
		}

		formattedKey := getTsKeyTagValue(structField)

		if formattedKey == "" {
			formattedKey = stringUtils.ShouldToLower(structField.Name, opt[tsopts.ToLowercase])
		}

		tsInf.AddProperty(formattedKey, tsType, isFieldOptional(structField))
	}

	return tsInf
}

// handleNestedStruct handles nested struct field
// for named struct new interface need to be created
// for anonymous struct interface is directly embedded into the parent interface
func (t *TsGen) handleNestedStruct(structField reflect.StructField, opt tsopts.TsGenOpts) string {
	if structField.Type.Name() != "" && structField.Type.Kind() == reflect.Struct {

		// Special handling for time.Time - it serializes to string in JSON
		if structField.Type.PkgPath() == "time" && structField.Type.Name() == "Time" {
			tsType := "string"
			return tsType
		}

		mergedOpt := tsopts.MergeOpts(
			tsopts.SetToLowercaseExportedField(opt[tsopts.ToLowercase]),
			tsopts.SetAddHeaderToInterface(true),
		)

		nestedHeaderName := getTsTypeTagValue(structField)
		if nestedHeaderName == "" {
			nestedHeaderName = getHeaderName(structField.Type, mergedOpt)
		}

		// check if the nested struct type has already been generated
		if _, exists := t.builder.GetType(nestedHeaderName); exists {
			return nestedHeaderName
		}

		// generate new interface for nested struct
		childTsInf := t.buildTsStruct(structField.Type, "", mergedOpt)
		childTsInf.SetPrimary(false)
		t.builder.RegisterType(childTsInf)

		return nestedHeaderName
	} else {
		anonymousTsInf := t.buildTsStruct(structField.Type, "", tsopts.SetAddHeaderToInterface(false))
		return anonymousTsInf.String()
	}
}

// New creates a new TsGen instance with the provided options.
func New(options ...tsopts.TsGenOpts) *TsGen {
	opt := tsopts.MergeOpts(options...)

	// default values
	if _, exists := opt[tsopts.AddHeaderToInterface]; !exists {
		opt[tsopts.AddHeaderToInterface] = true
	}

	return &TsGen{
		opt:     opt,
		builder: tsInterface.NewBuilder(),
	}
}
