// Example 08: Syntax Safety
//
// Demonstrates writing exons syntax AS CONTENT (v0.15.0):
//   - markdown fence mode (WithMarkdownFences): fenced code blocks are inert,
//     a fence whose info string starts with "exons" renders live
//   - verbatim tilde fences {~~ ... ~~} with length escalation
//   - raw blocks with byte-exact bodies
//   - Validate() lints for tag-like syntax inside inert fences
//   - Spec.ContentFormat set by the SKILL.md import path
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	exons "github.com/itsatony/go-exons"
)

func main() {
	data, err := os.ReadFile("skill.exons")
	if err != nil {
		log.Fatalf("failed to read exons file: %v", err)
	}

	// Markdown fence mode: fenced code blocks in the body are inert, so the
	// teaching examples inside them are NOT executed. Only the ```exons
	// fence and the prose tags render.
	engine := exons.MustNew(exons.WithMarkdownFences())

	tmpl, err := engine.Parse(string(data))
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	ctx := context.Background()
	output, err := tmpl.Execute(ctx, map[string]any{
		"topic":    "syntax safety",
		"greeting": "Servus",
	})
	if err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	fmt.Println("=== Rendered body (fences inert, ```exons live) ===")
	fmt.Println(output)

	// Validate() warns when an inert fence contains tag-like syntax — the
	// nudge for authors who meant to write a live ```exons fence.
	result, err := engine.Validate(string(data))
	if err != nil {
		log.Fatalf("failed to validate template: %v", err)
	}
	fmt.Println("=== Validate() warnings ===")
	for _, w := range result.Warnings() {
		fmt.Printf("line %d: %s\n", w.Position.Line, w.Message)
	}

	// Raw blocks are byte-exact since v0.15.0: bodies may contain broken
	// syntax and round-trip verbatim (first {~/exons.raw~} closes).
	rawDemo := `{~exons.raw~}a lone {~ and \{~ survive here{~/exons.raw~}`
	rawOut, err := exons.MustNew().Execute(ctx, rawDemo, nil)
	if err != nil {
		log.Fatalf("failed to execute raw demo: %v", err)
	}
	fmt.Println("=== Raw block byte-fidelity ===")
	fmt.Println(rawOut)

	// The SKILL.md import path marks specs as markdown so downstream
	// consumers know to enable WithMarkdownFences.
	spec, err := exons.ImportFromSkillMD(string(data))
	if err != nil {
		log.Fatalf("failed to import as SKILL.md: %v", err)
	}
	fmt.Printf("=== ImportFromSkillMD ===\nContentFormat: %s\n", spec.ContentFormat)
}
