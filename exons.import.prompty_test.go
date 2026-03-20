package exons

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Tag conversion ---

func TestConvertPromptyTags_OpeningTags(t *testing.T) {
	input := `{~prompty.var name="x" /~}`
	result := convertPromptyTags(input)
	assert.Equal(t, `{~exons.var name="x" /~}`, result)
}

func TestConvertPromptyTags_ClosingTags(t *testing.T) {
	input := `{~prompty.message role="system"~}Hello{~/prompty.message~}`
	result := convertPromptyTags(input)
	assert.Equal(t, `{~exons.message role="system"~}Hello{~/exons.message~}`, result)
}

func TestConvertPromptyTags_MultipleTags(t *testing.T) {
	input := `{~prompty.if eval="x"~}{~prompty.var name="y" /~}{~/prompty.if~}`
	result := convertPromptyTags(input)
	assert.Equal(t, `{~exons.if eval="x"~}{~exons.var name="y" /~}{~/exons.if~}`, result)
}

func TestConvertPromptyTags_NoTags(t *testing.T) {
	input := "Hello, world!"
	result := convertPromptyTags(input)
	assert.Equal(t, "Hello, world!", result)
}

func TestConvertPromptyTags_ExonsTagsUnchanged(t *testing.T) {
	input := `{~exons.var name="x" /~}`
	result := convertPromptyTags(input)
	assert.Equal(t, `{~exons.var name="x" /~}`, result)
}

// --- isPromptyContent ---

func TestIsPromptyContent_True(t *testing.T) {
	assert.True(t, isPromptyContent([]byte(`Hello {~prompty.var name="x" /~}`)))
}

func TestIsPromptyContent_False(t *testing.T) {
	assert.False(t, isPromptyContent([]byte(`Hello {~exons.var name="x" /~}`)))
}

func TestIsPromptyContent_Empty(t *testing.T) {
	assert.False(t, isPromptyContent([]byte("")))
}

// --- splitPromptyFrontmatter ---

func TestSplitPromptyFrontmatter_Valid(t *testing.T) {
	content := "---\nname: test\n---\nBody here"
	fm, body, has := splitPromptyFrontmatter(content)
	assert.True(t, has)
	assert.Equal(t, "name: test", fm)
	assert.Equal(t, "Body here", body)
}

func TestSplitPromptyFrontmatter_NoFrontmatter(t *testing.T) {
	content := "Just a body"
	fm, body, has := splitPromptyFrontmatter(content)
	assert.False(t, has)
	assert.Equal(t, "", fm)
	assert.Equal(t, "Just a body", body)
}

func TestSplitPromptyFrontmatter_NoClosingDelim(t *testing.T) {
	content := "---\nname: test\nNo closing"
	_, _, has := splitPromptyFrontmatter(content)
	assert.False(t, has)
}

func TestSplitPromptyFrontmatter_BOM(t *testing.T) {
	content := "\xef\xbb\xbf---\nname: bom-test\n---\nBody"
	fm, body, has := splitPromptyFrontmatter(content)
	assert.True(t, has)
	assert.Equal(t, "name: bom-test", fm)
	assert.Equal(t, "Body", body)
}

// --- remapPromptyFields ---

func TestRemapPromptyFields_DelegationToDispatch(t *testing.T) {
	input := map[string]any{
		"name":       "test",
		"delegation": map[string]any{"trigger_keywords": []any{"dns"}},
	}
	result := remapPromptyFields(input)
	assert.NotNil(t, result[SpecFieldDispatch])
	assert.Nil(t, result["delegation"])
}

func TestRemapPromptyFields_TestsToVerifications(t *testing.T) {
	input := map[string]any{
		"name": "test",
		"tests": []any{
			map[string]any{"name": "t1", "user_message": "hello"},
		},
	}
	result := remapPromptyFields(input)
	verifs, ok := result[SpecFieldVerifications].([]any)
	require.True(t, ok)
	require.Len(t, verifs, 1)
	v, ok := verifs[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "hello", v["prompt"])
	assert.Nil(t, v["user_message"])
}

func TestRemapPromptyFields_PluginToRegistry(t *testing.T) {
	input := map[string]any{
		"name":   "test",
		"plugin": map[string]any{"namespace": "dns", "trust_level": "verified", "version": "1.0"},
	}
	result := remapPromptyFields(input)
	reg, ok := result[SpecFieldRegistry].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "dns", reg["namespace"])
	assert.Equal(t, "verified", reg["origin"])
	assert.Nil(t, reg["trust_level"])
}

