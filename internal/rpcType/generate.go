package rpcType

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v3"
	"github.com/iambpn/chirpc/internal/tsGen"
	"github.com/iambpn/chirpc/internal/tsGen/tsopts"
)

type TsGoSchema struct {
	method     string
	url        string
	returnType reflect.Type
	bodyType   reflect.Type
	paramsType reflect.Type
	queryType  reflect.Type
}

var types = []TsGoSchema{}

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

func RegisterHandler(method, url string, fnVal any) error {
	typeVal := reflect.TypeOf(fnVal)

	retType, err := extractReturnType(typeVal)
	if err != nil {
		return err
	}

	types = append(types, TsGoSchema{
		returnType: retType,
		method:     method,
		url:        url,
	})

	return nil
}

func ConvertToTs() (string, error) {
	rt := NewRpcType(true)
	globalTsTypes := orderedmap.NewOrderedMap[string, string]()

	for _, t := range types {
		tsgen := tsGen.New()
		err := tsgen.AddTypeWithName(t.returnType, "returnType", tsopts.TsGenOpts{})

		if err != nil {
			return "", err
		}

		registeredTypes := tsgen.GetRegisteredTypes()

		schema := RpcSchema{}

		for el := registeredTypes.Front(); el != nil; el = el.Next() {
			name := el.Key
			tsInf := el.Value
			switch name {
			case "returnType":
				{
					body, err := tsInf.GetProperty("Body")

					if err != nil {
						return "", err
					}

					schema.Response = body.Value
				}
			default:
				{
					if _, exists := globalTsTypes.Get(name); !exists {
						globalTsTypes.Set(name, tsInf.String())
					}
				}
			}
		}

		rt.AddType(t.method, t.url, schema)
	}

	tsTypes := []string{}
	for el := globalTsTypes.Front(); el != nil; el = el.Next() {
		tsTypes = append(tsTypes, el.Value)
	}

	return fmt.Sprintf("%s\n%s", strings.Join(tsTypes, "\n"), rt.String()), nil
}
