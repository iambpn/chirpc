// Command gen-schema writes the TypeScript ApiSchema of the example task API.
//
// It builds the same router as the server, so the schema always matches the served
// routes. It is a separate program because tsgen links the TypeScript compiler.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/iambpn/chirpc/cmd/example/api"
	"github.com/iambpn/chirpc/v1/tsgen"
)

//go:generate go run . -out ../client/src/apiSchema.ts

func main() {
	out := flag.String("out", "apiSchema.ts", "The file to write the TypeScript schema to.")
	flag.Parse()

	router := api.NewRouter(api.NewStore())
	if err := tsgen.GenerateRPCSchema(router, *out); err != nil {
		log.Fatalf("Could not generate the API schema: %v", err)
	}
	fmt.Printf("Wrote the API schema to %s.\n", *out)
}
