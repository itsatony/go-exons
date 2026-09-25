package exons

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// skillEnvDoc is go-exons#4's acceptance document: a skill declaring
// requirements.environment.code_execution: required.
const skillEnvDoc = `---
name: xlsx-builder
description: builds spreadsheets from tabular answers
type: skill
requirements:
  environment:
    code_execution: required
    packages:
      - python:openpyxl
      - python:pandas
    network: none
---
Build the workbook.
`

func TestEnvironmentValidate(t *testing.T) {
	tests := []struct {
		name    string
		env     *EnvironmentRequirement
		wantMsg string // "" = valid
	}{
		{"nil is valid", nil, ""},
		{"empty is valid", &EnvironmentRequirement{}, ""},
		{"code required", &EnvironmentRequirement{CodeExecution: EnvironmentCodeExecutionRequired}, ""},
		{"code optional", &EnvironmentRequirement{CodeExecution: EnvironmentCodeExecutionOptional}, ""},
		{"network required", &EnvironmentRequirement{Network: EnvironmentNetworkRequired}, ""},
		{"network optional", &EnvironmentRequirement{Network: EnvironmentNetworkOptional}, ""},
		{"network none", &EnvironmentRequirement{Network: EnvironmentNetworkNone}, ""},
		{"every ecosystem", &EnvironmentRequirement{CodeExecution: "required", Packages: []string{
			"python:openpyxl", "node:@scope/pkg", "r:ggplot2", "ruby:nokogiri", "rust:serde",
			"go:golang.org/x/text", "java:org.apache.poi/poi", "system:libreoffice",
		}}, ""},
		{"code none is not a value", &EnvironmentRequirement{CodeExecution: "none"}, ErrMsgEnvironmentCodeExecution},
		{"code case-folded is not a value", &EnvironmentRequirement{CodeExecution: "Required"}, ErrMsgEnvironmentCodeExecution},
		{"network out of vocabulary", &EnvironmentRequirement{Network: "offline"}, ErrMsgEnvironmentNetwork},
		{"unknown ecosystem", &EnvironmentRequirement{CodeExecution: "required", Packages: []string{"pypi:openpyxl"}}, ErrMsgEnvironmentPackageForm},
		{"uppercase ecosystem", &EnvironmentRequirement{CodeExecution: "required", Packages: []string{"Python:openpyxl"}}, ErrMsgEnvironmentPackageForm},
		{"bare name", &EnvironmentRequirement{CodeExecution: "required", Packages: []string{"openpyxl"}}, ErrMsgEnvironmentPackageForm},
		{"empty name", &EnvironmentRequirement{CodeExecution: "required", Packages: []string{"python:"}}, ErrMsgEnvironmentPackageForm},
		{"version specifier", &EnvironmentRequirement{CodeExecution: "required", Packages: []string{"python:openpyxl>=3.1"}}, ErrMsgEnvironmentPackageForm},
		{"whitespace", &EnvironmentRequirement{CodeExecution: "required", Packages: []string{"python:open pyxl"}}, ErrMsgEnvironmentPackageForm},
		{"url", &EnvironmentRequirement{CodeExecution: "required", Packages: []string{"python:https://evil.example/x"}}, ErrMsgEnvironmentPackageForm},
		{"duplicate package", &EnvironmentRequirement{CodeExecution: "required", Packages: []string{"python:a", "python:a"}}, ErrMsgEnvironmentPackageDup},
		{"packages without code_execution", &EnvironmentRequirement{Packages: []string{"python:openpyxl"}}, ErrMsgEnvironmentPackagesNeedExecution},
		{"packages with optional code_execution", &EnvironmentRequirement{CodeExecution: "optional", Packages: []string{"python:openpyxl"}}, ""},
		{"package at the length bound", &EnvironmentRequirement{CodeExecution: "required", Packages: []string{"python:" + strings.Repeat("a", MaxEnvironmentPackageLen-len("python:"))}}, ""},
		{"package over the length bound", &EnvironmentRequirement{CodeExecution: "required", Packages: []string{"python:" + strings.Repeat("a", MaxEnvironmentPackageLen-len("python:")+1)}}, ErrMsgEnvironmentPackageLong},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.env.Validate()
			if tc.wantMsg == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}

func environmentPackages(n int) []string {
	pkgs := make([]string, n)
	for i := range pkgs {
		pkgs[i] = fmt.Sprintf("python:package-number-%03d", i)
	}
	return pkgs
}

func TestEnvironmentPackageCountBound(t *testing.T) {
	assert.NoError(t, (&EnvironmentRequirement{CodeExecution: "required", Packages: environmentPackages(MaxEnvironmentPackages)}).Validate())
	err := (&EnvironmentRequirement{CodeExecution: "required", Packages: environmentPackages(MaxEnvironmentPackages + 1)}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgEnvironmentTooMany)
}

// TestEnvironmentValidatedThroughParse confirms go-exons#4's third box: the block is
// parsed and validated for type: skill by the same Parse → Spec.Validate path as
// agents, and a skill with no type key (Parse's default) is covered too.
func TestEnvironmentValidatedThroughParse(t *testing.T) {
	spec, err := Parse([]byte(skillEnvDoc))
	require.NoError(t, err)
	require.NotNil(t, spec.Requirements)
	assert.Equal(t, &EnvironmentRequirement{
		CodeExecution: EnvironmentCodeExecutionRequired,
		Packages:      []string{"python:openpyxl", "python:pandas"},
		Network:       EnvironmentNetworkNone,
	}, spec.Requirements.Environment)
	assert.True(t, spec.Requirements.Environment.RequiresCodeExecution())
	assert.False(t, spec.Requirements.Environment.RequiresNetwork())

	for name, mutate := range map[string]func(string) string{
		"skill":        func(d string) string { return d },
		"agent":        func(d string) string { return strings.Replace(d, "type: skill", "type: agent", 1) },
		"type omitted": func(d string) string { return strings.Replace(d, "type: skill\n", "", 1) },
	} {
		t.Run(name+" refuses an invalid block", func(t *testing.T) {
			bad := strings.Replace(mutate(skillEnvDoc), "network: none", "network: offline", 1)
			_, err := Parse([]byte(bad))
			require.Error(t, err)
			assert.Contains(t, err.Error(), ErrMsgEnvironmentNetwork)
		})
	}

	t.Run("prompt refuses the block", func(t *testing.T) {
		_, err := Parse([]byte(strings.Replace(skillEnvDoc, "type: skill", "type: prompt", 1)))
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgPromptNoEnvironment)
		_, err = Parse([]byte("---\nname: p\ndescription: d\ntype: prompt\nrequirements:\n  environment: {}\n---\nbody\n"))
		require.Error(t, err, "an empty environment block is still a block")
		assert.Contains(t, err.Error(), ErrMsgPromptNoEnvironment)
	})

	t.Run("prompt keeps its other requirements", func(t *testing.T) {
		_, err := Parse([]byte("---\nname: p\ndescription: d\ntype: prompt\nrequirements:\n  credentials:\n    - ref: r\n---\nbody\n"))
		assert.NoError(t, err, "only environment is prompt-prohibited")
	})

	// The library's policy for requirements: the parser ignores an unknown nested
	// key (the published schema refuses it — schema/agreement_test.go).
	t.Run("unknown key is ignored by the parser", func(t *testing.T) {
		doc := strings.Replace(skillEnvDoc, "    network: none\n", "    network: none\n    image: python:3.12-slim\n", 1)
		spec, err := Parse([]byte(doc))
		require.NoError(t, err)
		assert.Equal(t, EnvironmentNetworkNone, spec.Requirements.Environment.Network)
	})
}

