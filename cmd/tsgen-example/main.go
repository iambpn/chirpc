package main

import (
	"fmt"

	"github.com/coder/guts"
	"github.com/coder/guts/config"
)

type Nested struct {
	NestedField string
}

type Config struct {
	ToLower    bool `json:"to_lower"`
	AddHeader  bool `json:"add_header,omitempty"`
	GenericAny any
	Anyy       any `json:"json_any_field"` // make sure it works with json tags also
	Nested     Nested
	AnonNested AnonNested
}

type AnonNested struct {
	AnonField int
	Nested    Nested
}

type Config2 struct {
	FieldA  string
	FieldB  int
	Config1 Config
}

func main() {
	parser, err := guts.NewGolangParser()
	if err != nil {
		panic(err)
	}
	parser.IncludeCustomDeclaration(config.StandardMappings())
	if err := parser.IncludeGenerate("github.com/iambpn/chirpc/cmd/tsgen-example"); err != nil {
		panic(err)
	}

	typescript, err := parser.ToTypescript()
	if err != nil {
		panic(err)
	}
	typescript.ApplyMutations(config.ExportTypes)

	output, err := typescript.Serialize()
	if err != nil {
		panic(err)
	}
	fmt.Println(output)
}
