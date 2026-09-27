package typescript

import (
	"fmt"
	"strings"

	orderedmap "github.com/elliotchance/orderedmap/v3"
	"github.com/iambpn/chirpc/internal/rpc"
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
// Each method, URL, and member is written on its own line.
func (a *EndpointSchema) String() string {
	var b strings.Builder
	if a.shouldExport {
		b.WriteString("export ")
	}
	b.WriteString("type ApiSchema = {\n")

	for methodEl := a.types.Front(); methodEl != nil; methodEl = methodEl.Next() {
		fmt.Fprintf(&b, "  %q: {\n", strings.ToUpper(methodEl.Key))

		for urlEl := methodEl.Value.Front(); urlEl != nil; urlEl = urlEl.Next() {
			schema := urlEl.Value
			fmt.Fprintf(&b, "    %q: {\n", rpc.ColonPattern(urlEl.Key))

			queryKey := "query?"
			if schema.QueryRequired {
				queryKey = "query"
			}
			response := schema.Response
			if response == "" {
				response = "void"
			}

			writeMember(&b, "params", schema.Param)
			writeMember(&b, queryKey, schema.Query)
			writeMember(&b, "body", schema.Body)
			writeMember(&b, "response", response)
			b.WriteString("    };\n")
		}
		b.WriteString("  };\n")
	}

	b.WriteString("};\n")
	return b.String()
}

// writeMember writes one ApiSchema member line. It skips empty values and indents
// multi-line values so they line up with the member.
func writeMember(b *strings.Builder, key, value string) {
	if value == "" {
		return
	}
	value = strings.ReplaceAll(strings.TrimSpace(value), "\n", "\n      ")
	fmt.Fprintf(b, "      %s: %s;\n", key, value)
}

// NewEndpointSchema creates and returns a new EndpointSchema instance.
// If shouldExport is true, the generated TypeScript type will be exported.
func NewEndpointSchema(shouldExport bool) *EndpointSchema {
	return &EndpointSchema{
		shouldExport: shouldExport,
		types:        orderedmap.NewOrderedMap[string, *orderedmap.OrderedMap[string, RpcSchema]](),
	}
}
