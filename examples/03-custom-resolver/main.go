// Example 03: Custom Resolver
//
// Demonstrates how to implement and register a custom resolver.
// A "timestamp" resolver is created that outputs the current time
// in configurable formats.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	exons "github.com/itsatony/go-exons"
)

// TimestampResolver is a custom resolver that outputs timestamps.
// It supports a "format" attribute: "date" for date-only, or
// defaults to full RFC3339 datetime.
type TimestampResolver struct{}

func (r *TimestampResolver) TagName() string {
	return "timestamp"
}

func (r *TimestampResolver) Resolve(_ context.Context, _ *exons.Context, attrs exons.Attributes) (string, error) {
	format, _ := attrs.Get("format")
	now := time.Now()

	switch format {
	case "date":
		return now.Format("2006-01-02"), nil
	case "time":
		return now.Format("15:04:05"), nil
	default:
		return now.Format(time.RFC3339), nil
	}
}

func (r *TimestampResolver) Validate(_ exons.Attributes) error {
	// All attribute combinations are valid for this resolver
	return nil
}

func main() {
	// Read the .exons file from disk
	data, err := os.ReadFile("template.exons")
	if err != nil {
		log.Fatalf("failed to read exons file: %v", err)
	}

	// Create an engine and register the custom resolver
	engine := exons.MustNew()
	err = engine.RegisterResolver(&TimestampResolver{})
	if err != nil {
		log.Fatalf("failed to register resolver: %v", err)
	}

	// Show registered resolvers
	fmt.Printf("Registered resolvers: %v\n\n", engine.ListResolvers())

	// Parse and execute
	tmpl, err := engine.Parse(string(data))
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	ctx := context.Background()
	output, err := tmpl.Execute(ctx, map[string]any{
		"user_name": "Charlie",
	})
	if err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	fmt.Println("=== Output ===")
	fmt.Println(output)
}
