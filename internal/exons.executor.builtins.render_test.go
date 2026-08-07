package internal

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The shapes below are the ones that actually arrive: a prompt-template run posts
// its bound values as JSON, so a multiselect/sort value decodes to []any of string
// and an associate value to []any of map[string]any. Every case that matters is
// therefore expressed through a JSON round-trip rather than as hand-built Go values,
// so the test cannot pass against a renderer that only handles []string.
func decodeJSON(t *testing.T, raw string) any {
	t.Helper()
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	return v
}

func TestRenderValue_TheKindsThatBindComposites(t *testing.T) {
	t.Run("multiselect — a list of scalars reads as prose", func(t *testing.T) {
		assert.Equal(t, "cost, speed", valueToString(decodeJSON(t, `["cost","speed"]`)))
	})

	t.Run("sort — the reordered ranking, in the order it was ranked", func(t *testing.T) {
		// ⚠ Order is the whole POINT of a sort value. Rendering must never sort it.
		assert.Equal(t, "quality, cost, speed", valueToString(decodeJSON(t, `["quality","cost","speed"]`)))
	})

	t.Run("associate — pairs stay distinguishable", func(t *testing.T) {
		got := valueToString(decodeJSON(t, `[{"left":"eu","right":"gdpr"},{"left":"us","right":"ccpa"}]`))
		// Previously: `[map[left:eu right:gdpr] map[left:us right:ccpa]]`.
		assert.Equal(t, "(left: eu, right: gdpr), (left: us, right: ccpa)", got)
	})

	t.Run("file-upload — the accepted filenames", func(t *testing.T) {
		assert.Equal(t, "report.pdf, notes.md", valueToString(decodeJSON(t, `["report.pdf","notes.md"]`)))
	})

	t.Run("a top-level object renders its entries, keys sorted", func(t *testing.T) {
		// Sorted because a Go map has no order: range order would make the same
		// value produce a different prompt on different runs.
		assert.Equal(t, "beta: 2, gamma: 3", valueToString(decodeJSON(t, `{"gamma":3,"beta":2}`)))
		assert.Equal(t, "beta: 2, gamma: 3", valueToString(decodeJSON(t, `{"beta":2,"gamma":3}`)))
	})
}

func TestRenderValue_Scalars(t *testing.T) {
	cases := map[string]struct {
		in   any
		want string
	}{
		"string":       {"hello", "hello"},
		"empty string": {"", ""},
		"nil":          {nil, ""},
		"true":         {true, "true"},
		"false":        {false, "false"},
		"int":          {42, "42"},
		"int64":        {int64(-7), "-7"},
		"uint":         {uint(8), "8"},
		"float64":      {19.99, "19.99"},
		"float64 int":  {float64(3), "3"},
		"float32":      {float32(1.5), "1.5"},
		"bytes":        {[]byte("hi"), "hi"},
		"json number":  {json.Number("12.30"), "12.30"},
		"nil slice":    {[]string(nil), ""},
		"nil map":      {map[string]any(nil), ""},
		"empty slice":  {[]string{}, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, valueToString(tc.in))
		})
	}
}

func TestRenderValue_Separator(t *testing.T) {
	list := decodeJSON(t, `["a","b","c"]`)

	t.Run("the separator applies at the TOP level", func(t *testing.T) {
		assert.Equal(t, "a > b > c", renderValue(list, " > "))
	})

	t.Run("a nested list keeps the default separator and gains brackets", func(t *testing.T) {
		// `join` names the separator between the values the author is talking about.
		// Re-using it one level down would spell two different relations the same way.
		nested := decodeJSON(t, `[["a","b"],["c"]]`)
		assert.Equal(t, "[a, b] > [c]", renderValue(nested, " > "))
	})

	t.Run("an empty separator concatenates", func(t *testing.T) {
		assert.Equal(t, "abc", renderValue(list, ""))
	})
}

