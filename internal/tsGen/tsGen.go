package tsGen

import (
	"errors"
	"fmt"
	"reflect"

	orderedmap "github.com/elliotchance/orderedmap/v3"
	"github.com/iambpn/chirpc/internal/stringUtils"
	"github.com/iambpn/chirpc/internal/tsGen/tsInterface"
	"github.com/iambpn/chirpc/internal/tsGen/tsopts"
)

type GenericAny any

type TsGen struct {
	opt     tsopts.TsGenOpts
	builder *tsInterface.TsInterfaceBuilder
}

const structTagKey = "tsKey"
const structTagType = "tsType"
const structTagOptional = "tsOptional"

func (t *TsGen) GetType(field reflect.StructField) string {
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
		elemType := t.GetType(reflect.StructField{Type: field.Type.Elem()})
		return elemType + "[]"
	case reflect.Map:
		keyType := t.GetType(reflect.StructField{Type: field.Type.Key()})
		valueType := t.GetType(reflect.StructField{Type: field.Type.Elem()})
		return fmt.Sprintf("{ [key: %s]: %s }", keyType, valueType)
	case reflect.Struct:
		// handel nested anonymous struct
		if field.Type.Name() == "" {
			anonymousTsInf := t.buildTsStruct(field.Type, "", tsopts.SetAddHeaderToInterface(false))
			return anonymousTsInf.String()
		}

		headerName := getHeaderName(field.Type, t.opt)
		if _, exists := t.builder.QueryType(headerName); !exists {
			nestedTsInf := t.buildTsStruct(field.Type, "", tsopts.SetAddHeaderToInterface(true))
			nestedTsInf.SetPrimary(false)
			t.builder.RegisterType(nestedTsInf)
		}

		return headerName
	case reflect.Pointer:
		elemType := t.GetType(reflect.StructField{Type: field.Type.Elem()})
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

func (t *TsGen) AddValue(val any, opts ...tsopts.TsGenOpts) error {
	valType := reflect.TypeOf(val)

	err := t.registerStruct(valType, "", opts...)
	if err != nil {
		return err
	}

	return nil
}

func (t *TsGen) AddValueWithName(val any, headerName string, opts ...tsopts.TsGenOpts) error {
	valType := reflect.TypeOf(val)

	err := t.registerStruct(valType, headerName, opts...)
	if err != nil {
		return err
	}

	return nil
}

func (t *TsGen) AddType(valType reflect.Type, opts ...tsopts.TsGenOpts) error {
	err := t.registerStruct(valType, "", opts...)
	if err != nil {
		return err
	}

	return nil
}

func (t *TsGen) AddTypeWithName(valType reflect.Type, headerName string, opts ...tsopts.TsGenOpts) error {
	err := t.registerStruct(valType, headerName, opts...)
	if err != nil {
		return err
	}

	return nil
}

func (t *TsGen) String() string {
	return t.builder.String()
}

func (t *TsGen) GetRegisteredTypes() *orderedmap.OrderedMap[string, *tsInterface.TsInterface] {
	return t.builder.GetTypes()
}

func (t *TsGen) registerStruct(valType reflect.Type, headerName string, opts ...tsopts.TsGenOpts) error {
	allOpts := append([]tsopts.TsGenOpts{t.opt}, opts...)
	opt := tsopts.MergeOpts(allOpts...)

	if valType.Kind() != reflect.Struct {
		return errors.New("cannot build TS type for non-struct types")
	}

	tsInf := t.buildTsStruct(valType, headerName, opt)
	t.builder.RegisterType(tsInf)

	return nil
}

func (t *TsGen) buildTsStruct(valType reflect.Type, headerName string, opt tsopts.TsGenOpts) *tsInterface.TsInterface {
	if headerName == "" {
		headerName = getHeaderName(valType, opt)
	}
	// check if the type has already been generated
	if tsInf, exists := t.builder.QueryType(headerName); exists {
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

		// handle named nested struct field
		if structField.Type.Name() != "" && structField.Type.Kind() == reflect.Struct {
			// to check for cache header is required
			newOpt := tsopts.MergeOpts(
				tsopts.SetToLowerExportedField(t.opt[tsopts.ToLowercase]),
				tsopts.SetAddHeaderToInterface(true),
			)

			nestedHeaderName := getTagType(structField)
			if nestedHeaderName == "" {
				nestedHeaderName = getHeaderName(structField.Type, newOpt)
			}

			// check if the nested struct type has already been generated
			if _, exists := t.builder.QueryType(nestedHeaderName); exists {
				formattedKey := getTagKey(structField)

				if formattedKey == "" {
					formattedKey = stringUtils.ShouldToLower(structField.Name, opt[tsopts.ToLowercase])
				}

				tsInf.AddProperty(formattedKey, nestedHeaderName, isFieldOptional(structField))
				continue
			}

			childTsInf := t.buildTsStruct(structField.Type, "", newOpt)
			childTsInf.SetPrimary(false)
			t.builder.RegisterType(childTsInf)

			formattedKey := getTagKey(structField)

			if formattedKey == "" {
				formattedKey = stringUtils.ShouldToLower(structField.Name, opt[tsopts.ToLowercase])
			}

			tsInf.AddProperty(formattedKey, nestedHeaderName, isFieldOptional(structField))
			continue
		}

		tsType := getTagType(structField)
		if tsType == "" {
			tsType = t.GetType(structField)
		}

		formattedKey := getTagKey(structField)

		if formattedKey == "" {
			formattedKey = stringUtils.ShouldToLower(structField.Name, opt[tsopts.ToLowercase])
		}

		tsInf.AddProperty(formattedKey, tsType, isFieldOptional(structField))
	}

	return tsInf
}

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
