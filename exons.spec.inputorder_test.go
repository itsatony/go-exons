package exons

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// orderedDoc declares its inputs in an order that is NOT alphabetical and NOT the
// reverse of alphabetical, so a test passing against it cannot be passing by
// accident against either sort direction.
const orderedDoc = `---
name: intake
description: An interview whose question order is the point
type: prompt
subtype: template
inputs:
  full_name:
    type: text
    required: true
  address:
    type: text
  zip_code:
    type: text
  criteria:
    type: sort
    options:
      - value: cost
      - value: speed
---
Body`

func TestInputOrder_DerivedFromTheAuthoredMapping(t *testing.T) {
	spec, err := Parse([]byte(orderedDoc))
	require.NoError(t, err)

	want := []string{"full_name", "address", "zip_code", "criteria"}
	assert.Equal(t, want, spec.InputOrder, "the authored key order must survive the unmarshal")
	assert.Equal(t, want, spec.OrderedInputKeys())

	// The revert check: alphabetical is what this replaces, so the assertion is only
	// worth anything if the two differ.
	assert.NotEqual(t, []string{"address", "criteria", "full_name", "zip_code"}, want)
}

func TestInputOrder_ParseYAMLSpecTakesTheSamePath(t *testing.T) {
	// aigentverse projects field specs from ParseYAMLSpec, not Parse. Both go through
	// yaml.Unmarshal into a Spec, which is exactly why the capture lives in
	// UnmarshalYAML rather than in Parse.
	fm := `name: intake
description: d
inputs:
  zeta:
    type: text
  alpha:
    type: text`
	spec, err := ParseYAMLSpec(fm)
	require.NoError(t, err)
	assert.Equal(t, []string{"zeta", "alpha"}, spec.OrderedInputKeys())
}

func TestInputOrder_DocumentWithoutInputs(t *testing.T) {
	spec, err := ParseYAMLSpec("name: bare\ndescription: d")
	require.NoError(t, err)
	assert.Nil(t, spec.InputOrder)
	assert.Nil(t, spec.OrderedInputKeys())
}

func TestInputOrder_ExtensionsStillCatchUnknownKeys(t *testing.T) {
	// UnmarshalYAML decodes through a type alias precisely so `yaml:",inline"` keeps
	// working. If the alias trick were wrong, this is what would break first.
	spec, err := ParseYAMLSpec("name: x\ndescription: d\nsome_unknown_key: 7\ninputs:\n  a:\n    type: text")
	require.NoError(t, err)
	assert.Equal(t, 7, spec.Extensions["some_unknown_key"])
	assert.Equal(t, []string{"a"}, spec.InputOrder)
}

func TestInputOrder_DeclaredOrderWinsOverDerived(t *testing.T) {
	spec, err := ParseYAMLSpec(`name: x
description: d
input_order: [b, a]
inputs:
  a:
    type: text
  b:
    type: text`)
	require.NoError(t, err)
	assert.Equal(t, []string{"b", "a"}, spec.OrderedInputKeys())
}

func TestInputOrder_SurvivesAJSONRoundTrip(t *testing.T) {
	// The escape hatch that matters: a consumer storing the projection as JSON and
	// rehydrating it has no key order to recover, so the order must be ON the wire.
	spec, err := Parse([]byte(orderedDoc))
	require.NoError(t, err)

	raw, err := json.Marshal(spec)
	require.NoError(t, err)

	var back Spec
	require.NoError(t, json.Unmarshal(raw, &back))
	assert.Equal(t, spec.OrderedInputKeys(), back.OrderedInputKeys())
}

func TestInputOrder_SurvivesAYAMLRoundTrip(t *testing.T) {
	// yaml.Marshal writes a Go map with its keys sorted, so the round trip depends
	// entirely on input_order being emitted and re-read.
	spec, err := Parse([]byte(orderedDoc))
	require.NoError(t, err)

	raw, err := yaml.Marshal(spec)
	require.NoError(t, err)

	var back Spec
	require.NoError(t, yaml.Unmarshal(raw, &back))
	assert.Equal(t, spec.OrderedInputKeys(), back.OrderedInputKeys())
}

func TestOrderedInputKeys_IsTotal(t *testing.T) {
	t.Run("an input missing from a declared order sorts after the named ones", func(t *testing.T) {
		s := &Spec{
			Inputs: map[string]*InputDef{
				"a": {Type: InputTypeText},
				"b": {Type: InputTypeText},
				"c": {Type: InputTypeText},
			},
			InputOrder: []string{"c"},
		}
		assert.Equal(t, []string{"c", "a", "b"}, s.OrderedInputKeys())
	})

	t.Run("a spec built in Go with no order at all is still stable", func(t *testing.T) {
		s := &Spec{Inputs: map[string]*InputDef{"b": {}, "a": {}, "c": {}}}
		for i := 0; i < 20; i++ {
			assert.Equal(t, []string{"a", "b", "c"}, s.OrderedInputKeys())
		}
	})

	t.Run("an order naming a removed input does not emit it", func(t *testing.T) {
		s := &Spec{
			Inputs:     map[string]*InputDef{"a": {}},
			InputOrder: []string{"gone", "a"},
		}
		assert.Equal(t, []string{"a"}, s.OrderedInputKeys())
	})

	t.Run("nil spec, and a spec with no inputs", func(t *testing.T) {
		var nilSpec *Spec
		assert.Nil(t, nilSpec.OrderedInputKeys())
		assert.Nil(t, (&Spec{}).OrderedInputKeys())
	})
}

