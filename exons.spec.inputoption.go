package exons

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// A select option may be written as a bare string (v0.44.0, vAudience/aigentverse#258).
//
//	options: [sachlich, locker, begeistert]
//
// is the most natural way to write a dropdown, and the shape anyone porting a
// comma-separated option list tries first. Until v0.44.0 it failed the whole Parse with
// "cannot unmarshal !!str `sachlich` into exons.InputOption". A bare string now decodes
// to InputOption{Value: s} — Label stays EMPTY, because the documented fallback
// (Label falls back to Value) already covers display, and an empty Label keeps the
// re-serialised entry short.
//
// ⚠ THE SCALAR FORM IS STRICTER THAN THE MAPPING FORM, ON PURPOSE. `- {value: ""}` has
// always parsed into an option with an empty value and still does (byte-identical:
// a stored document must keep loading). A bare EMPTY string is refused instead: the new
// form has no stored documents to protect, and an empty or whitespace-only entry in a
// string list is far more often a stray comma or a half-deleted line than an intended
// empty value. A YAML null entry (`[a, ~]`) never reaches this method — yaml.v3 drops a
// null element of a non-pointer slice before any Unmarshaler is consulted, exactly as it
// did for the mapping form — and the schema refuses it (schemaStricterNull).
//
// ⚠ ROUND-TRIP: there is deliberately NO MarshalYAML/MarshalJSON. A parsed bare string
// is written back by Serialize as the mapping form (`- value: sachlich`), which every
// consumer that predates v0.44.0 already reads. Emitting a bare string would change the
// serialised bytes of every existing `{value: x}` option without a label — a silent
// format change for consumers that parse the frontmatter themselves.
//
// The same type backs InputDef.AssociateWith (the right-hand set of an associate
// input), so `associate_with: [analyst, reviewer]` is accepted by the same rule.

// UnmarshalYAML decodes an option from either a bare scalar (the value) or a
// {value, label} mapping. The mapping is decoded through a method-less alias, so its
// behaviour is byte-identical to the plain struct decode it replaces. Errors are
// *yaml.TypeError entries ("line N: …"), so they aggregate with yaml.v3's own
// unmarshal errors and keep a position NewFrontmatterParseErrorAt can locate.
func (o *InputOption) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// Decode rather than read node.Value, so a tagged scalar (`!!binary …`) is
		// resolved exactly as it would be in the mapping form's `value:` field.
		var s string
		if err := node.Decode(&s); err != nil {
			return err
		}
		if strings.TrimSpace(s) == "" {
			return &yaml.TypeError{Errors: []string{fmt.Sprintf(ErrFmtOptionEmptyScalar, node.Line)}}
		}
		*o = InputOption{Value: s}
		return nil
	case yaml.MappingNode:
		type rawInputOption InputOption
		var raw rawInputOption
		if err := node.Decode(&raw); err != nil {
			return err
		}
		*o = InputOption(raw)
		return nil
	default:
		return &yaml.TypeError{Errors: []string{fmt.Sprintf(ErrFmtOptionShape, node.Line, yamlKindName(node.Kind))}}
	}
}

// UnmarshalJSON is UnmarshalYAML's twin for a JSON-sourced spec: a JSON string is the
// value, an object is decoded exactly as before, and `null` is a no-op (encoding/json's
// convention for Unmarshalers, and what the plain struct decode did). An empty or
// whitespace-only string is refused, as in YAML.
func (o *InputOption) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	switch {
	case bytes.Equal(trimmed, []byte("null")):
		return nil
	case len(trimmed) > 0 && trimmed[0] == '"':
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		if strings.TrimSpace(s) == "" {
			return errors.New(ErrMsgOptionEmptyJSON)
		}
		*o = InputOption{Value: s}
		return nil
	case len(trimmed) > 0 && trimmed[0] == '{':
		type rawInputOption InputOption
		var raw rawInputOption
		if err := json.Unmarshal(trimmed, &raw); err != nil {
			return err
		}
		*o = InputOption(raw)
		return nil
	default:
		return errors.New(ErrMsgOptionShapeJSON)
	}
}

// yamlKindName names a node kind in an author-facing message.
func yamlKindName(k yaml.Kind) string {
	switch k {
	case yaml.SequenceNode:
		return yamlKindSequence
	case yaml.MappingNode:
		return yamlKindMapping
	case yaml.ScalarNode:
		return yamlKindScalar
	case yaml.AliasNode:
		return yamlKindAlias
	case yaml.DocumentNode:
		return yamlKindDocument
	default:
		return yamlKindUnknown
	}
}
