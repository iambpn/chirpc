package main

import (
	"fmt"
	"time"

	"github.com/iambpn/chirpc/internal/tsGen"
	"github.com/iambpn/chirpc/internal/tsGen/tsopts"
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
	gen := tsGen.New(tsopts.SetToLowercaseExportedField(false))

	// Add User type
	if err := gen.AddValue(User{}); err != nil {
		panic(err)
	}

	// Add Post type
	if err := gen.AddValue(Post{}); err != nil {
		panic(err)
	}

	// Print generated TypeScript interfaces
	fmt.Println("Generated TypeScript interfaces:")
	fmt.Println(gen.String())
}
