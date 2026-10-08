package exons

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// optionDoc is a prompt template whose one input `ton` is a select with the given
// options body (indented under `options:` by the caller).
func optionDoc(options string) string {
	return "---\nname: kurs\ndescription: d\ntype: prompt\ninputs:\n  ton:\n    type: select\n    options:" + options + "\n---\nbody"
}

// A select option may be written as a bare string (vAudience/aigentverse#258).
func TestInputOption_ABareStringIsTheValue(t *testing.T) {
	want := []InputOption{{Value: "sachlich"}, {Value: "locker"}, {Value: "begeistert"}}
	cases := map[string]string{
		"flow form":  " [sachlich, locker, begeistert]",
		"block form": "\n      - sachlich\n      - locker\n      - begeistert",
		"quoted":     " [\"sachlich\", 'locker', begeistert]",
	}
	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			spec, err := Parse([]byte(optionDoc(options)))
			require.NoError(t, err)
			assert.Equal(t, want, spec.Inputs["ton"].Options,
				"a bare string is the VALUE; the label stays empty and falls back to the value")
		})
	}
}

func TestInputOption_AMixedListDecodesEachEntryByItsOwnShape(t *testing.T) {
	spec, err := Parse([]byte(optionDoc("\n      - sachlich\n      - {value: locker, label: Locker}\n      - value: begeistert")))
	require.NoError(t, err)
	assert.Equal(t, []InputOption{
		{Value: "sachlich"},
		{Value: "locker", Label: "Locker"},
		{Value: "begeistert"},
	}, spec.Inputs["ton"].Options)
}

// The mapping form decodes exactly as the plain struct did — including the
// pre-existing leniency for an empty value and an ignored unknown key.
func TestInputOption_TheMappingFormIsUnchanged(t *testing.T) {
	spec, err := Parse([]byte(optionDoc("\n      - {value: a, label: A}\n      - {value: \"\"}\n      - {value: b, colour: red}\n      - {}")))
	require.NoError(t, err)
	assert.Equal(t, []InputOption{{Value: "a", Label: "A"}, {Value: ""}, {Value: "b"}, {}}, spec.Inputs["ton"].Options)
}

// A non-string scalar is coerced the way the mapping form's `value: 42` always was.
func TestInputOption_ANonStringScalarIsCoercedLikeTheMappingForm(t *testing.T) {
	spec, err := Parse([]byte(optionDoc(" [1, 2.5, true]")))
	require.NoError(t, err)
	assert.Equal(t, []InputOption{{Value: "1"}, {Value: "2.5"}, {Value: "true"}}, spec.Inputs["ton"].Options)
}

// The scalar form is STRICTER than the mapping form: an empty or blank bare string is
// refused, where `{value: ""}` still parses (a stored document must keep loading).
func TestInputOption_AnEmptyBareStringIsRefused(t *testing.T) {
	for name, options := range map[string]string{
		"empty flow entry":  " [a, \"\"]",
		"blank block entry": "\n      - a\n      - '  '",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(optionDoc(options)))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "an option written as a bare string must not be empty")
			assert.Contains(t, err.Error(), `key "inputs.ton.options"`)
		})
	}
}

// A YAML null entry never reaches the Unmarshaler: yaml.v3 drops it, as it did before
// v0.44.0 for the mapping form. Pinned so a yaml.v3 change in that behaviour is seen.
func TestInputOption_ANullEntryIsDroppedNotDecoded(t *testing.T) {
	spec, err := Parse([]byte(optionDoc(" [a, ~, b]")))
	require.NoError(t, err)
	assert.Equal(t, []InputOption{{Value: "a"}, {Value: "b"}}, spec.Inputs["ton"].Options)
}

