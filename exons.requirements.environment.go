package exons

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// requirements.environment (v0.33.0, go-exons#4): a skill or agent declares the
// EXECUTION ENVIRONMENT it needs, so a registry or runtime can preflight — decide
// before activation whether the definition can work at all — instead of failing
// mid-conversation. A template that drives a binary is inert without code
// execution, and nothing else in a definition says so.
//
// ⛔ ABSTRACT, NEVER A RUNTIME OR IMAGE NAME. "code execution with python:openpyxl"
// is portable; "forgebox-office:1.4" or "python:3.12-slim" names one platform's
// artefact and makes the definition meaningless everywhere else. Which sandbox,
// image or interpreter satisfies the declaration is the runtime's decision.
//
// ⚠ ABSENT IS NOT "NONE". Every field is optional and an absent field means the
// author SAID NOTHING about it — never that the definition does not need it. A
// consumer that reads an undeclared network as "works offline" invents a promise
// the author never made. `network: none` is the explicit "needs no network".

// Code-execution values.
//   - required: the definition cannot work without executing code.
//   - optional: it executes code when the environment offers it and degrades
//     (does less, not nothing) without.
//
// There is deliberately no "none": a definition that executes no code simply
// does not declare code_execution.
const (
	EnvironmentCodeExecutionRequired = "required"
	EnvironmentCodeExecutionOptional = "optional"
)

// Network values.
//   - required: the definition cannot work without outbound network access.
//   - optional: it uses the network when offered and degrades without.
//   - none:     it needs NO network — a runtime may place it in a network-less
//     sandbox. This is the one value that is a positive statement about absence,
//     which is why it exists: an undeclared network is unknown, not none.
const (
	EnvironmentNetworkRequired = "required"
	EnvironmentNetworkOptional = "optional"
	EnvironmentNetworkNone     = "none"
)

// Package ecosystems accepted before the ':' of a packages entry. The set is
// CLOSED so two runtimes cannot disagree about spelling ("pypi" vs "python",
// "npm" vs "node"); a new ecosystem is an additive release.
//   - python: a PyPI distribution (python:openpyxl)
//   - node:   an npm package, scoped names included (node:@scope/pkg)
//   - r:      a CRAN package
//   - ruby:   a RubyGems gem
//   - rust:   a crates.io crate
//   - go:     a Go module path (go:golang.org/x/text)
//   - java:   a Maven artefact (java:org.apache.poi/poi)
//   - system: an OS-level tool the definition shells out to (system:libreoffice,
//     system:ffmpeg) — a TOOL, never a distribution or image name
const (
	EnvironmentEcosystemPython = "python"
	EnvironmentEcosystemNode   = "node"
	EnvironmentEcosystemR      = "r"
	EnvironmentEcosystemRuby   = "ruby"
	EnvironmentEcosystemRust   = "rust"
	EnvironmentEcosystemGo     = "go"
	EnvironmentEcosystemJava   = "java"
	EnvironmentEcosystemSystem = "system"
)

// EnvironmentEcosystems lists the accepted ecosystems in their documented order.
// It returns a fresh slice so a caller cannot widen the vocabulary.
func EnvironmentEcosystems() []string {
	return []string{
		EnvironmentEcosystemPython, EnvironmentEcosystemNode, EnvironmentEcosystemR,
		EnvironmentEcosystemRuby, EnvironmentEcosystemRust, EnvironmentEcosystemGo,
		EnvironmentEcosystemJava, EnvironmentEcosystemSystem,
	}
}

// EnvironmentPackagePattern is the whole shape of a packages entry:
// `<ecosystem>:<name>`. The name is a bare package name — letters, digits and
// . _ @ / + - — so a version specifier (">=3.1", "==2"), whitespace, a second
// ':' and therefore any URL are refused. Versions are deliberately not
// expressible: the field is INFORMATIONAL, and a version pin would read as a
// promise a runtime is expected to honour. The schema states the same pattern;
// schema/agreement_test.go holds the two together.
const EnvironmentPackagePattern = `^(python|node|r|ruby|rust|go|java|system):[A-Za-z0-9@][A-Za-z0-9._@/+-]*$`