func TestRenderValue_Nesting(t *testing.T) {
	t.Run("a map inside a map is parenthesised", func(t *testing.T) {
		got := valueToString(decodeJSON(t, `{"outer":{"inner":1}}`))
		assert.Equal(t, "outer: (inner: 1)", got)
	})

	t.Run("a cycle terminates instead of exhausting the stack", func(t *testing.T) {
		// Not reachable from JSON, but valueToString takes `any` from a
		// caller-supplied Context and a Go caller can hand it a cycle.
		type node struct{ Next any }
		a := &node{}
		a.Next = a
		assert.NotPanics(t, func() { _ = valueToString(map[string]any{"a": a}) })
	})

	// ⚠ THE CYCLE TEST ABOVE PASSED AGAINST A HOLE. A struct field lands on the
	// untraversed `%v` arm, so it never exercised the depth bound at all — while the
	// simplest cycle in Go, a self-referential interface, recursed through the
	// pointer arm without incrementing depth until the stack died. Bound every arm.
	t.Run("a self-referential interface terminates", func(t *testing.T) {
		var i any
		i = &i
		assert.NotPanics(t, func() { _ = valueToString(i) })
	})

	t.Run("a self-referential slice terminates", func(t *testing.T) {
		s := make([]any, 1)
		s[0] = s
		assert.NotPanics(t, func() { _ = valueToString(s) })
	})
}

func TestRenderValue_Structs(t *testing.T) {
	// Not reachable from a JSON-decoded value, but a Go caller can seed a Context
	// with one — and `%v` gave `{Ada 36}`, the same gibberish this file exists to
	// stop. Declaration order, because a struct has one.
	type person struct {
		Name   string
		Age    int
		hidden string //nolint:unused // present to prove unexported fields are skipped
	}
	assert.Equal(t, "Name: Ada, Age: 36", valueToString(person{Name: "Ada", Age: 36, hidden: "x"}))
}

func TestRenderValue_ElidesPastTheDepthBound(t *testing.T) {
	// Says something was left out, rather than dumping a truncated debug form —
	// and unlike `%v`, cannot itself recurse forever.
	deep := any("leaf")
	for i := 0; i < 12; i++ {
		deep = []any{deep}
	}
	assert.Contains(t, valueToString(deep), elidedValue)
}

func TestRenderValue_MapKeysAreSortedAsKEYS(t *testing.T) {
	// ⚠ Sorting the rendered `"k: v"` entries looks equivalent and is not: ':' sorts
	// after digits and space, so a key that is a prefix of another came out reversed
	// while the comment claimed "keys sorted". Deterministic, but not what it said.
	assert.Equal(t, "a: 1, a1: 2", valueToString(decodeJSON(t, `{"a1":2,"a":1}`)))
	assert.Equal(t, "item: 1, item 2: 2", valueToString(decodeJSON(t, `{"item 2":2,"item":1}`)))
}

func TestRenderValue_NamedByteSlicesAreText(t *testing.T) {
	// Matched on the element kind, not on `case []byte` — a named type would
	// otherwise render as `104, 105`.
	type blob []byte
	assert.Equal(t, "hi", valueToString(blob("hi")))
	assert.Equal(t, `{"a":1}`, valueToString(json.RawMessage(`{"a":1}`)))
}

func TestRenderValue_TimeIsRFC3339(t *testing.T) {
	// time.Time IS a fmt.Stringer, and its String() is Go's debug form. Deliberate
	// case ahead of the Stringer arm.
	ts := time.Date(2026, 8, 7, 9, 30, 0, 0, time.UTC)
	assert.Equal(t, "2026-08-07T09:30:00Z", valueToString(ts))
}

// A Stringer must not be shadowed by the composite arms, and a concrete scalar must
// not be shadowed by a Stringer — the ordering inside renderAtDepth is load-bearing.
type stringerSlice []string

func (s stringerSlice) String() string { return "I am a slice with an opinion" }

func TestRenderValue_StringerWinsOverTheCompositeArm(t *testing.T) {
	assert.Equal(t, "I am a slice with an opinion", valueToString(stringerSlice{"a", "b"}))
}
