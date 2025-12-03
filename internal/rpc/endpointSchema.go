package rpc

import (
	"fmt"
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v3"
)

// EndpointSchema manages a collection of RPC schemas organized by HTTP method and URL.
type EndpointSchema struct {
	shouldExport bool // indicates if the generated TypeScript type should be exported
	types        *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, RpcSchema]]
}

// AddRpcSchema adds a new RpcSchema for the given HTTP method and URL.
func (a *EndpointSchema) AddRpcSchema(method, url string, schema RpcSchema) {
	// lazily initialize map for method to preserve insertion order
	urls, ok := a.types.Get(method)
	if !ok {
		urls = orderedmap.NewOrderedMap[string, RpcSchema]()
		a.types.Set(method, urls)
	}

	urls.Set(url, schema)
}

// String returns a string representation of the RPC type as a TypeScript type definition.
func (a *EndpointSchema) String() string {
	result := []string{}
	if a.shouldExport {
		result = append(result, "export")
	}

	result = append(result, "type ApiSchema = {")

	for methodEl := a.types.Front(); methodEl != nil; methodEl = methodEl.Next() {
		method := methodEl.Key
		urls := methodEl.Value
		result = append(result, fmt.Sprintf(`"%s": {`, strings.ToUpper(method)))

		for urlEl := urls.Front(); urlEl != nil; urlEl = urlEl.Next() {
			url := convertURLPattern(urlEl.Key)
			schema := urlEl.Value
			result = append(result, fmt.Sprintf(`"%s": {`, url))
			if schema.Param != "" {
				result = append(result, fmt.Sprintf("params: %s;", schema.Param))
			}
			if schema.Query != "" {
				result = append(result, fmt.Sprintf("query?: %s;", schema.Query))
			}
			if schema.Body != "" {
				result = append(result, fmt.Sprintf("body: %s;", schema.Body))
			}
			if schema.Response != "" {
				result = append(result, fmt.Sprintf("response: %s;", schema.Response))
			} else {
				result = append(result, "response: void;")
			}
			result = append(result, "};")
		}
		result = append(result, "};")
	}
	result = append(result, "};")
	return strings.Join(result, " ")
}

// NewEndpointSchema creates and returns a new EndpointSchema instance.
// If shouldExport is true, the generated TypeScript type will be exported.
func NewEndpointSchema(shouldExport bool) *EndpointSchema {
	return &EndpointSchema{
		shouldExport: shouldExport,
		types:        orderedmap.NewOrderedMap[string, *orderedmap.OrderedMap[string, RpcSchema]](),
	}
}