var environmentPackageRegex = regexp.MustCompile(EnvironmentPackagePattern)

// Bounds on requirements.environment.packages. Packages are informational, so the
// bound is generous for a real skill and small against a flooded document.
// Entries are ASCII by pattern, so the length is the same in runes and bytes.
const (
	MaxEnvironmentPackages   = 64
	MaxEnvironmentPackageLen = 128
)

// MaxCompatibilityLength (500, exons.constants.go) is the agentskills.io bound on
// the `compatibility` field that AgentSkillsCompatibility renders into.

// AgentSkillsFieldCompatibility is the agentskills.io frontmatter key for free-text
// environment requirements. go-exons has no typed field for it: an author's
// value lives in Spec.Extensions under this key.
const AgentSkillsFieldCompatibility = "compatibility"

// EnvironmentRequirement declares the abstract execution environment a skill or
// agent needs. Every field is optional; see the file comment for why absent is
// "unknown", not "none".
//
// An empty block (`environment: {}`) is valid and declares nothing; a YAML
// re-emission drops it (yaml.v3 omits a pointer to a zero struct), which is the
// same declaration. A prompt refuses the block in any form, empty included, as
// the schema does. Packages has
// no empty-vs-absent distinction — `packages: []` and no packages both declare
// no packages — so omitempty loses nothing here.
type EnvironmentRequirement struct {
	// CodeExecution is EnvironmentCodeExecutionRequired or
	// EnvironmentCodeExecutionOptional; empty means undeclared.
	CodeExecution string `yaml:"code_execution,omitempty" json:"code_execution,omitempty"`
	// Packages are INFORMATIONAL `<ecosystem>:<name>` entries
	// (EnvironmentPackagePattern) the definition expects the executing
	// environment to provide. Unique, compared verbatim, order preserved. A
	// runtime may preflight on them; nothing in go-exons installs anything.
	// Declaring any requires CodeExecution.
	Packages []string `yaml:"packages,omitempty" json:"packages,omitempty"`
	// Network is EnvironmentNetworkRequired, EnvironmentNetworkOptional or
	// EnvironmentNetworkNone; empty means undeclared (NOT none).
	Network string `yaml:"network,omitempty" json:"network,omitempty"`
}

// IsZero reports whether the block declares nothing (nil, or every field empty).
func (e *EnvironmentRequirement) IsZero() bool {
	return e == nil || (e.CodeExecution == "" && len(e.Packages) == 0 && e.Network == "")
}

// RequiresCodeExecution reports whether the definition cannot work without code
// execution — the preflight question a runtime without a sandbox asks first.
func (e *EnvironmentRequirement) RequiresCodeExecution() bool {
	return e != nil && e.CodeExecution == EnvironmentCodeExecutionRequired
}

// RequiresNetwork reports whether the definition cannot work without network
// access. False for optional, none AND undeclared — ask Network directly to
// tell those apart.
func (e *EnvironmentRequirement) RequiresNetwork() bool {
	return e != nil && e.Network == EnvironmentNetworkRequired
}

