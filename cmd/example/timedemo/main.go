package main

import (
	"fmt"
	"time"

	"github.com/coder/guts"
	"github.com/coder/guts/config"
)

type User struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type Post struct {
	ID          int         `json:"id"`
	Title       string      `json:"title"`
	PublishedAt time.Time   `json:"published_at"`
	Timestamps  []time.Time `json:"timestamps"`
}

func main() {
	parser, err := guts.NewGolangParser()
	if err != nil {
		panic(err)
	}
	parser.IncludeCustomDeclaration(config.StandardMappings())
	if err := parser.IncludeGenerate("github.com/iambpn/chirpc/cmd/example/timedemo"); err != nil {
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

	// Print generated TypeScript interfaces
	fmt.Println("Generated TypeScript interfaces:")
	fmt.Println(output)
}
