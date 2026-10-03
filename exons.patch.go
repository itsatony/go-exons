package exons

import (
	"bytes"
	"errors"
	"sort"
	"strings"

	"github.com/itsatony/go-cuserr"
	"github.com/itsatony/go-exons/execution"
	"gopkg.in/yaml.v3"
)

// ⛔ WHY PatchSource EXISTS (v0.40.0, vAudience/atlas#803).
//
// A consumer that edits ONE field of a stored definition — an agent's tool list, its model, the
// corpora it may read — had only one route: Parse, change the Spec, Serialize. That route damages
// the document in ways no reviewer sees in the diff they asked for:
//
//   - ExportFull drops `credentials:` / `credential:` (they are excluded by design, FullExport-
//     WithCredentials exists for that), so every edit through it strips the agent's credentials;
//   - yaml.Marshal of a map discards every comment and re-sorts every key;
//   - Engine.Parse RENDERS `{~…~}` tags in the frontmatter before decoding it, so an author's
//     templated model or provider is replaced by today's value of the variable — and saved.
//
// PatchSource edits the frontmatter TEXT in place, located through yaml.v3's node positions: only
// the lines of the edited entries change; every other byte — comments, key order, unknown keys,
// credentials, templated values, quoting style, the body — is copied through unchanged. It never
// renders anything. And it refuses rather than guesses: the result is re-parsed and must (1) pass
// Parse and Spec.Validate, and (2) decode to EXACTLY the original document with the edits applied
// in Go (the self-check), or PatchSource returns an error and no bytes.

// yaml.v3 core-schema tags PatchSource reads and writes.
const (
	yamlTagStr  = "!!str"
	yamlTagNull = "!!null"
)

// SourceEdit is one typed edit PatchSource applies. Build it with a Set… constructor; the zero
// value is refused. The set of paths is closed on purpose: each one is a field a consumer has a
// reason to edit without owning the rest of the document, and each has a Go-side twin the
// self-check can compare against.
type SourceEdit struct {
	path   []string
	value  any
	remove bool
	// apply performs the same edit on a parsed Spec (the self-check's expected value). It sets
	// the leaf and creates absent containers; pruning an emptied container is applied separately,
	// from what the text edit actually pruned, so the two sides cannot disagree about it.
	apply func(*Spec)
	// invalid, when set, is why the constructor refused its arguments (reported by PatchSource).
	invalid string
}

// Path returns the dotted frontmatter path the edit targets (e.g. "tools.allow").
func (e SourceEdit) Path() string { return strings.Join(e.path, ".") }

// SetToolsAllow sets tools.allow. nil REMOVES the key (no narrowing: every tool); an empty
// non-nil slice writes `allow: []` (no tools at all); a list writes exactly those names.
func SetToolsAllow(names []string) SourceEdit {
	cp := copyStrings(names)
	return SourceEdit{
		path: []string{SpecFieldTools, ToolsFieldAllow}, value: cp, remove: cp == nil,
		apply: func(s *Spec) {
			if s.Tools == nil {
				if cp == nil {
					return
				}
				s.Tools = &ToolsConfig{}
			}
			s.Tools.Allow = cp
		},
	}
}

// SetRequirementsResources replaces requirements.resources with res. nil removes the key; an
// empty non-nil slice writes `resources: []`.
func SetRequirementsResources(res []ResourceRequirement) SourceEdit {
	var cp []ResourceRequirement
	if res != nil {
		cp = make([]ResourceRequirement, len(res))
		copy(cp, res)
	}
	return SourceEdit{
		path: []string{SpecFieldRequirements, RequirementsFieldResources}, value: cp, remove: cp == nil,
		apply: func(s *Spec) {
			if s.Requirements == nil {
				if cp == nil {
					return
				}
				s.Requirements = &SpecRequirements{}
			}
			s.Requirements.Resources = cp
		},
	}
}

