// Package typescript converts the routes collected by the rpc package into
// TypeScript declarations and an ApiSchema type. It is kept apart from the rpc
// package so that servers which do not generate TypeScript do not link the
// TypeScript compiler.
package typescript

import (
	"fmt"
	"strings"

	"github.com/iambpn/chirpc/internal/rpc"
	"github.com/iambpn/chirpc/internal/tags"
)

// Convert generates the TypeScript declarations and the ApiSchema type for
// every handler in schemas and its mounted routers.
func Convert(schemas *rpc.RouterRpcSchemas) (string, error) {
	routes, err := schemas.Routes()
	if err != nil {
		return "", err
	}

	typeOverrides, err := schemas.TypeOverrides()
	if err != nil {
		return "", err
	}

	eps := NewEndpointSchema(true)
	converter, err := newGutsConverter(typeOverrides)
	if err != nil {
		return "", err
	}

	for _, route := range routes {
		response, err := converter.responseType(route.Response)
		if err != nil {
			return "", err
		}

		schema := RpcSchema{Response: response}
		if route.Body != nil {
			schema.Body, err = converter.inlineStruct(route.Body)
			if err != nil {
				return "", err
			}
		}
		if route.Query != nil {
			schema.Query, err = converter.inlineStruct(route.Query)
			if err != nil {
				return "", err
			}
			schema.QueryRequired = tags.HasRequiredField(route.Query)
		}
		if len(route.Params) > 0 {
			schema.Param = sliceToTsInf(route.Params)
		}

		eps.AddRpcSchema(route.Method, route.URL, schema)
	}

	declarations, err := converter.declarationStrings()
	if err != nil {
		return "", err
	}

	parts := make([]string, 0, len(declarations)+1)
	for _, declaration := range declarations {
		parts = append(parts, strings.TrimSpace(declaration))
	}
	parts = append(parts, eps.String())
	return strings.Join(parts, "\n\n"), nil
}

// sliceToTsInf builds a TypeScript interface string mapping each slug to string,
// or returns 'never' if the slice is empty.
func sliceToTsInf(slice []string) string {
	if len(slice) == 0 {
		return "never"
	}

	var inf strings.Builder
	for _, s := range slice {
		fmt.Fprintf(&inf, `"%s": string;`, s)
	}

	return fmt.Sprintf("{ %s }", inf.String())
}
