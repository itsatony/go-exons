package exons

import (
	"strings"
	"testing"
)

// The prompt subtype and the extended input-kind vocabulary are DECLARATION-ONLY
// and ADVISORY. These tests pin both halves of that contract: the data round-trips
// faithfully, and go-exons refuses to judge it. The second half matters more than
// the first — the moment Validate starts rejecting an unknown kind, every consumer
// pinning a newer vocabulary than the library breaks on documents that are fine.

func TestSubtypeRoundTripsThroughYAML(t *testing.T) {
	src := "name: bwa-analysis\ntype: prompt\nsubtype: template\n"

	spec, err := ParseYAMLSpec(src)
	if err != nil {
		t.Fatalf("ParseYAMLSpec: %v", err)
	}
	if spec.Subtype != SubtypePromptTemplate {
		t.Fatalf("Subtype = %q, want %q", spec.Subtype, SubtypePromptTemplate)
	}
	// It must land on the named field, never in the Extensions catch-all — a
	// subtype riding in Extensions would be stripped by any export that drops them.
	if _, inExtensions := spec.Extensions[SpecFieldSubtype]; inExtensions {
		t.Fatal("subtype leaked into Extensions; it must bind to the named Spec field")
	}

	out, err := spec.Serialize(DefaultSerializeOptions())
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if !strings.Contains(string(out), "subtype: template") {
		t.Fatalf("serialized output lost the subtype:\n%s", out)
	}
}

// A subtype refines a type. If an export strips `type:` the subtype must go with
// it, or the document declares a refinement of nothing.
func TestSubtypeIsNeverEmittedWithoutItsType(t *testing.T) {
	spec := &Spec{Name: "bwa", Type: DocumentTypePrompt, Subtype: SubtypePromptTemplate}

	out, err := spec.Serialize(AgentSkillsExportOptions())
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	got := string(out)
	if strings.Contains(got, SpecFieldSubtype) {
		t.Fatalf("agent-skills export stripped type: but kept subtype:\n%s", got)
	}

	// And the same coupling in the other direction: a subtype with no type at all
	// is not emitted even on the permissive path.
	orphan := &Spec{Name: "orphan", Subtype: SubtypePromptTemplate}
	out, err = orphan.Serialize(DefaultSerializeOptions())
	if err != nil {
		t.Fatalf("Serialize orphan: %v", err)
	}
	if strings.Contains(string(out), SpecFieldSubtype) {
		t.Fatalf("emitted a subtype with no type:\n%s", out)
	}
}

// Decision: go-exons declares, the executing system validates. An unrecognised
// subtype is legal here.
func TestSubtypeIsNotValidated(t *testing.T) {
	for _, subtype := range []string{SubtypePromptFragment, SubtypePromptTemplate, "something-nobody-has-defined-yet", ""} {
		spec := &Spec{Name: "p", Description: "d", Type: DocumentTypePrompt, Subtype: subtype}
		if err := spec.Validate(); err != nil {
			t.Fatalf("Validate rejected subtype %q: %v", subtype, err)
		}
	}
}

func TestIsAgentSkillsCompatibleAccountsForSubtype(t *testing.T) {
	// Agent Skills has never heard of `subtype`, so a spec carrying one is not
	// compatible — even though Type is empty and every other field is clean.
	spec := &Spec{Name: "p", Subtype: SubtypePromptFragment}
	if spec.IsAgentSkillsCompatible() {
		t.Fatal("a spec carrying a subtype reported itself Agent Skills compatible")
	}
	if !(&Spec{Name: "p"}).IsAgentSkillsCompatible() {
		t.Fatal("a bare spec should still be Agent Skills compatible")
	}
}