// SetResourceMode sets requirements.resource_modes.<kind>. An empty mode removes the kind's key,
// so the mode is derived again (ResourceMode). The kind must match ResourceKindPattern and the
// mode must be in the vocabulary; otherwise PatchSource refuses the edit. Whether the mode agrees
// with the resources entries of the kind is judged on the RESULT, so an edit list may change both.
func SetResourceMode(kind string, mode ResourceMode) SourceEdit {
	e := SourceEdit{
		path: []string{SpecFieldRequirements, SpecFieldResourceModes, kind}, value: string(mode), remove: mode == "",
		apply: func(s *Spec) {
			if mode == "" {
				if s.Requirements != nil {
					delete(s.Requirements.ResourceModes, kind)
				}
				return
			}
			if s.Requirements == nil {
				s.Requirements = &SpecRequirements{}
			}
			if s.Requirements.ResourceModes == nil {
				s.Requirements.ResourceModes = map[string]ResourceMode{}
			}
			s.Requirements.ResourceModes[kind] = mode
		},
	}
	switch {
	case !resourceKindRegex.MatchString(kind):
		e.invalid = ErrMsgResourceModeKindForm
	case mode != "" && !isValidResourceMode(mode):
		e.invalid = ErrMsgResourceModeInvalid
	}
	return e
}

// SetSkills replaces skills with skills. nil removes the key.
func SetSkills(skills []SkillRef) SourceEdit {
	var cp []SkillRef
	if skills != nil {
		cp = make([]SkillRef, len(skills))
		copy(cp, skills)
	}
	return SourceEdit{
		path: []string{SpecFieldSkills}, value: cp, remove: cp == nil,
		apply: func(s *Spec) { s.Skills = cp },
	}
}

// SetExecutionProvider sets execution.provider; "" removes the key.
func SetExecutionProvider(provider string) SourceEdit {
	return executionEdit(ExecutionFieldProvider, provider, func(c *execution.Config, v string) { c.Provider = v })
}

// SetExecutionModel sets execution.model; "" removes the key.
func SetExecutionModel(model string) SourceEdit {
	return executionEdit(ExecutionFieldModel, model, func(c *execution.Config, v string) { c.Model = v })
}

// SetExecutionReasoningEffort sets execution.reasoning_effort; "" removes the key.
func SetExecutionReasoningEffort(effort string) SourceEdit {
	return executionEdit(ExecutionFieldReasoningEffort, effort, func(c *execution.Config, v string) { c.ReasoningEffort = v })
}

func executionEdit(field, v string, set func(*execution.Config, string)) SourceEdit {
	return SourceEdit{
		path: []string{SpecFieldExecution, field}, value: v, remove: v == "",
		apply: func(s *Spec) {
			if s.Execution == nil {
				if v == "" {
					return
				}
				s.Execution = &execution.Config{}
			}
			set(s.Execution, v)
		},
	}
}

// SetDisplayName sets display_name; "" removes the key.
func SetDisplayName(name string) SourceEdit {
	return SourceEdit{
		path: []string{SpecFieldDisplayName}, value: name, remove: name == "",
		apply: func(s *Spec) { s.DisplayName = name },
	}
}