// Validate checks the environment block: the code_execution and network enums,
// the packages bound, length, format and uniqueness, and that packages are only
// declared alongside code_execution. A nil block is valid. Values are compared
// verbatim, never trimmed or case-folded, like every other requirements list.
//
// Unknown keys inside `environment:` are ignored by the parser, as everywhere
// else in requirements; the published schema closes the object and refuses them
// (the schema-stricter "closed" class of schema/agreement_test.go).
func (e *EnvironmentRequirement) Validate() error {
	if e == nil {
		return nil
	}
	switch e.CodeExecution {
	case "", EnvironmentCodeExecutionRequired, EnvironmentCodeExecutionOptional:
	default:
		return NewSpecValidationError(ErrMsgEnvironmentCodeExecution, e.CodeExecution)
	}
	switch e.Network {
	case "", EnvironmentNetworkRequired, EnvironmentNetworkOptional, EnvironmentNetworkNone:
	default:
		return NewSpecValidationError(ErrMsgEnvironmentNetwork, e.Network)
	}
	if len(e.Packages) > MaxEnvironmentPackages {
		return NewSpecValidationError(ErrMsgEnvironmentTooMany, "")
	}
	seen := make(map[string]struct{}, len(e.Packages))
	for _, p := range e.Packages {
		if utf8.RuneCountInString(p) > MaxEnvironmentPackageLen {
			return NewSpecValidationError(ErrMsgEnvironmentPackageLong, "")
		}
		if !environmentPackageRegex.MatchString(p) {
			return NewSpecValidationError(ErrMsgEnvironmentPackageForm, p)
		}
		if _, dup := seen[p]; dup {
			return NewSpecValidationError(ErrMsgEnvironmentPackageDup, p)
		}
		seen[p] = struct{}{}
	}
	if len(e.Packages) > 0 && e.CodeExecution == "" {
		return NewSpecValidationError(ErrMsgEnvironmentPackagesNeedExecution, "")
	}
	return nil
}

// Clone returns a deep copy of the block (nil for nil).
func (e *EnvironmentRequirement) Clone() *EnvironmentRequirement {
	if e == nil {
		return nil
	}
	clone := *e
	if e.Packages != nil {
		clone.Packages = make([]string, len(e.Packages))
		copy(clone.Packages, e.Packages)
	}
	return &clone
}

// Rendered sentence parts, one spelling each. Unexported: the sentence is prose
// for people, and a consumer should decide on the structured block, never on it.
const (
	compatCodeRequired     = "Requires code execution"
	compatCodeOptional     = "Uses code execution when available"
	compatPackagesJoin     = " with packages "
	compatPackagesMoreFmt  = " and %d more"
	compatPackagesCountFmt = " with %d packages"
	compatNetworkRequired  = "Requires network access."
	compatNetworkOptional  = "Uses network access when available."
	compatNetworkNone      = "Needs no network access."
	compatSentenceEnd      = "."
	compatPackageSeparator = ", "
)

// CompatibilitySentence renders the block as agentskills.io `compatibility` prose
// — at most MaxCompatibilityLength (500) characters, empty when the block
// declares nothing. Deterministic: packages appear in declared order.
//
// Examples:
//
//	Requires code execution with packages python:openpyxl, python:pandas. Needs no network access.
//	Uses code execution when available. Requires network access.
//
// ⚠ IT NEVER TRUNCATES A PACKAGE NAME AND NEVER DROPS ONE SILENTLY: when the list
// does not fit, it renders as many whole entries as do and says how many it left
// out (" and 12 more"). A cut name would state a package that does not exist; an
// omitted one would make the sentence claim a shorter list than the document.
func (e *EnvironmentRequirement) CompatibilitySentence() string {
	if e.IsZero() {
		return ""
	}
	var network string
	switch e.Network {
	case EnvironmentNetworkRequired:
		network = compatNetworkRequired
	case EnvironmentNetworkOptional:
		network = compatNetworkOptional
	case EnvironmentNetworkNone:
		network = compatNetworkNone
	}

	var code string
	switch e.CodeExecution {
	case EnvironmentCodeExecutionRequired:
		code = compatCodeRequired
	case EnvironmentCodeExecutionOptional:
		code = compatCodeOptional
	}
	if code == "" {
		// Validate refuses packages without code_execution; an unvalidated block
		// renders only what it can state truthfully.
		return network
	}

	// Budget for the code clause: the whole limit minus the network sentence and
	// the space before it.
	budget := MaxCompatibilityLength
	if network != "" {
		budget -= utf8.RuneCountInString(network) + 1
	}
	clause := code + compatPackagesClause(e.Packages, budget-utf8.RuneCountInString(code)-len(compatSentenceEnd)) + compatSentenceEnd
	if network == "" {
		return clause
	}
	return clause + " " + network
}