func TestExtendedInputKindsRoundTrip(t *testing.T) {
	src := `name: quarterly
type: prompt
subtype: template
inputs:
  months:
    type: multiselect
    label: Months
    options:
      - value: jan
        label: January
      - value: feb
  ranking:
    type: sort
    options:
      - value: revenue
      - value: cost
  owners:
    type: associate
    options:
      - value: region
    associate_with:
      - value: analyst
  report:
    type: file-upload
    accept: ["application/pdf", ".csv"]
    max_size_bytes: 10485760
    max_files: 3
  headcount:
    type: number
  audited:
    type: boolean
`
	spec, err := ParseYAMLSpec(src)
	if err != nil {
		t.Fatalf("ParseYAMLSpec: %v", err)
	}

	if got := spec.Inputs["ranking"].Options; len(got) != 2 || got[0].Value != "revenue" {
		// Order is the declared initial ordering for a sort input and is load-bearing.
		t.Fatalf("sort options lost their order: %+v", got)
	}

	assoc := spec.Inputs["owners"]
	if len(assoc.Options) != 1 || len(assoc.AssociateWith) != 1 {
		t.Fatalf("associate did not bind both sides: %+v", assoc)
	}
	if assoc.AssociateWith[0].Value != "analyst" {
		t.Fatalf("associate_with = %q, want analyst", assoc.AssociateWith[0].Value)
	}

	file := spec.Inputs["report"]
	// MaxSizeBytes bounds each file; MaxFiles bounds how many. Two limits, and a
	// consumer enforcing only one of them is the bug this asserts against.
	if file.MaxSizeBytes != 10485760 {
		t.Fatalf("MaxSizeBytes = %d", file.MaxSizeBytes)
	}
	if file.MaxFiles != 3 {
		t.Fatalf("MaxFiles = %d", file.MaxFiles)
	}
	if len(file.Accept) != 2 || file.Accept[0] != "application/pdf" {
		t.Fatalf("Accept = %v", file.Accept)
	}

	out, err := spec.Serialize(DefaultSerializeOptions())
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	for _, want := range []string{"associate_with", "max_size_bytes", "max_files", "accept"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("serialized output dropped %q:\n%s", want, out)
		}
	}
}

func TestInputKindsAreNotValidated(t *testing.T) {
	// Nonsense on every axis: an unknown kind, options on a kind that ignores them,
	// a select with no options, a negative file cap. All of it is the executing
	// system's problem, not ours.
	spec := &Spec{
		Name:        "p",
		Description: "d",
		Type:        DocumentTypePrompt,
		Inputs: map[string]*InputDef{
			"a": {Type: "not-a-real-kind"},
			"b": {Type: InputTypeText, Options: []InputOption{{Value: "x"}}},
			"c": {Type: InputTypeSelect},
			"d": {Type: InputTypeFileUpload, MaxFiles: -1},
		},
	}
	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate judged an input kind: %v", err)
	}
}

func TestCloneIsolatesEveryInputSlice(t *testing.T) {
	original := &Spec{
		Name:    "p",
		Type:    DocumentTypePrompt,
		Subtype: SubtypePromptTemplate,
		Inputs: map[string]*InputDef{
			"x": {
				Type:          InputTypeAssociate,
				Options:       []InputOption{{Value: "left"}},
				AssociateWith: []InputOption{{Value: "right"}},
				Accept:        []string{"application/pdf"},
			},
		},
	}

	clone := original.Clone()
	clone.Subtype = SubtypePromptFragment
	clone.Inputs["x"].Options[0].Value = "mutated"
	clone.Inputs["x"].AssociateWith[0].Value = "mutated"
	clone.Inputs["x"].Accept[0] = "mutated"

	src := original.Inputs["x"]
	if original.Subtype != SubtypePromptTemplate {
		t.Fatal("Clone aliased Subtype")
	}
	if src.Options[0].Value != "left" {
		t.Fatal("Clone aliased Options")
	}
	if src.AssociateWith[0].Value != "right" {
		t.Fatal("Clone aliased AssociateWith")
	}
	if src.Accept[0] != "application/pdf" {
		t.Fatal("Clone aliased Accept")
	}
}

func TestInputTypeToMIMEDescribesTheStructuredKinds(t *testing.T) {
	cases := map[string]string{
		InputTypeText:        A2AMIMETextPlain,
		InputTypeNumber:      A2AMIMETextPlain,
		InputTypeBoolean:     A2AMIMETextPlain,
		InputTypeSelect:      A2AMIMETextPlain,
		InputTypeMultiselect: A2AMIMEApplicationJSON,
		InputTypeSort:        A2AMIMEApplicationJSON,
		InputTypeAssociate:   A2AMIMEApplicationJSON,
		InputTypeFileUpload:  A2AMIMEApplicationOctetStream,
		// The vocabulary is open, so an unknown kind degrades rather than failing.
		"invented-tomorrow": A2AMIMETextPlain,
	}
	for kind, want := range cases {
		if got := inputTypeToMIME(kind); got != want {
			t.Errorf("inputTypeToMIME(%q) = %q, want %q", kind, got, want)
		}
	}
}
