// Example 02: Chat Agent
//
// Demonstrates parsing an agent-type .exons file with message tags,
// executing it, and extracting structured messages for use with chat APIs.
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
	data, err := os.ReadFile("agent.exons")
	if err != nil {
		log.Fatalf("failed to read exons file: %v", err)
	}

	// Create an engine
	engine := exons.MustNew()

	// Parse the template
	tmpl, err := engine.Parse(string(data))
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	// Print spec metadata including execution config
	spec := tmpl.Spec()
	if spec != nil {
		fmt.Printf("Agent: %s\n", spec.Name)
		fmt.Printf("Description: %s\n", spec.Description)
		if spec.Execution != nil {
			fmt.Printf("Provider: %s\n", spec.Execution.Provider)
			fmt.Printf("Model: %s\n", spec.Execution.Model)
			if spec.Execution.Temperature != nil {
				fmt.Printf("Temperature: %.1f\n", *spec.Execution.Temperature)
			}
		}
		fmt.Println()
	}

	// Execute and extract structured messages
	ctx := context.Background()
	messages, err := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{
		"user_name": "Bob",
		"question":  "Can you explain what go-exons does?",
	})
	if err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	// Print extracted messages — these are ready to send to an LLM API
	fmt.Println("=== Extracted Messages ===")
	for i, msg := range messages {
		fmt.Printf("\nMessage %d:\n", i+1)
		fmt.Printf("  Role:    %s\n", msg.Role)
		fmt.Printf("  Content: %s\n", msg.Content)
		fmt.Printf("  Cache:   %v\n", msg.Cache)
	}
}
