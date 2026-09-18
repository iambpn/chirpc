package rpc

import (
	"fmt"
	"strings"
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
	converter, err := newGutsConverter()
	if err != nil {
		return "", err
	}

	for _, t := range r.schemas {
		response, err := converter.responseType(t.returnType)
		if err != nil {
			return "", err
		}

		schema := RpcSchema{Response: response}
		if t.bodyType != nil {
			schema.Body, err = converter.inlineStruct(t.bodyType)
			if err != nil {
				return "", err
			}
		}
		if t.queryType != nil {
			schema.Query, err = converter.inlineStruct(t.queryType)
			if err != nil {
				return "", err
			}
		}

		// set params
		if t.paramsType != "" {
			schema.Param = t.paramsType
		}

		eps.AddRpcSchema(t.method, t.url, schema)
	}

	declarations, err := converter.declarationStrings()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s\n%s", strings.Join(declarations, "\n"), eps.String()), nil
}

// NewRouterRpcSchemas creates a new RouterRpcSchemas instance with an empty schema collection.
func NewRouterRpcSchemas() *RouterRpcSchemas {
	return &RouterRpcSchemas{
		schemas: []*HandlerSchema{},
	}
}
