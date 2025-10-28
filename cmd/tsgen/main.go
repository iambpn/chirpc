package main

import (
	"fmt"

	"github.com/iambpn/chirpc/internal/tsGen"
)

type Nested struct {
	NestedField string
}

type Config struct {
	ToLower    bool `tsKey:"to_lower"`
	AddHeader  bool `tsOptional:"true"`
	GenericAny tsGen.GenericAny
	Anyy       any
	Nested     Nested
	AnonNested struct {
		AnonField int
		Nested    Nested
	}
}

type Config2 struct {
	FieldA  string
	FieldB  int
	Config1 Config
}

func main() {
	tsGen := tsGen.New()

	c1 := Config{
		ToLower:    true,
		AddHeader:  true,
		GenericAny: 123,
		Anyy:       10,
		Nested: Nested{
			NestedField: "example",
		},
		AnonNested: struct {
			AnonField int
			Nested    Nested
		}{
			AnonField: 42,
			Nested: Nested{
				NestedField: "anon example",
			},
		},
	}

	err := tsGen.AddValue(c1)

	if err != nil {
		panic(err)
	}

	c2 := Config2{
		FieldA:  "test",
		FieldB:  100,
		Config1: c1,
	}

	tsGen.AddValue(c2)

	fmt.Println(tsGen.String())
}