// compatPackagesClause renders " with packages a, b" (or " with packages a, b and
// N more") within budget characters, or "" when there are no packages. When not
// even one whole entry fits beside its suffix, only the count is stated
// (" with N packages").
func compatPackagesClause(pkgs []string, budget int) string {
	if len(pkgs) == 0 {
		return ""
	}
	full := compatPackagesJoin + strings.Join(pkgs, compatPackageSeparator)
	if utf8.RuneCountInString(full) <= budget {
		return full
	}
	// Fit as many whole entries as leave room for the " and N more" suffix.
	var sb strings.Builder
	sb.WriteString(compatPackagesJoin)
	used := utf8.RuneCountInString(compatPackagesJoin)
	shown := 0
	for i, p := range pkgs {
		sep := ""
		if i > 0 {
			sep = compatPackageSeparator
		}
		// Packages are ASCII by pattern; the rune count keeps an unvalidated
		// block within the limit too.
		add := utf8.RuneCountInString(sep) + utf8.RuneCountInString(p)
		if used+add+utf8.RuneCountInString(compatMore(len(pkgs)-i-1)) > budget {
			break
		}
		sb.WriteString(sep)
		sb.WriteString(p)
		used += add
		shown++
	}
	if shown == 0 {
		return fmt.Sprintf(compatPackagesCountFmt, len(pkgs))
	}
	return sb.String() + compatMore(len(pkgs)-shown)
}

// compatMore renders the " and N more" suffix, or "" for zero.
func compatMore(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf(compatPackagesMoreFmt, n)
}

// AgentSkillsCompatibility composes the agentskills.io `compatibility` value for
// this document: the author's own `compatibility` text (Spec.Extensions), then the
// sentence rendered from requirements.environment. At most 500 characters; empty
// when neither exists.
//
// ⚠ THE AUTHOR'S TEXT IS APPENDED TO, NEVER REPLACED AND NEVER CUT. An author who
// wrote compatibility prose chose those words, so they stay first and verbatim;
// the structured block is appended because a portable consumer that sees only the
// prose would otherwise miss a declared requirement — the silent failure this
// helper exists to prevent — while a duplicated statement is merely redundant.
// Two exceptions keep the result truthful and within bounds:
//   - the rendered sentence is not appended when the author's text already
//     contains it verbatim (a SKILL.md exported by this helper and re-imported), so
//     repeated export → import → export does not grow the field;
//   - when author text plus sentence exceed 500 characters, the author's text is
//     returned alone (the structured block still travels in every full export).
//
// An author value that is not a non-empty string of at most 500 characters is not
// a valid agentskills.io compatibility and is treated as absent here; it is still
// preserved verbatim in Extensions for full exports.
//
// Rendering happens at EXPORT only. Nothing writes the composed text back into
// the Spec, so a full-fidelity round trip keeps the author's field unchanged.
func (s *Spec) AgentSkillsCompatibility() string {
	if s == nil {
		return ""
	}
	author := ""
	if v, ok := s.Extensions[AgentSkillsFieldCompatibility].(string); ok {
		if n := utf8.RuneCountInString(v); n > 0 && n <= MaxCompatibilityLength {
			author = v
		}
	}
	var sentence string
	if s.Requirements != nil {
		sentence = s.Requirements.Environment.CompatibilitySentence()
	}
	switch {
	case sentence == "":
		return author
	case author == "":
		return sentence
	case strings.Contains(author, sentence):
		return author
	}
	combined := strings.TrimRight(author, " \t\r\n") + " " + sentence
	if utf8.RuneCountInString(combined) > MaxCompatibilityLength {
		return author
	}
	return combined
}