func copyStrings(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

// PatchSource applies edits to the frontmatter of an exons document and returns the new document.
// Edits apply in order (a later edit to the same path wins); the result is judged once, at the end.
//
// Preserved byte for byte, outside the edited entries: comments, key order, blank lines, quoting
// and flow style, unknown keys (Extensions), credentials, `{~…~}` templated values (never
// rendered), the delimiters and the body. Inside an edited entry the value is re-emitted by
// yaml.v3 (two-space indentation, at the entry's own column); a comment on the entry's key line
// is kept; comments between the lines of a replaced value are not. Removing the last key of a
// block (tools, requirements, requirements.resource_modes, execution) removes the block, so no
// empty `tools:` is left behind; comments are never removed.
//
// It refuses — returning an error matching ErrPatchRefused and no bytes — when:
//   - the document has no frontmatter, or does not itself pass Parse (ErrPatchSourceInvalid): a
//     frontmatter that is YAML only after its tags are rendered cannot be edited without rendering
//     it, so quote the tag (the engine's documented advice) and it can;
//   - an edit is invalid or meets a shape it will not edit textually — an alias or anchor in the
//     edited entry, a non-mapping where a block is expected (ErrPatchUnsupportedShape);
//   - the result fails Parse or Spec.Validate (ErrPatchResultInvalid, wrapping the reason);
//   - the result does not decode to the original with the edits applied (ErrPatchSelfCheck).
func PatchSource(src []byte, edits ...SourceEdit) ([]byte, error) {
	prefix, fm, suffix, ok := splitFrontmatter(src)
	if !ok {
		return nil, newPatchError(ErrPatchNoFrontmatter, "", nil)
	}
	if _, err := Parse(src); err != nil {
		return nil, newPatchError(ErrPatchSourceInvalid, "", err)
	}

	pruned := make([][]string, len(edits))
	for i, e := range edits {
		if e.apply == nil {
			return nil, newPatchError(ErrPatchEditInvalid, "", errors.New(ErrMsgPatchZeroEdit))
		}
		if e.invalid != "" {
			return nil, newPatchError(ErrPatchEditInvalid, e.Path(), errors.New(e.invalid))
		}
		next, p, err := patchFrontmatter(fm, e)
		if err != nil {
			return nil, err
		}
		fm = next
		pruned[i] = p
	}

	out := make([]byte, 0, len(prefix)+len(fm)+len(suffix))
	out = append(out, prefix...)
	out = append(out, fm...)
	out = append(out, suffix...)

	got, err := Parse(out)
	if err != nil {
		return nil, newPatchError(ErrPatchResultInvalid, "", err)
	}
	want, err := Parse(src)
	if err != nil {
		return nil, newPatchError(ErrPatchSourceInvalid, "", err)
	}
	// In edit order, each edit followed by the pruning ITS text edit performed: a later edit may
	// recreate a block an earlier one emptied.
	for i, e := range edits {
		e.apply(want)
		for _, p := range pruned[i] {
			pruneContainer(want, p)
		}
	}
	if err := sameDocument(want, got); err != nil {
		return nil, newPatchError(ErrPatchSelfCheck, "", err)
	}
	return out, nil
}

// splitFrontmatter cuts src exactly where Parse does: prefix is everything up to and including
// the opening delimiter line, fm the YAML text, suffix everything from the newline before the
// closing delimiter to the end (closing delimiter and body, untouched).
func splitFrontmatter(src []byte) (prefix, fm, suffix []byte, ok bool) {
	content := string(src)
	trimmed := strings.TrimLeft(content, "\xef\xbb\xbf \t")
	if !strings.HasPrefix(trimmed, YAMLFrontmatterDelimiter) {
		return nil, nil, nil, false
	}
	start := len(content) - len(trimmed) + len(YAMLFrontmatterDelimiter)
	rest := content[start:]
	switch {
	case strings.HasPrefix(rest, "\n"):
		start++
	case strings.HasPrefix(rest, "\r\n"):
		start += 2
	}
	closeIdx := strings.Index(content[start:], "\n"+YAMLFrontmatterDelimiter)
	if closeIdx == -1 {
		return nil, nil, nil, false
	}
	end := start + closeIdx
	return src[:start], src[start:end], src[end:], true
}

// fmLines is the frontmatter as lines without their "\n" (a CRLF document keeps each "\r").
type fmLines struct {
	lines []string
	cr    bool
}

func (f *fmLines) String() string { return strings.Join(f.lines, "\n") }

// replace swaps lines [from, to] (1-based, inclusive) for repl.
func (f *fmLines) replace(from, to int, repl []string) {
	out := make([]string, 0, len(f.lines)-(to-from+1)+len(repl))
	out = append(out, f.lines[:from-1]...)
	out = append(out, repl...)
	out = append(out, f.lines[to:]...)
	f.lines = out
}

// insertAfter inserts repl after line at (1-based; 0 inserts at the top).
func (f *fmLines) insertAfter(at int, repl []string) { f.replace(at+1, at, repl) }

// pathFrame is one level of the walk: the mapping holding path[level], the index of that key's
// entry in it (or -1), and the column its keys start at.
type pathFrame struct {
	m      *yaml.Node
	entry  int
	indent int
}

// patchFrontmatter applies one edit to the frontmatter text. It returns the new text and the
// dotted paths of the containers it removed because the edit emptied them.
func patchFrontmatter(fm []byte, e SourceEdit) ([]byte, []string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(fm, &doc); err != nil {
		return nil, nil, newPatchError(ErrPatchSourceInvalid, e.Path(), err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || !isBlockMapping(doc.Content[0]) {
		return nil, nil, newPatchError(ErrPatchUnsupportedShape, e.Path(), errors.New(ErrMsgPatchRootNotBlock))
	}
	text := &fmLines{lines: strings.Split(string(fm), "\n"), cr: bytes.Contains(fm, []byte("\r\n"))}

	var leaf *yaml.Node
	if !e.remove {
		leaf = &yaml.Node{}
		if err := leaf.Encode(e.value); err != nil {
			return nil, nil, newPatchError(ErrPatchEditInvalid, e.Path(), err)
		}
	}

	frames := make([]pathFrame, 0, len(e.path))
	cur := doc.Content[0]
	if len(cur.Content) == 0 {
		return nil, nil, newPatchError(ErrPatchUnsupportedShape, e.Path(), errors.New(ErrMsgPatchRootNotBlock))
	}
	for i, key := range e.path {
		indent := cur.Content[0].Column - 1
		j := findKey(cur, key)
		frames = append(frames, pathFrame{m: cur, entry: j, indent: indent})

		if j < 0 { // absent: insert the rest of the path under this mapping
			if e.remove {
				return fm, nil, nil
			}
			lines, err := renderEntry(key, nestValue(e.path[i+1:], leaf), indent, "", text.cr)
			if err != nil {
				return nil, nil, newPatchError(ErrPatchEditInvalid, e.Path(), err)
			}
			_, end := entryLines(cur, len(cur.Content)/2-1, indent, text.lines)
			text.insertAfter(end, lines)
			return []byte(text.String()), nil, nil
		}

		keyNode, val := cur.Content[2*j], cur.Content[2*j+1]
		if i == len(e.path)-1 { // the leaf entry itself
			if hasAnchorOrAlias(val) {
				return nil, nil, newPatchError(ErrPatchUnsupportedShape, e.Path(), errors.New(ErrMsgPatchAnchor))
			}
			if e.remove {
				return removeEntry(text, frames, e.path)
			}
			start, end := entryLines(cur, j, indent, text.lines)
			comment := keyNode.LineComment
			if comment == "" && start == end {
				comment = val.LineComment
			}
			lines, err := renderEntry(key, leaf, indent, comment, text.cr)
			if err != nil {
				return nil, nil, newPatchError(ErrPatchEditInvalid, e.Path(), err)
			}
			text.replace(start, end, lines)
			return []byte(text.String()), nil, nil
		}

		switch {
		case isBlockMapping(val) && len(val.Content) > 0:
			cur = val // descend
		case isNull(val):
			if e.remove {
				return fm, nil, nil
			}
			start, end := entryLines(cur, j, indent, text.lines)
			lines, err := renderEntry(key, nestValue(e.path[i+1:], leaf), indent, keyNode.LineComment, text.cr)
			if err != nil {
				return nil, nil, newPatchError(ErrPatchEditInvalid, e.Path(), err)
			}
			text.replace(start, end, lines)
			return []byte(text.String()), nil, nil
		case val.Kind == yaml.MappingNode: // flow style ({…}) — re-emit this one entry, still flow
			if hasAnchorOrAlias(val) {
				return nil, nil, newPatchError(ErrPatchUnsupportedShape, e.Path(), errors.New(ErrMsgPatchAnchor))
			}
			if err := setInNode(val, e.path[i+1:], leaf); err != nil {
				return nil, nil, newPatchError(ErrPatchUnsupportedShape, e.Path(), err)
			}
			start, end := entryLines(cur, j, indent, text.lines)
			lines, err := renderEntry(key, val, indent, keyNode.LineComment, text.cr)
			if err != nil {
				return nil, nil, newPatchError(ErrPatchEditInvalid, e.Path(), err)
			}
			text.replace(start, end, lines)
			return []byte(text.String()), nil, nil
		default:
			return nil, nil, newPatchError(ErrPatchUnsupportedShape, e.Path(), errors.New(ErrMsgPatchNotMapping+": "+strings.Join(e.path[:i+1], ".")))
		}
	}
	return fm, nil, nil // unreachable: every path has at least one element
}

// removeEntry deletes the leaf entry — or, when that leaves its block empty, the nearest
// enclosing block that would otherwise be left empty (never the root). It returns the dotted
// paths of the blocks it removed with it.
func removeEntry(text *fmLines, frames []pathFrame, path []string) ([]byte, []string, error) {
	level := len(frames) - 1
	var pruned []string
	for level > 0 && len(frames[level].m.Content) == 2 {
		pruned = append(pruned, strings.Join(path[:level], "."))
		level--
	}
	f := frames[level]
	if hasAnchorOrAlias(f.m.Content[2*f.entry+1]) {
		return nil, nil, newPatchError(ErrPatchUnsupportedShape, strings.Join(path, "."), errors.New(ErrMsgPatchAnchor))
	}
	start, end := entryLines(f.m, f.entry, f.indent, text.lines)
	text.replace(start, end, nil)
	return []byte(text.String()), pruned, nil
}

// entryLines returns the 1-based, inclusive line range of entry j of block mapping m: from its
// key's line to its value's last content line. Blank lines and comment-only lines after the value
// are NOT included (they belong to what follows, or to nobody), so a replacement keeps them.
//
// The value ends at the first content line indented less than the key, or at the key's own
// indentation unless that line continues a compact block sequence (`key:\n- a`).
func entryLines(m *yaml.Node, j, indent int, lines []string) (int, int) {
	key, val := m.Content[2*j], m.Content[2*j+1]
	start := key.Line
	end := start
	compactSeq := val.Kind == yaml.SequenceNode && val.Style&yaml.FlowStyle == 0
	for n := start + 1; n <= len(lines); n++ {
		line := lines[n-1]
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		ind := len(line) - len(strings.TrimLeft(line, " "))
		if ind < indent {
			break
		}
		if ind == indent {
			if compactSeq && (t == "-" || strings.HasPrefix(t, "- ")) {
				end = n
				continue
			}
			break
		}
		end = n
	}
	return start, end
}

// renderEntry renders `key: value` with yaml.v3 at two-space indentation, shifted to indent, and
// appends comment (if any) to the first line — where the original entry's key-line comment sat.
func renderEntry(key string, val *yaml.Node, indent int, comment string, cr bool) ([]string, error) {
	root := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: yamlTagStr, Value: key}, val,
	}}
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
	pad := strings.Repeat(" ", indent)
	for i, l := range lines {
		if i == 0 && comment != "" && !strings.Contains(l, comment) {
			l += " " + comment
		}
		if l != "" {
			l = pad + l
		}
		if cr {
			l += "\r"
		}
		lines[i] = l
	}
	return lines, nil
}

