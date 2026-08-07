package exons

import (
	"sort"

	"gopkg.in/yaml.v3"
)

// Authored input order — recovering the one thing `Inputs map[string]*InputDef`
// throws away.
//
// A YAML mapping HAS an order; Go's map does not. yaml.Unmarshal drops it at the
// moment of decode, which is why every downstream form projection had to sort by
// key and document that it was doing so. The order is recoverable only here, from
// the yaml.Node, before the map exists.

// UnmarshalYAML decodes a Spec and captures the authored order of its `inputs`
// mapping into InputOrder.
//
// The decode itself is delegated verbatim to a type alias, which has no methods and
// therefore does not recurse back into here, and which carries the identical field
// tags — including `yaml:",inline"` on Extensions, so the catch-all keeps catching
// exactly what it caught before. This method adds information; it changes no
// existing decode behaviour.
//
// ⚠ One documented consequence of taking over the decode: `node.Decode` starts a
// fresh decode, so a PARENT decoder's KnownFields(true) no longer reaches a Spec
// nested inside another struct. Moot in practice — the `,inline` Extensions map
// absorbs every unknown key, so strict decoding of a Spec never rejected anything
// anyway — but it is a real difference, recorded rather than discovered later.
func (s *Spec) UnmarshalYAML(node *yaml.Node) error {
	type rawSpec Spec
	var raw rawSpec
	if err := node.Decode(&raw); err != nil {
		return err
	}
	*s = Spec(raw)

	// An explicitly authored `input_order:` wins. It is the escape hatch for a Spec
	// that never came from a YAML mapping (one rebuilt from JSON, say), and honouring
	// the document over a derivation is the same rule the rest of the parser follows.
	if len(s.InputOrder) == 0 {
		s.InputOrder = inputKeyOrder(node, s.Inputs)
	}
	return nil
}

// inputKeyOrder reads the key order of the top-level `inputs` mapping out of a
// document node, keeping only keys that survived into the DECODED map.
//
// ⚠ THE FILTER IS NOT TIDINESS — IT IS WHAT MAKES A DERIVED ORDER CORRECT BY
// CONSTRUCTION. A YAML mapping's raw key list is not the same set as the decoded
// map's: a merge key `<<: *defaults` is a literal key in the node and never a key in
// the map. Without the filter, a document using `<<` inside `inputs:` derived an
// order containing "<<", which validateInputOrder then rejected — so a document that
// had always parsed suddenly did not, and the comment promising "a derived order
// cannot fail these checks" was false. Filtering here means that promise holds, and
// the merged-in inputs simply sort into the alphabetical tail rather than being
// invented an order they never had.
//
// Returns nil when there is no `inputs` mapping to read — no inputs at all, an
// `inputs: *alias` whose order lives somewhere else entirely, or a shape the decode
// above has already rejected.
func inputKeyOrder(node *yaml.Node, declared map[string]*InputDef) []string {
	mapping := resolveAlias(node)
	if mapping != nil && mapping.Kind == yaml.DocumentNode && len(mapping.Content) == 1 {
		mapping = resolveAlias(mapping.Content[0])
	}
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}

	// A mapping node's Content alternates key, value, key, value…
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value != SpecFieldInputs {
			continue
		}
		inputs := resolveAlias(mapping.Content[i+1])
		if inputs == nil || inputs.Kind != yaml.MappingNode {
			return nil
		}
		keys := make([]string, 0, len(inputs.Content)/2)
		for j := 0; j+1 < len(inputs.Content); j += 2 {
			key := inputs.Content[j].Value
			if _, ok := declared[key]; !ok {
				continue
			}
			keys = append(keys, key)
		}
		if len(keys) == 0 {
			return nil
		}
		return keys
	}
	return nil
}

// resolveAlias follows an `*anchor` reference to the node it names, so an aliased
// `inputs:` is read rather than silently dropped to alphabetical.
func resolveAlias(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
}

// OrderedInputKeys returns every declared input key exactly once, in authored order.
//
// This is the method a form projection should walk — NOT InputOrder, and not a
// `range` over Inputs. It is total where both of those are partial: the keys named
// in InputOrder come first in that order (skipping any that name no declared input,
// which Validate rejects but a hand-built Spec can still contain), then every
// remaining input sorted by key, so a Spec assembled in Go without an order still
// yields a stable one rather than Go's randomised map iteration.
//
// Returns nil for a spec with no inputs.
func (s *Spec) OrderedInputKeys() []string {
	if s == nil || len(s.Inputs) == 0 {
		return nil
	}

	out := make([]string, 0, len(s.Inputs))
	placed := make(map[string]struct{}, len(s.Inputs))
	for _, k := range s.InputOrder {
		if _, declared := s.Inputs[k]; !declared {
			continue
		}
		if _, already := placed[k]; already {
			continue
		}
		placed[k] = struct{}{}
		out = append(out, k)
	}

	rest := make([]string, 0, len(s.Inputs)-len(out))
	for k := range s.Inputs {
		if _, already := placed[k]; !already {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

// validateInputOrder checks a DECLARED input order against the inputs it orders.
// A derived order cannot fail these — inputKeyOrder filters it against the decoded
// map — so every error here is an author's contradiction, stated rather than absorbed.
//
// ⚠ IT RUNS FROM Validate, SO IT RUNS ON THE Parse PATH ONLY. ParseYAMLSpec
// validates nothing at all — not the name, not the description, not this — and that
// leniency is its contract, relied on by consumers projecting a form from an
// in-progress draft. The safety net for that path is OrderedInputKeys, which is
// total: it drops an entry naming no declared input and dedups a repeated one, so a
// bad order degrades to a different ORDER rather than to a missing field.
func (s *Spec) validateInputOrder() error {
	if len(s.InputOrder) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(s.InputOrder))
	for _, k := range s.InputOrder {
		if _, dup := seen[k]; dup {
			return NewSpecInputOrderDuplicateError(k)
		}
		seen[k] = struct{}{}
		if _, declared := s.Inputs[k]; !declared {
			return NewSpecInputOrderUnknownError(k)
		}
	}
	return nil
}
