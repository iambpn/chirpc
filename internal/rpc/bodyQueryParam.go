package rpc

// BodyQueryParamType holds schema metadata for request body, query string, and path parameters.
// It is used to fluently declare the expected types for an RPC handler.
// Its methods panic when Schema is nil, because the types would be lost.
type BodyQueryParamType struct {
	Schema *HandlerSchema
}

// BodyType registers the concrete Go type (or example instance) that represents
// the HTTP request body for this RPC. It panics when body is not a struct or a
// pointer to a struct. Returns the receiver to allow method chaining.
func (b *BodyQueryParamType) BodyType(body any) *BodyQueryParamType {
	b.requireSchema("BodyType")
	b.Schema.SetBodyType(body)
	return b
}

// QueryType registers the concrete Go type (or example instance) that represents
// the URL query string parameters for this RPC. It panics when query is not a struct
// or a pointer to a struct. Returns the receiver to allow method chaining.
func (b *BodyQueryParamType) QueryType(query any) *BodyQueryParamType {
	b.requireSchema("QueryType")
	b.Schema.SetQueryType(query)
	return b
}

// Params sets extra URL path parameter names on the underlying schema.
// It is a no-op when slugs is empty. Returns the receiver to allow method chaining.
func (b *BodyQueryParamType) Params(slugs []string) *BodyQueryParamType {
	if len(slugs) == 0 {
		return b
	}

	b.requireSchema("Params")
	b.Schema.SetParamsType(slugs)
	return b
}

// requireSchema panics when the builder has no schema to update.
func (b *BodyQueryParamType) requireSchema(method string) {
	if b.Schema == nil {
		panic("chirpc: " + method + " was called on a BodyQueryParamType with no Schema.")
	}
}

// NewBodyQueryParamType creates a new BodyQueryParamType wrapping the provided HandlerSchema.
func NewBodyQueryParamType(schema *HandlerSchema) *BodyQueryParamType {
	return &BodyQueryParamType{
		Schema: schema,
	}
}
