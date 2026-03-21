// Example 01: Basic Prompt
//
// Demonstrates the simplest usage of go-exons:
// parse a prompt-type .exons file, execute it with data, and print the output.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	exons "github.com/itsatony/go-exons"
)

func main() {
	// Read the .exons file from disk
	data, err := os.ReadFile("prompt.exons")
	if err != nil {
		log.Fatalf("failed to read exons file: %v", err)
	}

	// Create an engine with default settings
	engine := exons.MustNew()

	// Parse the template (frontmatter + body)
	tmpl, err := engine.Parse(string(data))
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	// Print spec metadata
	spec := tmpl.Spec()
	if spec != nil {
		fmt.Printf("Spec Name: %s\n", spec.Name)
		fmt.Printf("Spec Type: %s\n", spec.Type)
		fmt.Printf("Spec Description: %s\n\n", spec.Description)
	}

	// Execute with custom data
	ctx := context.Background()
	output, err := tmpl.Execute(ctx, map[string]any{
		"user_name": "Alice",
		"topic":     "declarative agent specs",
	})
	if err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	fmt.Println("=== Output with user_name=Alice ===")
	fmt.Println(output)

	// Execute again with defaults (no data)
	output2, err := tmpl.Execute(ctx, nil)
	if err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	fmt.Println("=== Output with defaults ===")
	fmt.Println(output2)
}
