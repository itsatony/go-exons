package exons

import "sort"

// ResourceMode says how much of one category a definition may use: every item
// of it, only the items the document lists, or none at all (v0.40.0,
// vAudience/atlas#803).
//
// ⛔ ONE CATEGORY, ONE SPELLING. Resources (requirements.resources, grouped by
// kind) gained this three-way answer because "none" had no way to be written:
// an absent list already meant "no narrowing". Tools did NOT gain a second
// spelling — tools.allow already says all three things (absent = all, [] =
// none, a list = listed), and Spec.ToolMode reads it. Two spellings for one
// fact is how a runtime ends up enforcing one and displaying the other.
type ResourceMode string

// The ResourceMode vocabulary. An unrecognised value is refused by Validate.
const (
	// ResourceModeAll: every resource of the kind the runtime can bind. Declaring
	// entries of the kind alongside it is refused — "all" would make them dead.
	ResourceModeAll ResourceMode = "all"
	// ResourceModeListed: only the requirements.resources entries of the kind. A
	// listed mode with no entry of its kind is refused — it would mean "none"
	// spelled as a list nobody wrote.
	ResourceModeListed ResourceMode = "listed"
	// ResourceModeNone: no resource of the kind at all. Entries of the kind
	// alongside it are refused for the same reason as under "all".
	ResourceModeNone ResourceMode = "none"
)

// ResourceModes lists the vocabulary in its documented order (the schema's enum).
func ResourceModes() []ResourceMode {
	return []ResourceMode{ResourceModeAll, ResourceModeListed, ResourceModeNone}
}

// isValidResourceMode reports whether m is in the vocabulary. Unlike access and
// scope, an EMPTY mode is not valid: a key written with no value says nothing,
// and the absent key already has a meaning (ResourceMode derives it).
func isValidResourceMode(m ResourceMode) bool {
	switch m {
	case ResourceModeAll, ResourceModeListed, ResourceModeNone:
		return true
	default:
		return false
	}
}

// ResourceMode returns the mode this definition declares for one resource kind.
//
// A declared resource_modes entry wins. An absent one is DERIVED exactly as a
// document written before v0.40.0 has always meant it: entries of the kind in
// requirements.resources → ResourceModeListed, none → ResourceModeAll. So a
// consumer may call this on every document, old or new, and get the answer the
// author meant. Safe on a nil receiver (→ ResourceModeAll).
func (r *SpecRequirements) ResourceMode(kind string) ResourceMode {
	if r == nil {
		return ResourceModeAll
	}
	if m, ok := r.ResourceModes[kind]; ok {
		return m
	}
	if r.hasResourceOfKind(kind) {
		return ResourceModeListed
	}
	return ResourceModeAll
}

// hasResourceOfKind reports whether any requirements.resources entry has kind.
func (r *SpecRequirements) hasResourceOfKind(kind string) bool {
	for _, res := range r.Resources {
		if res.Kind == kind {
			return true
		}
	}
	return false
}

// validateResourceModes checks resource_modes: at most MaxRequirementEntries
// keys, each a kind in ResourceKindPattern's shape, each value in the
// vocabulary — and the cross-field rules the schema cannot express: "all" or
// "none" alongside entries of that kind is refused (the entries would be dead
// text that a reader takes for a narrowing), and "listed" with no entry of the
// kind is refused (it is "none" spelled as an empty list nobody wrote).
//
// Keys are visited in sorted order so the same document always reports the
// same first problem.
func (r *SpecRequirements) validateResourceModes() error {
	if len(r.ResourceModes) == 0 {
		return nil
	}
	if len(r.ResourceModes) > MaxRequirementEntries {
		return NewSpecValidationError(ErrMsgRequirementTooManyEntries, SpecFieldResourceModes)
	}
	kinds := make([]string, 0, len(r.ResourceModes))
	for k := range r.ResourceModes {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		mode := r.ResourceModes[kind]
		if !resourceKindRegex.MatchString(kind) || requirementFieldTooLong(kind) {
			return NewSpecValidationError(ErrMsgResourceModeKindForm, kind)
		}
		if !isValidResourceMode(mode) {
			return NewSpecValidationError(ErrMsgResourceModeInvalid, kind+"="+string(mode))
		}
		has := r.hasResourceOfKind(kind)
		switch {
		case mode == ResourceModeListed && !has:
			return NewSpecValidationError(ErrMsgResourceModeListedEmpty, kind)
		case mode != ResourceModeListed && has:
			return NewSpecValidationError(ErrMsgResourceModeWithEntries, kind+"="+string(mode))
		}
	}
	return nil
}

// ToolMode returns how much of the tool surface this definition may use, read
// from the ONE place that says it, tools.allow: absent (no tools block, or no
// allow key) → ResourceModeAll; an explicit empty list → ResourceModeNone; a
// non-empty list → ResourceModeListed. There is deliberately no
// resource_modes-style key for tools (see ResourceMode). Safe on a nil spec.
func (s *Spec) ToolMode() ResourceMode {
	if s == nil || s.Tools == nil || s.Tools.Allow == nil {
		return ResourceModeAll
	}
	if len(s.Tools.Allow) == 0 {
		return ResourceModeNone
	}
	return ResourceModeListed
}
