// Example 07: A2A Agent Card
//
// Demonstrates generating a Google A2A protocol Agent Card from spec metadata.
// The Agent Card describes the agent's capabilities for discovery and
// orchestration on A2A networks.
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

	// Parse the spec
	spec, err := exons.Parse(data)
	if err != nil {
		log.Fatalf("failed to parse spec: %v", err)
	}

	fmt.Printf("Agent: %s\n", spec.Name)
	fmt.Printf("Description: %s\n", spec.Description)
	fmt.Printf("Type: %s\n\n", spec.Type)

	// Print spec metadata that will enrich the Agent Card
	if spec.Registry != nil {
		fmt.Printf("Registry version: %s\n", spec.Registry.Version)
	}
	if spec.Dispatch != nil {
		fmt.Printf("Trigger keywords: %v\n", spec.Dispatch.TriggerKeywords)
		fmt.Printf("Trigger description: %s\n", spec.Dispatch.TriggerDescription)
	}
	if spec.Safety != nil {
		fmt.Printf("Guardrails: %s\n", spec.Safety.Guardrails)
		fmt.Printf("Deny tools: %v\n", spec.Safety.DenyTools)
	}
	fmt.Println()

	// Compile the A2A Agent Card
	ctx := context.Background()
	card, err := spec.CompileAgentCard(ctx, &exons.A2ACardOptions{
		URL:                  "https://agents.example.com/research",
		ProviderOrganization: "Acme Research Corp",
		ProviderURL:          "https://acme-research.example.com",
		// Version is auto-derived from registry.version (2.1.0)
	})
	if err != nil {
		log.Fatalf("failed to compile agent card: %v", err)
	}

	// Print card details
	fmt.Println("=== A2A Agent Card ===")
	fmt.Printf("Name: %s\n", card.Name)
	fmt.Printf("URL: %s\n", card.URL)
	fmt.Printf("Version: %s\n", card.Version)
	fmt.Printf("Protocol Version: %s\n", card.ProtocolVersion)

	if card.Provider != nil {
		fmt.Printf("Provider: %s (%s)\n", card.Provider.Organization, card.Provider.URL)
	}

	if card.Capabilities != nil {
		fmt.Printf("Streaming: %v\n", card.Capabilities.Streaming)
	}

	fmt.Printf("Input Modes: %v\n", card.DefaultInputModes)
	fmt.Printf("Output Modes: %v\n", card.DefaultOutputModes)

	if len(card.Skills) > 0 {
		fmt.Printf("\nSkills (%d):\n", len(card.Skills))
		for _, skill := range card.Skills {
			fmt.Printf("  - %s (id: %s, tags: %v)\n", skill.Name, skill.ID, skill.Tags)
		}
	}

	if len(card.Metadata) > 0 {
		fmt.Printf("\nMetadata:\n")
		for k, v := range card.Metadata {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	// Serialize to JSON
	jsonBytes, err := card.ToJSONPretty()
	if err != nil {
		log.Fatalf("failed to serialize agent card: %v", err)
	}

	fmt.Println("\n=== JSON Output ===")
	fmt.Println(string(jsonBytes))
}
