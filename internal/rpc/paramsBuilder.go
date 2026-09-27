package rpc

// ParamsBuilder is returned by AddHandler. It can add extra path parameter names
// to the handler's schema. Request body and query types come from AddTypedHandler.
// Its methods panic when Schema is nil, because the change would be lost.
type ParamsBuilder struct {
	Schema *HandlerSchema
}

// Params sets extra URL path parameter names on the underlying schema.
// It is a no-op when slugs is empty. Returns the receiver to allow method chaining.
func (b *ParamsBuilder) Params(slugs []string) *ParamsBuilder {
	if len(slugs) == 0 {
		return b
	}

	b.requireSchema("Params")
	b.Schema.SetParamsType(slugs)
	return b
}

// requireSchema panics when the builder has no schema to update.
func (b *ParamsBuilder) requireSchema(method string) {
	if b.Schema == nil {
		panic("chirpc: " + method + " was called on a ParamsBuilder with no Schema.")
	}
}

// NewParamsBuilder creates a new ParamsBuilder wrapping the provided HandlerSchema.
func NewParamsBuilder(schema *HandlerSchema) *ParamsBuilder {
	return &ParamsBuilder{
		Schema: schema,
	}
}
