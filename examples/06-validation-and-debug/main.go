// Example 06: Validation and Debug
//
// Demonstrates validation, dry-run analysis, and explain features.
// These tools help inspect templates without executing against an LLM.
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

	source := string(data)
	engine := exons.MustNew()

	// --- Step 1: Validate the template ---
	fmt.Println("=== Validation ===")
	validationResult, err := engine.Validate(source)
	if err != nil {
		log.Fatalf("validation failed: %v", err)
	}

	fmt.Printf("Valid: %v\n", validationResult.IsValid())
	fmt.Printf("Errors: %d\n", len(validationResult.Errors()))
	fmt.Printf("Warnings: %d\n", len(validationResult.Warnings()))

	for _, issue := range validationResult.Issues() {
		fmt.Printf("  [%s] %s (tag: %s, line: %d)\n",
			issue.Severity, issue.Message, issue.TagName, issue.Position.Line)
	}
	fmt.Println()

	// --- Step 2: Parse the template ---
	tmpl, err := engine.Parse(source)
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}

	// --- Step 3: Dry Run (static analysis without execution) ---
	fmt.Println("=== Dry Run ===")
	ctx := context.Background()
	dryRunData := map[string]any{
		"user_name": "Alice",
		// Intentionally omitting "show_details", "topic", and "items"
		// to demonstrate missing variable detection
	}

	dryResult := tmpl.DryRun(ctx, dryRunData)
	fmt.Printf("Valid: %v\n", dryResult.Valid)
	fmt.Printf("Variables found: %d\n", len(dryResult.Variables))

	for _, v := range dryResult.Variables {
		status := "found"
		if !v.InData {
			status = "MISSING"
			if v.HasDefault {
				status = fmt.Sprintf("missing (default: %q)", v.Default)
			}
		}
		fmt.Printf("  - %s [line %d]: %s\n", v.Name, v.Line, status)
	}

	fmt.Printf("Conditionals: %d\n", len(dryResult.Conditionals))
	for _, c := range dryResult.Conditionals {
		fmt.Printf("  - %s [line %d] (hasElse: %v)\n", c.Condition, c.Line, c.HasElse)
	}

	fmt.Printf("Loops: %d\n", len(dryResult.Loops))
	for _, l := range dryResult.Loops {
		fmt.Printf("  - for %s in %s [line %d] (inData: %v)\n",
			l.ItemVar, l.Source, l.Line, l.InData)
	}

	if len(dryResult.Warnings) > 0 {
		fmt.Printf("Warnings: %d\n", len(dryResult.Warnings))
		for _, w := range dryResult.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}

	fmt.Printf("\nPlaceholder Output:\n%s\n", dryResult.Output)

	// --- Step 4: Explain (full execution with detailed trace) ---
	fmt.Println("=== Explain ===")
	explainData := map[string]any{
		"user_name":    "Alice",
		"show_details": true,
		"topic":        "go-exons templates",
		"items":        []any{"parse", "validate", "execute"},
	}

	explainResult := tmpl.Explain(ctx, explainData)
	fmt.Printf("AST:\n%s\n", explainResult.AST)
	fmt.Printf("Execution time: %v\n", explainResult.Timing.Total)
	fmt.Printf("Variable accesses: %d\n", len(explainResult.Variables))

	for _, va := range explainResult.Variables {
		fmt.Printf("  - %s [line %d]: %v\n", va.Path, va.Line, va.Value)
	}

	if explainResult.Error != nil {
		fmt.Printf("Error: %v\n", explainResult.Error)
	}

	fmt.Printf("\nFinal Output:\n%s\n", explainResult.Output)
}