func TestRemapPromptyFields_ExtraFieldsPassthrough(t *testing.T) {
	// Extra prompty-only fields stay as top-level keys.
	// They get captured by Spec.Extensions (yaml:",inline") during Parse().
	input := map[string]any{
		"name":    "test",
		"license": "MIT",
	}
	result := remapPromptyFields(input)
	assert.Equal(t, "MIT", result["license"])
	assert.Equal(t, "test", result["name"])
}

// --- flattenGenSpecWrapper ---

func TestFlattenGenSpecWrapper_Promotes(t *testing.T) {
	input := map[string]any{
		"name": "test",
		"genspec": map[string]any{
			"version": "1",
			"memory":  map[string]any{"scope": "test-scope"},
			"safety":  map[string]any{"guardrails": "enabled"},
		},
	}
	result := flattenGenSpecWrapper(input)
	assert.Nil(t, result["genspec"])
	mem, ok := result["memory"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test-scope", mem["scope"])
	safety, ok := result["safety"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "enabled", safety["guardrails"])
}

func TestFlattenGenSpecWrapper_NoGenSpec(t *testing.T) {
	input := map[string]any{"name": "test"}
	result := flattenGenSpecWrapper(input)
	assert.Equal(t, input, result)
}

func TestFlattenGenSpecWrapper_RemapsNestedFields(t *testing.T) {
	input := map[string]any{
		"name": "test",
		"genspec": map[string]any{
			"delegation": map[string]any{"trigger_keywords": []any{"dns"}},
			"tests":      []any{map[string]any{"name": "t1", "user_message": "hi"}},
			"plugin":     map[string]any{"trust_level": "internal"},
		},
	}
	result := flattenGenSpecWrapper(input)
	assert.Nil(t, result["genspec"])
	assert.NotNil(t, result[SpecFieldDispatch])
	verifs, ok := result[SpecFieldVerifications].([]any)
	require.True(t, ok)
	require.Len(t, verifs, 1)
	v, ok := verifs[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "hi", v["prompt"])
	reg, ok := result[SpecFieldRegistry].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "internal", reg["origin"])
}

// --- ImportPrompty: full round-trip ---

