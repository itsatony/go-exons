// Example 09: Typed Inputs
//
// The frontmatter `inputs:` block, end to end. Three releases built this format
// (v0.19.0 subtype, v0.21.0 the exons.input verb and the reserved `input`
// context root, v0.24.0 inheritance) and until now there was no runnable
// example of any of it — only inline snippets, which are not compiled and not
// tested.
//
// What this demonstrates, in the order main() does it:
//
//  1. DECLARED INPUTS ARE READABLE UNDER `input`. {~exons.input name="tone" /~}
//     IS the path input.tone, so exons.if and exons.for reach declared inputs
//     with no special grammar: in="input.sources" simply works.
//
//  2. DEFAULTS APPLY. An input the caller does not bind takes its declared
//     default. Binding nothing at all still renders a complete document.
//
//  3. INHERITANCE MERGES DECLARATIONS (v0.24.0). report.exons extends
//     base-report, and the parent's `audience` is in scope for the child with
//     no restatement. Where both declare the same name — `tone` — the CHILD
//     wins: the composing document is the authority, the composed document
//     supplies the fallback.
//
//  4. DECLARED-BUT-UNBOUND IS NOT THE SAME AS UNDECLARED. `sources` is
//     declared and unbound, so it is PRESENT AS NIL and its exons.for iterates
//     zero times, silently. `input.timeline` is declared nowhere, so the path
//     genuinely does not exist and the loop FAILS — which is why it carries
//     onerror="default". Before v0.24.0 that attribute was parsed and then
//     dropped on the floor, so a block tag could not be given an escape hatch
//     at all.
//
//  5. VALIDATION BELONGS BEFORE THE RENDER. `summary` is required.
//     ValidateInputBinding reports every violation at once, so a form can mark
//     all its bad fields in one pass — render time is far too late to ask a
//     user for a missing value.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	exons "github.com/itsatony/go-exons"
)

func main() {
	engine := exons.MustNew()
	ctx := context.Background()

	// The parent must be REGISTERED under the name the child extends. Inheritance
	// is resolved by name through the engine, not by file path.
	parentSrc, err := os.ReadFile("base-report.exons")
	if err != nil {
		log.Fatalf("failed to read base-report.exons: %v", err)
	}
	if err := engine.RegisterTemplate("base-report", string(parentSrc)); err != nil {
		log.Fatalf("failed to register the parent template: %v", err)
	}

	childSrc, err := os.ReadFile("report.exons")
	if err != nil {
		log.Fatalf("failed to read report.exons: %v", err)
	}
	tmpl, err := engine.Parse(string(childSrc))
	if err != nil {
		log.Fatalf("failed to parse report.exons: %v", err)
	}

	// ---------------------------------------------------------------------
	// The declared contract, merged across the extends chain.
	// ---------------------------------------------------------------------
	// DeclaredInputs is the MERGED contract. Template.Spec() is the parse result and reports
	// this document's own frontmatter alone — with it, an extending document silently omits
	// every field its parent declares.
	// The error is not decorative: it reports that the extends chain could NOT be walked to its
	// end (a cycle, a missing parent, an over-deep chain), which means the contract below is
	// PARTIAL. Displaying a partial contract is fine; publishing one as the document's whole
	// contract is not — and a caller with no error to check has no way to tell the difference.
	declared, err := tmpl.DeclaredInputs()
	if err != nil {
		log.Fatalf("the declared contract is incomplete: %v", err)
	}
	keys, err := tmpl.DeclaredInputKeys()
	if err != nil {
		log.Fatalf("the declared contract is incomplete: %v", err)
	}
	section("The contract this document declares")
	for _, name := range keys {
		def := declared[name]
		origin := "own"
		if _, isOwn := tmpl.Spec().Inputs[name]; !isOwn {
			origin = "inherited"
		}
		fmt.Printf("  %-10s %-12s %-10s default=%-10v required=%v\n", name, def.Type, origin, def.Default, def.Required)
	}

	// ---------------------------------------------------------------------
	// Validation runs BEFORE the render.
	// ---------------------------------------------------------------------
	spec := tmpl.Spec()
	section("Validating an empty binding")
	for _, verr := range spec.ValidateInputBinding(map[string]any{}) {
		fmt.Printf("  refused: %v\n", verr)
	}

	values := map[string]any{
		"summary":  "checkout latency tripled for 40 minutes",
		"severity": 2,
		"sources":  []string{"logs", "pager"},
	}
	section("Validating a real binding")
	if errs := spec.ValidateInputBinding(values); len(errs) == 0 {
		fmt.Println("  accepted")
	} else {
		for _, verr := range errs {
			fmt.Printf("  refused: %v\n", verr)
		}
	}

	// ---------------------------------------------------------------------
	// Render. BindInputs applies the declared defaults and hands back the map
	// that belongs under the reserved `input` key, so a caller never hand-rolls
	// the namespace or re-implements the default rules.
	// ---------------------------------------------------------------------
	section("Rendered with values bound")
	output, err := tmpl.Execute(ctx, map[string]any{
		exons.ContextKeyInput: spec.BindInputs(values),
	})
	if err != nil {
		log.Fatalf("render failed: %v", err)
	}
	fmt.Println(indent(output))

	// Nothing bound at all. `summary` is required, so this document is
	// incomplete by its own declaration — but it still RENDERS, because
	// requiredness is a validation contract and not a render-time trap.
	// `sources` is unbound, so its loop produces nothing and says nothing.
	section("Rendered with nothing bound (defaults only)")
	output, err = tmpl.Execute(ctx, nil)
	if err != nil {
		log.Fatalf("render failed: %v", err)
	}
	fmt.Println(indent(output))

	// ---------------------------------------------------------------------
	// DryRun sees the whole document, parent body included.
	// ---------------------------------------------------------------------
	section("DryRun over the resolved document")
	dr := tmpl.DryRun(ctx, nil)
	for _, ref := range dr.Inputs {
		fmt.Printf("  input %-10s declared=%v (line %d)\n", ref.Name, ref.Declared, ref.Line)
	}
	fmt.Printf("  analysis complete: %v\n", dr.AnalysisComplete())
}

func section(title string) {
	fmt.Printf("\n=== %s ===\n", title)
}

// indent shifts a rendered document right so it reads as a block rather than as
// more program output.
func indent(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, line := range lines {
		lines[i] = "  | " + line
	}
	return strings.Join(lines, "\n")
}
