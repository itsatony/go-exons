// Example 04: Tool Export
//
// Demonstrates parsing an agent with tool definitions and exporting
// them to various LLM provider formats (OpenAI, Anthropic, Gemini, MCP, Cohere).
package main

import (
	"encoding/json"
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

	// Parse the spec to access tool definitions
	spec, err := exons.Parse(data)
	if err != nil {
		log.Fatalf("failed to parse spec: %v", err)
	}

	fmt.Printf("Agent: %s\n", spec.Name)
	fmt.Printf("Tools defined: %d\n\n", len(spec.Tools.Functions))

	// Print each function definition
	for _, fn := range spec.Tools.Functions {
		fmt.Printf("  Function: %s — %s\n", fn.Name, fn.Description)
	}
	fmt.Println()

	// Export to all provider formats using the batch methods on ToolsConfig
	providerFormats := map[string][]map[string]any{
		"OpenAI":    spec.Tools.ToOpenAITools(),
		"Anthropic": spec.Tools.ToAnthropicTools(),
		"Gemini":    spec.Tools.ToGeminiTools(),
		"MCP":       spec.Tools.ToMCPTools(),
		"Cohere":    spec.Tools.ToCohereTools(),
	}

	// Also demonstrate individual function export
	for _, fn := range spec.Tools.Functions {
		fmt.Printf("=== %s: Individual Exports ===\n", fn.Name)
		printJSON("  OpenAI", fn.ToOpenAITool())
		printJSON("  Anthropic", fn.ToAnthropicTool())
		printJSON("  Gemini", fn.ToGeminiTool())
		printJSON("  MCP", fn.ToMCPTool())
		printJSON("  Cohere", fn.ToCohereTool())
		fmt.Println()
	}

	// Print batch exports
	for provider, tools := range providerFormats {
		fmt.Printf("=== %s Batch Export (%d tools) ===\n", provider, len(tools))
		j, _ := json.MarshalIndent(tools, "", "  ")
		fmt.Println(string(j))
		fmt.Println()
	}
}

// printJSON prints a labeled JSON object.
func printJSON(label string, v any) {
	j, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		fmt.Printf("%s: error: %v\n", label, err)
		return
	}
	fmt.Printf("%s: %s\n", label, string(j))
}
