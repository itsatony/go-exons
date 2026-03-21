// Example 05: Template Composition
//
// Demonstrates using MapSpecResolver to compose templates via {~exons.ref~} tags.
// A main agent references a skill by slug, and the resolver provides
// the skill's template body at execution time.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	exons "github.com/itsatony/go-exons"
)

func main() {
	// Read the main agent's .exons file
	data, err := os.ReadFile("main-agent.exons")
	if err != nil {
		log.Fatalf("failed to read exons file: %v", err)
	}

	// Create an engine
	engine := exons.MustNew()

	// Create a MapSpecResolver and add the greeting skill.
	// In a real application, skills might be loaded from a database or file system.
	resolver := exons.NewMapSpecResolver()

	// Create the skill spec
	greetingSpec := &exons.Spec{
		Name:        "greeting-skill",
		Description: "A skill that generates personalized greetings",
		Type:        "skill",
	}

	// The skill's template body — this is what gets injected at the {~exons.ref~} site
	greetingBody := "Greeting Skill: Generate a warm, personalized greeting. " +
		"Include the user's name if provided and mention the current time of day."

	resolver.Add("greeting-skill", greetingSpec, greetingBody)

	// Wire the resolver into the engine
	engine.SetSpecResolver(resolver)

	fmt.Printf("Skills registered in resolver: %d\n", resolver.Count())
	fmt.Printf("Has 'greeting-skill': %v\n\n", resolver.Has("greeting-skill"))

	// Parse the main agent template
	tmpl, err := engine.Parse(string(data))
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	// Print spec info
	spec := tmpl.Spec()
	if spec != nil {
		fmt.Printf("Agent: %s\n", spec.Name)
		fmt.Printf("Skills referenced: %d\n", len(spec.Skills))
		for _, s := range spec.Skills {
			fmt.Printf("  - %s\n", s.Slug)
		}
		fmt.Println()
	}

	// Execute via engine.Execute so the SpecResolver is injected into the context.
	// This allows {~exons.ref slug="greeting-skill" /~} to be resolved automatically.
	ctx := context.Background()
	source := string(data)
	output, err := engine.Execute(ctx, source, map[string]any{
		"user_request": "Can you say hello to me?",
	})
	if err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	// Extract structured messages from the raw output
	messages := exons.ExtractMessagesFromOutput(output)

	// Print the messages showing the resolved ref
	fmt.Println("=== Extracted Messages ===")
	for i, msg := range messages {
		fmt.Printf("\nMessage %d [%s]:\n%s\n", i+1, msg.Role, msg.Content)
	}
}