func TestInputOption_ANestedSequenceIsAReadableError(t *testing.T) {
	_, err := Parse([]byte(optionDoc(" [[a, b], c]")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "an option must be a string or a {value, label} mapping, not a sequence")
	var te *yaml.TypeError
	assert.True(t, errors.As(err, &te), "the cause is still yaml.v3's TypeError")
}

// associate_with is the same type, so it takes the same two shapes.
func TestInputOption_AssociateWithTakesBareStringsToo(t *testing.T) {
	doc := "---\nname: kurs\ndescription: d\ntype: prompt\ninputs:\n  owners:\n    type: associate\n    options: [region, {value: team, label: Team}]\n    associate_with: [analyst, reviewer]\n---\nbody"
	spec, err := Parse([]byte(doc))
	require.NoError(t, err)
	assert.Equal(t, []InputOption{{Value: "region"}, {Value: "team", Label: "Team"}}, spec.Inputs["owners"].Options)
	assert.Equal(t, []InputOption{{Value: "analyst"}, {Value: "reviewer"}}, spec.Inputs["owners"].AssociateWith)
}

// The bound value is checked against a bare-string option's value.
func TestInputOption_ABareStringOptionAllowsItsValue(t *testing.T) {
	spec, err := Parse([]byte(optionDoc(" [sachlich, locker]")))
	require.NoError(t, err)
	assert.True(t, optionAllows(spec.Inputs["ton"].Options, "locker"))
	assert.False(t, optionAllows(spec.Inputs["ton"].Options, "Locker"))
	assert.Empty(t, spec.ValidateInputBinding(map[string]any{"ton": "sachlich"}))
	assert.Len(t, spec.ValidateInputBinding(map[string]any{"ton": "laut"}), 1)
}

// ROUND-TRIP: a parsed bare string is written back in the MAPPING form, which every
// consumer predating v0.44.0 reads, and re-parses to the same option. There is
// deliberately no MarshalYAML: emitting bare strings would change the bytes of every
// existing label-less `{value: x}` option.
func TestInputOption_ABareStringSerializesAsTheMappingForm(t *testing.T) {
	spec, err := Parse([]byte(optionDoc(" [sachlich, locker]")))
	require.NoError(t, err)

	out, err := spec.Serialize(DefaultSerializeOptions())
	require.NoError(t, err)
	assert.Contains(t, string(out), "- value: sachlich\n")
	assert.NotContains(t, string(out), "label:", "an empty label is omitted, so the entry stays short")

	again, err := Parse(out)
	require.NoError(t, err)
	assert.Equal(t, spec.Inputs["ton"].Options, again.Inputs["ton"].Options)

	js, err := json.Marshal(spec.Inputs["ton"].Options)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"value":"sachlich"},{"value":"locker"}]`, string(js))
}

// The JSON twin: a JSON-sourced spec agrees with the YAML one.
func TestInputOption_UnmarshalJSON(t *testing.T) {
	t.Run("strings, objects and a mix", func(t *testing.T) {
		var got []InputOption
		require.NoError(t, json.Unmarshal([]byte(`["sachlich", {"value": "locker", "label": "Locker"}, {"value": ""}]`), &got))
		assert.Equal(t, []InputOption{{Value: "sachlich"}, {Value: "locker", Label: "Locker"}, {Value: ""}}, got)
	})
	t.Run("null is a no-op, as the plain struct decode was", func(t *testing.T) {
		var got []InputOption
		require.NoError(t, json.Unmarshal([]byte(`[null]`), &got))
		assert.Equal(t, []InputOption{{}}, got)
	})
	t.Run("a whole InputDef", func(t *testing.T) {
		var def InputDef
		require.NoError(t, json.Unmarshal([]byte(`{"type":"associate","options":["a"],"associate_with":["b"]}`), &def))
		assert.Equal(t, []InputOption{{Value: "a"}}, def.Options)
		assert.Equal(t, []InputOption{{Value: "b"}}, def.AssociateWith)
	})
	for name, tc := range map[string]struct{ in, want string }{
		"empty string": {`[""]`, ErrMsgOptionEmptyJSON},
		"blank string": {`["  "]`, ErrMsgOptionEmptyJSON},
		"number":       {`[42]`, ErrMsgOptionShapeJSON},
		"array":        {`[["a"]]`, ErrMsgOptionShapeJSON},
	} {
		t.Run("refuses "+name, func(t *testing.T) {
			var got []InputOption
			err := json.Unmarshal([]byte(tc.in), &got)
			require.Error(t, err)
			assert.True(t, strings.Contains(err.Error(), tc.want), "got %v", err)
		})
	}
}
