package rpc

import (
	"fmt"
	"os"
	"reflect"
)

// HandlerSchema represents RPC handler metadata used to generate TypeScript types.
// It stores HTTP method, URL, and Go types for return, body, query, and path params.
type HandlerSchema struct {
	method     string
	url        string
	returnType reflect.Type
	bodyType   reflect.Type
	paramsType string
	queryType  reflect.Type
}

// SetUrl sets the URL path for this handler schema.
func (p *HandlerSchema) SetUrl(url string) {
	p.url = url
}

// URL returns the URL path of this handler schema.
func (p *HandlerSchema) URL() string {
	return p.url
}

// SetBodyType assigns a struct type (value or pointer) as the request body type.
// Non-struct inputs are ignored with a warning to stderr.
func (p *HandlerSchema) SetBodyType(body any) {
	bodyType := reflect.TypeOf(body)

	if bodyType.Kind() == reflect.Pointer {
		bodyType = bodyType.Elem()
	}

	if bodyType.Kind() != reflect.Struct {
		fmt.Fprintf(os.Stderr, "Warning: body type must be a struct, got %s, skipping setting body type\n", bodyType.String())
		return
	}

	p.bodyType = bodyType
}

// SetQueryType assigns a struct type (value or pointer) as the query type.
// Non-struct inputs are ignored with a warning to stderr.
func (p *HandlerSchema) SetQueryType(query any) {
	queryType := reflect.TypeOf(query)

	if queryType.Kind() == reflect.Pointer {
		queryType = queryType.Elem()
	}

	if queryType.Kind() != reflect.Struct {
		fmt.Fprintf(os.Stderr, "Warning: query type is not a struct (got %s), skipping setting query type\n", queryType.String())
		return
	}

	p.queryType = queryType
}

// SetParamsType converts path param slugs into a TypeScript interface shape and stores it on the schema.
func (p *HandlerSchema) SetParamsType(slugs []string) {
	p.paramsType = sliceToTsInf(slugs)
}

// NewHandlerSchema creates a new HandlerSchema with the specified method, URL, and return type.
func NewHandlerSchema(method, url string, returnType reflect.Type) *HandlerSchema {
	return &HandlerSchema{
		method:     method,
		url:        url,
		returnType: returnType,
	}
}
