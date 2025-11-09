package rpcType

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v3"
	"github.com/iambpn/chirpc/internal/tsGen"
	"github.com/iambpn/chirpc/internal/tsGen/tsopts"
)

/*
Convert Go types used in RPC handlers to TypeScript interfaces.
*/

// TsGoSchema represents RPC handler metadata used to generate TypeScript types.
// It stores HTTP method, URL, and Go types for return, body, query, and path params.
type TsGoSchema struct {
	method     string
	url        string
	returnType reflect.Type
	bodyType   reflect.Type
	paramsType string
	queryType  reflect.Type
}

// types holds all registered TsGoSchema entries pending TypeScript conversion.
var types = []*TsGoSchema{}

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

// RegisterHandler registers an RPC handler with its method, URL, and return type.
// It returns a TsGoSchema for optional body, query, and params enrichment.
func RegisterHandler(method, url string, fnVal any) (*TsGoSchema, error) {
	typeVal := reflect.TypeOf(fnVal)

	retType, err := extractReturnType(typeVal)
	if err != nil {
		return nil, err
	}

	schema := TsGoSchema{
		method:     method,
		url:        url,
		returnType: retType,
	}

	types = append(types, &schema)

	return &schema, nil
}

// SetBodyType assigns a struct type (value or pointer) as the request body type.
// Non-struct inputs are ignored with a warning to stderr.
func SetBodyType(schema *TsGoSchema, body any) {
	bodyType := reflect.TypeOf(body)

	if bodyType.Kind() == reflect.Pointer {
		bodyType = bodyType.Elem()
	}

	if bodyType.Kind() != reflect.Struct {
		fmt.Fprintf(os.Stderr, "Warning: body type must be a struct, got %s, skipping setting body type\n", bodyType.String())
		return
	}

	schema.bodyType = bodyType
}

// SetQueryType assigns a struct type (value or pointer) as the query type.
// Non-struct inputs are ignored with a warning to stderr.
func SetQueryType(schema *TsGoSchema, query any) {
	queryType := reflect.TypeOf(query)

	if queryType.Kind() == reflect.Pointer {
		queryType = queryType.Elem()
	}

	if queryType.Kind() != reflect.Struct {
		fmt.Fprintf(os.Stderr, "Warning: query type is not a struct (got %s), skipping setting query type\n", queryType.String())
		return
	}

	schema.queryType = queryType
}

// SetParamsType converts path param slugs into a TypeScript interface shape and stores it on the schema.
func SetParamsType(schema *TsGoSchema, slugs []string) {
	schema.paramsType = sliceToTsInf(slugs)
}

// ConvertToTs generates consolidated TypeScript interfaces and RPC schema mappings
// for all registered handlers. It returns the combined TypeScript code.
func ConvertToTs() (string, error) {
	rt := NewRpcType(true)
	globalTsTypes := orderedmap.NewOrderedMap[string, string]()

	for _, t := range types {
		tsgen := tsGen.New()

		// add return type to tsgen
		err := tsgen.AddTypeWithName(t.returnType, "returnType", tsopts.TsGenOpts{})

		if err != nil {
			return "", err
		}

		// add body type to tsgen if exists
		if t.bodyType != nil {
			err = tsgen.AddTypeWithName(t.bodyType, "bodyType", tsopts.TsGenOpts{})

			if err != nil {
				return "", err
			}
		}

		// add query type to tsgen if exists
		if t.queryType != nil {
			err = tsgen.AddTypeWithName(t.queryType, "queryType", tsopts.TsGenOpts{})
			if err != nil {
				return "", err
			}
		}

		// fetched all registered types so we can check if body, param, query types exist
		registeredTypes := tsgen.GetRegisteredTypes()

		schema := RpcSchema{}

		for el := registeredTypes.Front(); el != nil; el = el.Next() {
			name := el.Key // headerName
			tsInf := el.Value
			switch name {
			case "returnType":
				{
					// For return type: tsInf is the ts representation of HttpResponse[T]
					// see above registerHandler function
					body, err := tsInf.GetProperty("Body")

					if err != nil {
						return "", err
					}

					schema.Response = body.Value
				}
			case "bodyType":
				{
					// for body type: tsInf is the ts representation of the struct
					// see above setBodyType function

					tsInf.AddInterfaceName("")

					body := tsInf.String()
					schema.Body = body
				}
			case "queryType":
				{
					// for query type: tsInf is the ts representation of the struct
					// see above setQueryType function

					// removing header name to build anonymous interface
					tsInf.AddInterfaceName("")

					query := tsInf.String()
					schema.Query = query
				}
			default:
				{
					if _, exists := globalTsTypes.Get(name); !exists {
						globalTsTypes.Set(name, tsInf.String())
					}
				}
			}
		}

		// set params
		if t.paramsType != "" {
			schema.Param = t.paramsType
		}

		rt.AddRpcSchema(t.method, t.url, schema)
	}

	tsTypes := []string{}
	for el := globalTsTypes.Front(); el != nil; el = el.Next() {
		tsTypes = append(tsTypes, el.Value)
	}

	return fmt.Sprintf("%s\n%s", strings.Join(tsTypes, "\n"), rt.String()), nil
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
