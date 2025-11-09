package rpcType

import (
	"fmt"
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v3"
)

// RpcSchema represents the schema for an RPC endpoint, including parameter, body, query, and response types.
type RpcSchema struct {
	Param    string
	Body     string
	Query    string
	Response string
}

// rpcType manages a collection of RPC schemas organized by HTTP method and URL.
type rpcType struct {
	shouldExport bool // indicates if the generated TypeScript type should be exported
	types        *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, RpcSchema]]
}

// AddRpcSchema adds a new RpcSchema for the given HTTP method and URL.
func (rt *rpcType) AddRpcSchema(method, url string, schema RpcSchema) {
	// lazily initialize map for method to preserve insertion order
	urls, ok := rt.types.Get(method)
	if !ok {
		urls = orderedmap.NewOrderedMap[string, RpcSchema]()
		rt.types.Set(method, urls)
	}

	urls.Set(url, schema)
}

// String returns a string representation of the RPC type as a TypeScript type definition.
func (rt *rpcType) String() string {
	result := []string{}
	if rt.shouldExport {
		result = append(result, "export")
	}

	result = append(result, "type ApiSchema = {")

	for methodEl := rt.types.Front(); methodEl != nil; methodEl = methodEl.Next() {
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

// convertURLPattern converts a URL pattern with curly braces to a colon-prefixed slug format.
func convertURLPattern(input string) string {
	var result []rune
	braces := 0
	var buffer []rune

	for _, r := range input {
		switch r {
		case '{':
			if braces == 0 {
				buffer = buffer[:0] // reset buffer for a new slug section
			}
			braces++
			if braces > 1 {
				buffer = append(buffer, r)
			}
		case '}':
			braces--
			if braces == 0 {
				// flush buffered content as :slug...
				result = append(result, ':')
				result = append(result, buffer...)
			} else if braces > 0 {
				buffer = append(buffer, r)
			}
		default:
			if braces > 0 {
				buffer = append(buffer, r)
			} else {
				result = append(result, r)
			}
		}
	}

	return string(result)
}

// NewRpcType creates and returns a new rpcType instance.
// If shouldExport is true, the generated TypeScript type will be exported.
func NewRpcType(shouldExport bool) *rpcType {
	return &rpcType{
		shouldExport: shouldExport,
		types:        orderedmap.NewOrderedMap[string, *orderedmap.OrderedMap[string, RpcSchema]](),
	}
}