// nestValue wraps leaf in one block mapping per remaining path element.
func nestValue(rest []string, leaf *yaml.Node) *yaml.Node {
	for i := len(rest) - 1; i >= 0; i-- {
		leaf = &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: yamlTagStr, Value: rest[i]}, leaf,
		}}
	}
	return leaf
}

// setInNode sets (leaf != nil) or removes (leaf == nil) path inside mapping node m, creating
// mappings for absent or null intermediates.
func setInNode(m *yaml.Node, path []string, leaf *yaml.Node) error {
	for i, key := range path {
		j := findKey(m, key)
		last := i == len(path)-1
		switch {
		case j < 0 && leaf == nil:
			return nil
		case j < 0:
			m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: yamlTagStr, Value: key}, nestValue(path[i+1:], leaf))
			return nil
		case last && leaf == nil:
			m.Content = append(m.Content[:2*j], m.Content[2*j+2:]...)
			return nil
		case last:
			m.Content[2*j+1] = leaf
			return nil
		}
		next := m.Content[2*j+1]
		if isNull(next) {
			if leaf == nil {
				return nil
			}
			m.Content[2*j+1] = nestValue(path[i+1:], leaf)
			return nil
		}
		if next.Kind != yaml.MappingNode {
			return errors.New(ErrMsgPatchNotMapping + ": " + strings.Join(path[:i+1], "."))
		}
		m = next
	}
	return nil
}