func TestImportPrompty_Empty(t *testing.T) {
	_, err := ImportPrompty([]byte{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgImportPromptyFailed)
}

func TestImportPrompty_MinimalPrompty(t *testing.T) {
	data := []byte(`---
name: test-agent
description: A test agent
type: agent
---
Hello {~prompty.var name="x" /~}
`)
	spec, err := ImportPrompty(data)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Equal(t, "test-agent", spec.Name)
	assert.Equal(t, DocumentTypeAgent, spec.Type)
	assert.Contains(t, spec.Body, `{~exons.var name="x" /~}`)
	assert.NotContains(t, spec.Body, `{~prompty.`)
}

func TestImportPrompty_FieldRemapping(t *testing.T) {
	data := []byte(`---
name: remap-test
description: Test field remapping
type: agent
delegation:
  trigger_keywords: [dns, domain]
  trigger_description: DNS tasks
tests:
  - name: test-one
    user_message: "Hello"
    expect:
      output_contains: "world"
plugin:
  namespace: test-ns
  trust_level: internal
  version: "1.0"
---
Body
`)
	spec, err := ImportPrompty(data)
	require.NoError(t, err)
	require.NotNil(t, spec)

	// delegation → dispatch
	require.NotNil(t, spec.Dispatch)
	assert.Equal(t, []string{"dns", "domain"}, spec.Dispatch.TriggerKeywords)
	assert.Equal(t, "DNS tasks", spec.Dispatch.TriggerDescription)

	// tests → verifications (user_message → prompt)
	require.Len(t, spec.Verifications, 1)
	assert.Equal(t, "test-one", spec.Verifications[0].Name)
	assert.Equal(t, "Hello", spec.Verifications[0].Prompt)

	// plugin → registry (trust_level → origin)
	require.NotNil(t, spec.Registry)
	assert.Equal(t, "test-ns", spec.Registry.Namespace)
	assert.Equal(t, OriginInternal, spec.Registry.Origin)
	assert.Equal(t, "1.0", spec.Registry.Version)
}

func TestImportPrompty_GenSpecFlattening(t *testing.T) {
	data := []byte(`---
name: genspec-test
description: Test genspec flattening
type: agent
genspec:
  version: "1"
  memory:
    scope: test-scope
    auto_recall: true
  dispatch:
    trigger_keywords: [search]
  safety:
    guardrails: enabled
---
Body
`)
	spec, err := ImportPrompty(data)
	require.NoError(t, err)
	require.NotNil(t, spec)

	// memory promoted from genspec
	require.NotNil(t, spec.Memory)
	assert.Equal(t, "test-scope", spec.Memory.Scope)

	// dispatch promoted from genspec
	require.NotNil(t, spec.Dispatch)
	assert.Equal(t, []string{"search"}, spec.Dispatch.TriggerKeywords)

	// safety promoted from genspec
	require.NotNil(t, spec.Safety)
	assert.Equal(t, GuardrailsEnabled, spec.Safety.Guardrails)
}

func TestImportPrompty_ExtraFieldsToExtensions(t *testing.T) {
	data := []byte(`---
name: extra-test
description: Test extra fields
type: skill
license: MIT
compatibility: ">=1.0"
---
Body
`)
	spec, err := ImportPrompty(data)
	require.NoError(t, err)
	require.NotNil(t, spec)

	// Extra fields should be in extensions
	assert.Equal(t, "MIT", spec.Extensions["license"])
	assert.Equal(t, ">=1.0", spec.Extensions["compatibility"])
}

func TestImportPrompty_TagConversionInBody(t *testing.T) {
	data := []byte(`---
name: tag-test
description: Test tags
type: agent
---
{~prompty.message role="system"~}
You are helpful.
{~/prompty.message~}
{~prompty.message role="user"~}
{~prompty.var name="query" /~}
{~/prompty.message~}
`)
	spec, err := ImportPrompty(data)
	require.NoError(t, err)
	require.NotNil(t, spec)

	assert.Contains(t, spec.Body, `{~exons.message role="system"~}`)
	assert.Contains(t, spec.Body, `{~/exons.message~}`)
	assert.Contains(t, spec.Body, `{~exons.var name="query" /~}`)
	assert.NotContains(t, spec.Body, `{~prompty.`)
	assert.NotContains(t, spec.Body, `{~/prompty.`)
}

func TestImportPrompty_MalformedYAML(t *testing.T) {
	data := []byte("---\n: : invalid yaml\n---\nBody")
	_, err := ImportPrompty(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgImportPromptyParseFailed)
}

func TestImportPrompty_NoFrontmatter(t *testing.T) {
	// Body-only content without frontmatter — Parse() wraps in empty frontmatter
	// which still succeeds (Parse returns a default skill spec with body).
	data := []byte("Hello world")
	spec, err := ImportPrompty(data)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Contains(t, spec.Body, "Hello world")
}

func TestImportPrompty_NoFrontmatterWithTags(t *testing.T) {
	// Body-only with prompty tags — tags are converted, body is preserved.
	data := []byte(`{~prompty.var name="x" /~}`)
	spec, err := ImportPrompty(data)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Contains(t, spec.Body, `{~exons.var name="x" /~}`)
}

// --- Import() routing ---

func TestImport_PromptyExtension(t *testing.T) {
	data := []byte(`---
name: route-test
description: Routing test
type: skill
---
{~prompty.var name="x" /~}
`)
	result, err := Import(data, "agent.prompty")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Spec)
	assert.Equal(t, "route-test", result.Spec.Name)
	assert.Contains(t, result.Spec.Body, `{~exons.var`)
}

func TestImport_GenSpecExtension(t *testing.T) {
	data := []byte(`---
name: genspec-route
description: GenSpec routing test
type: agent
memory:
  scope: test
---
Body
`)
	result, err := Import(data, "agent.genspec")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Spec)
	assert.Equal(t, "genspec-route", result.Spec.Name)
}

func TestImport_MDWithPromptyContent(t *testing.T) {
	data := []byte(`---
name: md-prompty
description: MD with prompty content
type: skill
---
{~prompty.var name="x" /~}
`)
	result, err := Import(data, "agent.md")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Spec)
	assert.Contains(t, result.Spec.Body, `{~exons.var`)
}

func TestImport_MDWithoutPromptyContent(t *testing.T) {
	data := []byte(`---
name: normal-md
description: Normal md file
type: skill
---
Just plain text
`)
	result, err := Import(data, "agent.md")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Spec)
	assert.Equal(t, "normal-md", result.Spec.Name)
}
