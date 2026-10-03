package exons

import "sort"

// ResourceMode says how much of one category a definition may use: every item
// of it, only the items the document lists, or none at all (v0.40.0,
// vAudience/atlas#803).
//
// ⛔ requirements.resources IS A DECLARATION OF NEED; resource_modes IS THE
// NARROWING. An entry in resources says "this definition needs a corpus called
// X" and narrows nothing by itself: a kind with no resource_modes key is ALL,
// with or without entries (review M2 — that is what every document written
// before v0.40.0 meant, and a runtime that started narrowing them on a library
// bump would take tools away from working agents). Only an explicit `listed` or
// `none` narrows.
//
// ⛔ ONE CATEGORY, ONE SPELLING. Tools did NOT gain a second
// spelling — tools.allow already says all three things (absent = all, [] =
// none, a list = listed), and Spec.ToolMode reads it. Two spellings for one
// fact is how a runtime ends up enforcing one and displaying the other.
type ResourceMode string

// The ResourceMode vocabulary. An unrecognised value is refused by Validate.
const (
	// ResourceModeAll: every resource of the kind the runtime can bind — the
	// answer for a kind with no resource_modes key. Entries of the kind may sit
	// beside it: they declare what the definition needs, they do not narrow.
	ResourceModeAll ResourceMode = "all"
	// ResourceModeListed: only the requirements.resources entries of the kind. A
	// listed mode with no entry of its kind is refused — it would mean "none"
	// spelled as a list nobody wrote.
	ResourceModeListed ResourceMode = "listed"
	// ResourceModeNone: no resource of the kind at all. Entries of the kind
	// alongside it are refused: the definition would declare a need it forbids.
	ResourceModeNone ResourceMode = "none"
)

// ResourceModeVocabulary lists the vocabulary in its documented order (the
// schema's enum). Named apart from SpecRequirements.ResourceModes, the field.
func ResourceModeVocabulary() []ResourceMode {
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

// ResourceMode returns the mode this definition declares for one resource kind:
// the resource_modes entry when there is one, otherwise ResourceModeAll —
// whether or not requirements.resources declares entries of the kind, because
// those declare a need and narrow nothing (see ResourceMode). So a document
// written before v0.40.0 is never narrowed by a library bump. Safe on a nil
// receiver (→ ResourceModeAll).
func (r *SpecRequirements) ResourceMode(kind string) ResourceMode {
	if r == nil {
		return ResourceModeAll
	}
	if m, ok := r.ResourceModes[kind]; ok {
		return m
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
// vocabulary — and the cross-field rules the schema cannot express: "none"
// alongside entries of that kind is refused (the definition would declare a need
// it forbids), and "listed" with no entry of the kind is refused (it is "none"
// spelled as an empty list nobody wrote). "all" beside entries is valid.
//
// Keys are visited in sorted order so the same document always reports the
// same first problem.
func (r *SpecRequirements) validateResourceModeVocabulary() error {
	if len(r.ResourceModes) == 0 {
		return nil
	}
	if len(r.ResourceModes) > MaxRequirementEntries {
		return NewSpecValidationError(ErrMsgRequirementTooManyEntries, RequirementsFieldResourceModes)
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
		case mode == ResourceModeNone && has:
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