func findKey(m *yaml.Node, key string) int {
	for j := 0; j+1 < len(m.Content); j += 2 {
		if m.Content[j].Kind == yaml.ScalarNode && m.Content[j].Value == key {
			return j / 2
		}
	}
	return -1
}

func isBlockMapping(n *yaml.Node) bool {
	return n != nil && n.Kind == yaml.MappingNode && n.Style&yaml.FlowStyle == 0
}

func isNull(n *yaml.Node) bool {
	return n.Kind == yaml.ScalarNode && n.Tag == yamlTagNull
}

func hasAnchorOrAlias(n *yaml.Node) bool {
	if n == nil {
		return false
	}
	if n.Anchor != "" || n.Kind == yaml.AliasNode {
		return true
	}
	for _, c := range n.Content {
		if hasAnchorOrAlias(c) {
			return true
		}
	}
	return false
}

// pruneContainer mirrors, on the Go side, a block the text edit removed because it was emptied.
func pruneContainer(s *Spec, path string) {
	switch path {
	case SpecFieldTools:
		s.Tools = nil
	case SpecFieldRequirements:
		s.Requirements = nil
	case SpecFieldRequirements + "." + SpecFieldResourceModes:
		if s.Requirements != nil {
			s.Requirements.ResourceModes = nil
		}
	case SpecFieldExecution:
		s.Execution = nil
	}
}

