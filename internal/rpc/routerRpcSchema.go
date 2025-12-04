package rpc

import (
	"fmt"
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v3"
	"github.com/iambpn/chirpc/internal/tsGen"
	"github.com/iambpn/chirpc/internal/tsGen/tsopts"
)

// RouterRpcSchemas manages a collection of handler schemas and provides
// TypeScript type generation for all registered RPC endpoints.
type RouterRpcSchemas struct {
	schemas []*HandlerSchema
}

// RegisterHandlers adds multiple HandlerSchema entries to the schema collection for TypeScript generation.
// If the input slice is empty, the function does nothing.
func (r *RouterRpcSchemas) RegisterHandlers(schemas []*HandlerSchema) {
	if len(schemas) == 0 {
		return
	}

	// add to global types slice for type generation
	r.schemas = append(r.schemas, schemas...)
}

func (r *RouterRpcSchemas) RegisterHandlerFrom(routerSchema *RouterRpcSchemas) {
	r.RegisterHandlers(routerSchema.schemas)
}

// RegisterHandler registers an RPC schema for type generation with its method, URL, and return type.
// It returns a HandlerSchema for optional body, query, and params enrichment.
func (r *RouterRpcSchemas) RegisterHandler(method, url string, fnVal any) (*HandlerSchema, error) {
	schema, err := BuildGoToTsSchema(method, url, fnVal)

	if err != nil {
		return nil, err
	}

	r.schemas = append(r.schemas, schema)

	return schema, nil
}

// ConvertToTs generates consolidated TypeScript interfaces and RPC schema mappings
// for all registered handlers. It returns the combined TypeScript code.
func (r *RouterRpcSchemas) ConvertToTs() (string, error) {
	rt := NewEndpointSchema(true)
	globalTsTypes := orderedmap.NewOrderedMap[string, string]()

	for _, t := range r.schemas {
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

// NewRouterRpcSchemas creates a new RouterRpcSchemas instance with an empty schema collection.
func NewRouterRpcSchemas() *RouterRpcSchemas {
	return &RouterRpcSchemas{
		schemas: []*HandlerSchema{},
	}
}
