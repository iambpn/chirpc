package rpc

import (
	"fmt"
	"reflect"
)

// HandlerSchema represents RPC handler metadata used to generate TypeScript types.
// It stores HTTP method, URL, Go types for return, body, and query, and extra path param names.
type HandlerSchema struct {
	method     string
	url        string
	returnType reflect.Type
	bodyType   reflect.Type
	params     []string
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
// It panics for any other input.
func (p *HandlerSchema) SetBodyType(body any) {
	p.bodyType = structType(body, "body")
}

// SetQueryType assigns a struct type (value or pointer) as the query type.
// It panics for any other input.
func (p *HandlerSchema) SetQueryType(query any) {
	p.queryType = structType(query, "query")
}

// structType returns the struct type of value, which may be a struct or a pointer to one.
// It panics for any other value, because the generated schema would silently miss the type.
func structType(value any, use string) reflect.Type {
	typ := reflect.TypeOf(value)
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == nil || typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("chirpc: the %s type must be a struct or a pointer to a struct, but got %T.", use, value))
	}
	return typ
}

// SetParamsType stores extra path param names for this handler.
// Params found in the full route URL are always included, so these are only needed
// for params that the URL pattern does not show.
func (p *HandlerSchema) SetParamsType(slugs []string) {
	p.params = slugs
}

// paramNames returns the path params for the handler served at fullURL.
// It lists the params in fullURL first, then any extra params set with SetParamsType.
func (p *HandlerSchema) paramNames(fullURL string) []string {
	names := parseURLSlugs(fullURL)
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		seen[name] = true
	}
	for _, name := range p.params {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

// NewHandlerSchema creates a new HandlerSchema with the specified method, URL, and return type.
func NewHandlerSchema(method, url string, returnType reflect.Type) *HandlerSchema {
	return &HandlerSchema{
		method:     method,
		url:        url,
		returnType: returnType,
	}
}