// TestEnvironmentRoundTrip is go-exons#4's acceptance: parse → serialize → parse
// yields the same block, over every full-export path (never yaml.Marshal(spec),
// where the struct tag would hide a buildSerializeMap that emitted nothing).
func TestEnvironmentRoundTrip(t *testing.T) {
	spec, err := Parse([]byte(skillEnvDoc))
	require.NoError(t, err)
	want := spec.Requirements.Environment

	exports := map[string]func() ([]byte, error){
		"ExportFull":     spec.ExportFull,
		"Serialize(nil)": func() ([]byte, error) { return spec.Serialize(nil) },
		"Serialize(FullExportWithCredentials)": func() ([]byte, error) {
			return spec.Serialize(FullExportWithCredentials())
		},
	}
	for name, export := range exports {
		t.Run(name, func(t *testing.T) {
			out, err := export()
			require.NoError(t, err)
			reparsed, err := Parse(out)
			require.NoError(t, err)
			require.NotNil(t, reparsed.Requirements)
			assert.Equal(t, want, reparsed.Requirements.Environment)
			assert.Equal(t, spec.Body, reparsed.Body)
		})
	}
	t.Run("ExportDirectory", func(t *testing.T) {
		archive, err := ExportDirectory(spec, nil)
		require.NoError(t, err)
		result, err := ImportDirectory(archive)
		require.NoError(t, err)
		require.NotNil(t, result.Spec.Requirements)
		assert.Equal(t, want, result.Spec.Requirements.Environment)
	})
	// An empty block declares nothing, and yaml.v3's omitempty drops a pointer to a
	// zero struct, so it re-emits as ABSENT — the same declaration. Pinned so a
	// change in that behaviour is seen rather than discovered.
	t.Run("empty block re-emits as absent", func(t *testing.T) {
		s, err := Parse([]byte("---\nname: s\ndescription: d\nrequirements:\n  environment: {}\n---\nb\n"))
		require.NoError(t, err)
		out, err := s.ExportFull()
		require.NoError(t, err)
		re, err := Parse(out)
		require.NoError(t, err)
		require.NotNil(t, s.Requirements.Environment, "Parse keeps the empty block")
		assert.True(t, re.Requirements == nil || re.Requirements.Environment.IsZero())
	})
	t.Run("yaml tags", func(t *testing.T) {
		out, err := yaml.Marshal(want)
		require.NoError(t, err)
		assert.Contains(t, string(out), "code_execution: required")
	})
}