// sameDocument compares two specs by their canonical YAML encoding (struct tags, sorted maps; the
// allow-lists keep nil apart from empty through their own marshallers) and their bodies.
//
// ⚠ On a mismatch it names only the TOP-LEVEL KEYS that differ, never their values: a document can
// carry anything an author wrote (context, extensions, credential refs), and this error travels to
// logs and API responses.
func sameDocument(want, got *Spec) error {
	w, err := yaml.Marshal(want)
	if err != nil {
		return err
	}
	g, err := yaml.Marshal(got)
	if err != nil {
		return err
	}
	if bytes.Equal(w, g) && want.Body == got.Body {
		return nil
	}
	var wm, gm map[string]any
	_ = yaml.Unmarshal(w, &wm)
	_ = yaml.Unmarshal(g, &gm)
	keys := map[string]struct{}{}
	for k := range wm {
		keys[k] = struct{}{}
	}
	for k := range gm {
		keys[k] = struct{}{}
	}
	var differ []string
	for k := range keys {
		wb, _ := yaml.Marshal(wm[k])
		gb, _ := yaml.Marshal(gm[k])
		if !bytes.Equal(wb, gb) {
			differ = append(differ, k)
		}
	}
	if want.Body != got.Body {
		differ = append(differ, "(body)")
	}
	sort.Strings(differ)
	return errors.New(ErrMsgPatchSelfCheckDetail + ": " + strings.Join(differ, ", "))
}

// Patch refusal sentinels. Every PatchSource error matches ErrPatchRefused and exactly one of the
// specific sentinels below (errors.Is), and keeps its cause in the unwrap chain.
var (
	ErrPatchRefused          = errors.New(ErrMsgPatchRefused)
	ErrPatchNoFrontmatter    = errors.New(ErrMsgPatchNoFrontmatter)
	ErrPatchSourceInvalid    = errors.New(ErrMsgPatchSourceInvalid)
	ErrPatchEditInvalid      = errors.New(ErrMsgPatchEditInvalid)
	ErrPatchUnsupportedShape = errors.New(ErrMsgPatchUnsupportedShape)
	ErrPatchResultInvalid    = errors.New(ErrMsgPatchResultInvalid)
	ErrPatchSelfCheck        = errors.New(ErrMsgPatchSelfCheck)
)

// PatchError is the cause inside every PatchSource refusal: which rule refused (Reason, one of the
// ErrPatch… sentinels), the edit path when one is to blame, and the underlying error.
type PatchError struct {
	Reason error
	Path   string
	Cause  error
}

func (e *PatchError) Error() string {
	msg := e.Reason.Error()
	if e.Path != "" {
		msg += " (" + e.Path + ")"
	}
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

// Is matches ErrPatchRefused and the specific Reason.
func (e *PatchError) Is(target error) bool {
	return target == ErrPatchRefused || target == e.Reason
}

// Unwrap returns the underlying cause (a validation error for ErrPatchResultInvalid).
func (e *PatchError) Unwrap() error { return e.Cause }

func newPatchError(reason error, path string, cause error) error {
	pe := &PatchError{Reason: reason, Path: path, Cause: cause}
	err := cuserr.WrapWithCustomError(pe, cuserr.ErrorCategoryValidation, ErrCodePatch, ErrMsgPatchRefused)
	if path != "" {
		return err.WithMetadata(MetaKeyPath, path)
	}
	return err
}
