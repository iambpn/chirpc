package rpc

import (
	"fmt"
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v3"
	"github.com/iambpn/chirpc/internal/tsGen"
	"github.com/iambpn/chirpc/internal/tsGen/tsopts"
)

// RouterRpcSchemas manages a collection of handler schemas and
// generates TypeScript types for registered RPC endpoints.
type RouterRpcSchemas struct {
	schemas []*HandlerSchema
}

// RegisterHandlers adds multiple HandlerSchema entries to the
// RPC schema collection for TypeScript type generation.
//
// If the input slice is empty, the function does nothing.
func (r *RouterRpcSchemas) RegisterHandlers(schemas []*HandlerSchema) {
	if len(schemas) == 0 {
		return
	}

	// remove the error handler schema if it exists cause
	// we only want one global error handler
	var filteredSchemas []*HandlerSchema
	for _, schema := range schemas {
		if schema.method == "ERROR_HANDLER" {
			continue
		}
		filteredSchemas = append(filteredSchemas, schema)
	}

	// add to global types slice for type generation
	r.schemas = append(r.schemas, filteredSchemas...)
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

	// replace existing error handler schema if exists
	if method == "ERROR_HANDLER" {
		for i, existingSchema := range r.schemas {
			if existingSchema.method == "ERROR_HANDLER" {
				r.schemas[i] = schema
				return schema, nil
			}
		}
	}

	r.schemas = append(r.schemas, schema)

	return schema, nil
}

// ConvertToTs generates consolidated TypeScript interfaces and RPC schema mappings
// for all registered handlers. It returns the combined TypeScript code.
func (r *RouterRpcSchemas) ConvertToTs() (string, error) {
	eps := NewEndpointSchema(true)
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
					// For return type: tsInf is the TS representation of HttpResponse[T]
					// see above registerHandler function
					body, err := tsInf.GetProperty("Body")

					if err != nil {
						return "", err
					}

					schema.Response = body.Value
				}
			case "bodyType":
				{
					// for body type: tsInf is the TS representation of the Struct
					// see setBodyType function

					// removing header name to build anonymous interface
					tsInf.AddInterfaceName("")

					body := tsInf.String()
					schema.Body = body
				}
			case "queryType":
				{
					// for query type: tsInf is the TS representation of the Struct
					// see setQueryType function

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

		if t.StreamType() != "" {
			schema.Stream = t.StreamType()
			schema.Response = fmt.Sprintf("AsyncIterable<%s>", schema.Response)
		}

		eps.AddRpcSchema(t.method, t.url, schema)
	}

	tsTypes := []string{}
	for el := globalTsTypes.Front(); el != nil; el = el.Next() {
		tsTypes = append(tsTypes, el.Value)
	}

	return fmt.Sprintf("%s\n%s", strings.Join(tsTypes, "\n"), eps.String()), nil
}

// NewRouterRpcSchemas creates a new RouterRpcSchemas instance with an empty schema collection.
func NewRouterRpcSchemas() *RouterRpcSchemas {
	return &RouterRpcSchemas{
		schemas: []*HandlerSchema{},
	}
}