func TestEnvironmentClone(t *testing.T) {
	orig := &SpecRequirements{Environment: &EnvironmentRequirement{CodeExecution: "required", Packages: []string{"python:a"}}}
	clone := orig.Clone()
	require.NotNil(t, clone.Environment)
	clone.Environment.Packages[0] = "python:b"
	clone.Environment.Network = "none"
	assert.Equal(t, "python:a", orig.Environment.Packages[0], "the packages slice must not be shared")
	assert.Empty(t, orig.Environment.Network)
	assert.Nil(t, (*EnvironmentRequirement)(nil).Clone())

	spec := MustParse([]byte(skillEnvDoc))
	sc := spec.Clone()
	sc.Requirements.Environment.Packages[0] = "python:x"
	assert.Equal(t, "python:openpyxl", spec.Requirements.Environment.Packages[0], "Spec.Clone must deep-copy the block")
}

func TestEnvironmentCompatibilitySentence(t *testing.T) {
	tests := []struct {
		name string
		env  *EnvironmentRequirement
		want string
	}{
		{"nil", nil, ""},
		{"empty", &EnvironmentRequirement{}, ""},
		{"code required", &EnvironmentRequirement{CodeExecution: "required"}, "Requires code execution."},
		{"code optional", &EnvironmentRequirement{CodeExecution: "optional"}, "Uses code execution when available."},
		{"network only", &EnvironmentRequirement{Network: "required"}, "Requires network access."},
		{"network optional", &EnvironmentRequirement{Network: "optional"}, "Uses network access when available."},
		{"everything", &EnvironmentRequirement{CodeExecution: "required", Packages: []string{"python:openpyxl", "python:pandas"}, Network: "none"},
			"Requires code execution with packages python:openpyxl, python:pandas. Needs no network access."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.env.CompatibilitySentence())
		})
	}
}

// TestEnvironmentCompatibilitySentenceFitsAndNeverCutsAName drives the maximum
// package list through the 500-character agentskills.io bound: the sentence fits,
// every package it names is a WHOLE declared entry, and the count it leaves out is
// stated — so the sentence neither invents a package nor shortens the list silently.
func TestEnvironmentCompatibilitySentenceFitsAndNeverCutsAName(t *testing.T) {
	pkgs := make([]string, MaxEnvironmentPackages)
	for i := range pkgs {
		pkgs[i] = fmt.Sprintf("python:%s-%02d", strings.Repeat("p", 100), i)
	}
	env := &EnvironmentRequirement{CodeExecution: "required", Packages: pkgs, Network: "required"}
	require.NoError(t, env.Validate())
	got := env.CompatibilitySentence()
	assert.LessOrEqual(t, utf8.RuneCountInString(got), MaxCompatibilityLength)
	assert.True(t, strings.HasSuffix(got, " Requires network access."), got)

	listed := strings.TrimPrefix(strings.SplitN(got, ".", 2)[0], "Requires code execution with packages ")
	idx := strings.LastIndex(listed, " and ")
	require.Positive(t, idx, "an overflowing list must say how many it left out: %q", got)
	names := strings.Split(listed[:idx], ", ")
	for i, n := range names {
		assert.Equal(t, pkgs[i], n, "every listed name is a whole declared entry, in order")
	}
	assert.Equal(t, fmt.Sprintf(" and %d more", len(pkgs)-len(names)), listed[idx:])

	t.Run("no entry fits: the count alone", func(t *testing.T) {
		assert.Equal(t, " with 3 packages", compatPackagesClause([]string{"python:aaaa", "python:b", "python:c"}, 10))
	})
}