func TestInputOrder_ValidateRejectsAnAuthorsContradiction(t *testing.T) {
	t.Run("naming an input that is not declared", func(t *testing.T) {
		_, err := Parse([]byte(`---
name: x
description: d
input_order: [a, typo]
inputs:
  a:
    type: text
---
Body`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not declared")
	})

	t.Run("naming the same input twice", func(t *testing.T) {
		_, err := Parse([]byte(`---
name: x
description: d
input_order: [a, a]
inputs:
  a:
    type: text
---
Body`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "twice")
	})

	t.Run("a DERIVED order can never trip either check", func(t *testing.T) {
		_, err := Parse([]byte(orderedDoc))
		require.NoError(t, err)
	})
}

// ⚠ EVERY TEST BELOW EXISTS BECAUSE REVIEW FOUND THE FEATURE INERT, OR WORSE, ON A
// PATH THE ORIGINAL TESTS DID NOT TAKE. They are grouped so the lesson stays legible:
// a test that exercises a path consumers do not use passes while the thing is broken.

func TestInputOrder_AMergeKeyInsideInputsStillParses(t *testing.T) {
	// ⚠ REGRESSION, AND IT WAS A REAL BREAKAGE: `<<` is a literal key in the yaml
	// NODE and never a key in the decoded MAP, so the derived order contained "<<",
	// validateInputOrder rejected it, and a document that had always parsed stopped
	// parsing. inputKeyOrder now filters against the decoded map, which is also what
	// makes "a derived order cannot fail validation" true rather than merely claimed.
	spec, err := Parse([]byte(`---
name: merged
description: d
inputs:
  <<: &defaults
    a: {type: text}
  b: {type: text}
---
Body`))
	require.NoError(t, err)
	assert.NotContains(t, spec.InputOrder, "<<")
	// `b` was written here; `a` arrived by merge and has no authored position, so it
	// sorts into the alphabetical tail rather than being invented one.
	assert.Equal(t, []string{"b", "a"}, spec.OrderedInputKeys())
}

func TestInputOrder_SurvivesTheExportAPI(t *testing.T) {
	// ⚠ THE ROUND-TRIP TEST ABOVE PASSED WHILE THIS FAILED. It called yaml.Marshal on
	// the struct, where the field tag does the work — but a consumer calls ExportFull,
	// which builds a map, and yaml.Marshal SORTS a map's keys. So the documented
	// export path silently re-alphabetized the very order Parse had just recovered.
	spec, err := Parse([]byte(orderedDoc))
	require.NoError(t, err)

	raw, err := spec.ExportFull()
	require.NoError(t, err)
	exported := string(raw)
	assert.Contains(t, exported, "input_order")

	back, err := Parse([]byte(exported))
	require.NoError(t, err)
	assert.Equal(t, spec.OrderedInputKeys(), back.OrderedInputKeys())
}

func TestInputOrder_AnAliasedInputsMappingIsRead(t *testing.T) {
	spec, err := ParseYAMLSpec(`name: x
description: d
_shared: &shared
  zeta: {type: text}
  alpha: {type: text}
inputs: *shared`)
	require.NoError(t, err)
	assert.Equal(t, []string{"zeta", "alpha"}, spec.OrderedInputKeys())
}

func TestInputOrder_PromptyImportDoesNotFabricateAnOrder(t *testing.T) {
	// ⚠ The prompty importer round-trips frontmatter through a Go map before Parse
	// sees it, so the order Parse would derive is ALPHABETICAL — a fabrication
	// indistinguishable downstream from an authored order. The order is taken from
	// the source document instead.
	spec, err := ImportPrompty([]byte(`---
name: legacy
description: d
inputs:
  zeta: {type: text}
  alpha: {type: text}
---
{~prompty.body~}`))
	require.NoError(t, err)
	assert.Equal(t, []string{"zeta", "alpha"}, spec.OrderedInputKeys())
}

func TestInputOrder_ClonePreservesIt(t *testing.T) {
	spec, err := Parse([]byte(orderedDoc))
	require.NoError(t, err)

	clone := spec.Clone()
	assert.Equal(t, spec.OrderedInputKeys(), clone.OrderedInputKeys())

	// Copied, not aliased.
	clone.InputOrder[0] = "mutated"
	assert.Equal(t, "full_name", spec.InputOrder[0])
}
