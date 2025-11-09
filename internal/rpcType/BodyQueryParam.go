package rpcType

import "fmt"

// BodyQueryParamType holds schema metadata for request body, query string, and path parameters.
// It is used to fluently declare the expected types for an RPC handler.
// A nil Schema causes its mutator methods to become no-ops (with a warning logged).
type BodyQueryParamType struct {
	Schema *TsGoSchema
}

// BodyType registers the concrete Go type (or example instance) that represents
// the HTTP request body for this RPC. If Schema is nil it prints a warning and
// performs no action. Returns the receiver to allow method chaining.
func (b *BodyQueryParamType) BodyType(body any) *BodyQueryParamType {
	if b.Schema == nil {
		fmt.Println("Warning: Cannot set body type because Schema is nil (check for handler registration errors)")
		return b
	}

	SetBodyType(b.Schema, body)
	return b
}

// QueryType registers the concrete Go type (or example instance) that represents
// the URL query string parameters for this RPC. If Schema is nil it prints a warning
// and performs no action. Returns the receiver to allow method chaining.
func (b *BodyQueryParamType) QueryType(query any) *BodyQueryParamType {
	if b.Schema == nil {
		fmt.Println("Warning: Cannot set query type because Schema is nil (check for handler registration errors)")
		return b
	}

	SetQueryType(b.Schema, query)
	return b
}

// Params sets the expected URL path parameter slugs on the underlying schema.
// It is a no-op when slugs is empty or when Schema is nil (a warning is printed).
// Returns the receiver to allow method chaining.
func (b *BodyQueryParamType) Params(slugs []string) *BodyQueryParamType {
	if len(slugs) == 0 {
		return b
	}

	if b.Schema == nil {
		fmt.Println("Warning: Cannot set params type because Schema is nil (check for handler registration errors)")
		return b
	}

	SetParamsType(b.Schema, slugs)
	return b
}