func TestAgentSkillsCompatibility(t *testing.T) {
	envReq := &SpecRequirements{Environment: &EnvironmentRequirement{CodeExecution: "required"}}
	const sentence = "Requires code execution."
	withAuthor := func(v any) map[string]any { return map[string]any{AgentSkillsFieldCompatibility: v} }

	tests := []struct {
		name string
		spec *Spec
		want string
	}{
		{"nil spec", nil, ""},
		{"neither", &Spec{}, ""},
		{"environment only", &Spec{Requirements: envReq}, sentence},
		{"author only", &Spec{Extensions: withAuthor("Needs Excel files.")}, "Needs Excel files."},
		{"author then sentence", &Spec{Requirements: envReq, Extensions: withAuthor("Needs Excel files.")}, "Needs Excel files. " + sentence},
		{"author trailing space trimmed before the join", &Spec{Requirements: envReq, Extensions: withAuthor("Needs Excel files.\n")}, "Needs Excel files. " + sentence},
		{"author already says it", &Spec{Requirements: envReq, Extensions: withAuthor("Office. " + sentence)}, "Office. " + sentence},
		{"author non-string is absent", &Spec{Requirements: envReq, Extensions: withAuthor(42)}, sentence},
		{"author over the bound is absent", &Spec{Requirements: envReq, Extensions: withAuthor(strings.Repeat("x", MaxCompatibilityLength+1))}, sentence},
		{"author plus sentence over the bound: author alone", &Spec{Requirements: envReq, Extensions: withAuthor(strings.Repeat("x", MaxCompatibilityLength-5))}, strings.Repeat("x", MaxCompatibilityLength-5)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.spec.AgentSkillsCompatibility()
			assert.Equal(t, tc.want, got)
			assert.LessOrEqual(t, utf8.RuneCountInString(got), MaxCompatibilityLength)
		})
	}
}

// TestAgentSkillsExportCarriesTheEnvironment: the Agent-Skills card has no
// requirements block, so `compatibility` is the ONLY way a portable consumer learns
// the skill needs code execution. The full export keeps the author's field VERBATIM
// (the composed text is written nowhere), and export → import → export is stable.
func TestAgentSkillsExportCarriesTheEnvironment(t *testing.T) {
	doc := strings.Replace(skillEnvDoc, "type: skill\n", "type: skill\ncompatibility: Works on xlsx files.\n", 1)
	spec, err := Parse([]byte(doc))
	require.NoError(t, err)
	want := "Works on xlsx files. Requires code execution with packages python:openpyxl, python:pandas. Needs no network access."

	card, err := spec.ExportToSkillMD()
	require.NoError(t, err)
	var fm map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(strings.Split(strings.TrimPrefix(string(card), "---\n"), "\n---\n")[0]), &fm))
	assert.Equal(t, want, fm[AgentSkillsFieldCompatibility])
	assert.Equal(t, "xlsx-builder", fm[SpecFieldName], "positive anchor: the card is not empty")
	assert.NotContains(t, fm, SpecFieldRequirements, "the card's vocabulary stays closed")

	full, err := spec.ExportFull()
	require.NoError(t, err)
	reparsed, err := Parse(full)
	require.NoError(t, err)
	assert.Equal(t, "Works on xlsx files.", reparsed.Extensions[AgentSkillsFieldCompatibility], "a full export keeps the author's field verbatim")

	imported, err := ImportFromSkillMD(string(card))
	require.NoError(t, err)
	imported.Requirements = spec.Requirements.Clone() // an author re-adds the block
	again, err := imported.ExportToSkillMD()
	require.NoError(t, err)
	var fm2 map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(strings.Split(strings.TrimPrefix(string(again), "---\n"), "\n---\n")[0]), &fm2))
	assert.Equal(t, want, fm2[AgentSkillsFieldCompatibility], "re-export does not append the sentence twice")

	t.Run("no environment, no author: no key", func(t *testing.T) {
		plain := &Spec{Name: "plain", Description: "d", Body: "b"}
		out, err := plain.ExportAgentSkill()
		require.NoError(t, err)
		assert.NotContains(t, string(out), AgentSkillsFieldCompatibility)
	})
}
