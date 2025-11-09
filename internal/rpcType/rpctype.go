package rpcType

import (
	"fmt"
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v3"
)

type RpcSchema struct {
	Param    string
	Body     string
	Query    string
	Response string
}

// Collection of RPC Schemas
type rpcType struct {
	shouldExport bool
	types        *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, RpcSchema]]
}

func (rt *rpcType) AddRpcSchema(method, url string, schema RpcSchema) {
	// lazily initialize map for method to preserve insertion order
	urls, ok := rt.types.Get(method)
	if !ok {
		urls = orderedmap.NewOrderedMap[string, RpcSchema]()
		rt.types.Set(method, urls)
	}

	urls.Set(url, schema)
}

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

func NewRpcType(shouldExport bool) *rpcType {
	return &rpcType{
		shouldExport: shouldExport,
		types:        orderedmap.NewOrderedMap[string, *orderedmap.OrderedMap[string, RpcSchema]](),
	}
}
