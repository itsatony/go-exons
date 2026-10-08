package exons

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/itsatony/go-cuserr"
	"gopkg.in/yaml.v3"
)

// A frontmatter YAML error names the line, the key and — where it is unambiguous — the
// fix (v0.44.0, vAudience/aigentverse#258).
//
// yaml.v3 reports a line RELATIVE TO THE TEXT IT WAS GIVEN, which for Parse is the
// frontmatter body: the document's line 1 is `---`, so yaml's "line 3" is the
// document's line 4. Its messages also name no key, and its most common authoring
// failure — a prose value containing ": " (`description: … Einzelmeldungen: Aktuelles`)
// — reads "mapping values are not allowed in this context", which tells an author (or
// an authoring agent) nothing about what to change. NewFrontmatterParseErrorAt adds:
//
//   - the DOCUMENT line (and, when it differs, yaml's own frontmatter line, so the
//     unchanged yaml text in the cause still reconciles);
//   - the dotted path of the nearest key at or above that line (a line scan of the
//     frontmatter text — no second YAML parse, which would fail the same way);
//   - for the ": " case only, the fix: quote the value or make it a block scalar.
//
// ⚠ WHAT STAYS STABLE: the error code (ErrCodeConfig), the message PREFIX
// (ErrMsgFrontmatterParse — consumers match on it) and the cause, which is the
// unmodified yaml.v3 error (errors.As to *yaml.TypeError still works). The location
// sits between the prefix and the cause. An error with no line in it (yaml.v3 has a
// few, e.g. invalid UTF-8) is wrapped exactly as NewFrontmatterParseError wraps it.
//
// The key scan is a heuristic over text that failed to parse, so it is advisory: it
// reads block-style `key:` lines and sequence items, and it can be fooled by a
// key-looking line inside a block scalar. It never invents a key — a line it cannot
// attribute yields the line alone.

// frontmatterKeyLineRe matches a block-mapping key line: indentation, any sequence
// dashes, the key (double-quoted, single-quoted or plain) and the `:` value indicator
// followed by a space or the end of the line. Group 4 is the rest of the line.
var frontmatterKeyLineRe = regexp.MustCompile(
	`^( *)((?:- +)*)("(?:[^"\\]|\\.)*"|'(?:[^']|'')*'|[^\s"'#\[\]{},&*!|>%@` + "`" + `-][^#]*?) *:(?:[ \t]+(.*)|$)`)

// yamlErrorLineRe extracts yaml.v3's "line N: message" (after its "yaml: " prefix for a
// syntax error; as-is for a TypeError entry).
var yamlErrorLineRe = regexp.MustCompile(`^line (\d+): (.*)$`)

// yamlErrorPrefix is the prefix yaml.v3 puts on a syntax error.
const yamlErrorPrefix = "yaml: "

// NewFrontmatterParseErrorAt is NewFrontmatterParseError with the source it failed on:
// frontmatter is the exact text handed to yaml.v3 and firstLine the DOCUMENT line that
// text begins on (2 for Parse's usual `---\n` opening, 1 for bare YAML). The result
// carries MetaKeyLine (document line), MetaKeyFrontmatterLine (yaml's line) and, when a
// key was found, MetaKeyFrontmatterKey. With no line in the cause it returns exactly
// NewFrontmatterParseError(cause).
func NewFrontmatterParseErrorAt(cause error, frontmatter string, firstLine int) error {
	fmLine, yamlMsg, ok := yamlErrorLine(cause)
	if !ok {
		return NewFrontmatterParseError(cause)
	}
	if firstLine < 1 {
		firstLine = 1
	}
	docLine := fmLine + firstLine - 1

	var msg strings.Builder
	msg.WriteString(ErrMsgFrontmatterParse)
	msg.WriteString(": ")
	fmt.Fprintf(&msg, ErrFmtFrontmatterAtLine, docLine)
	if docLine != fmLine {
		fmt.Fprintf(&msg, ErrFmtFrontmatterYAMLLine, fmLine)
	}

	unquotedColon := strings.HasPrefix(yamlMsg, yamlMsgMappingValues)
	path, leaf, keyLine := frontmatterKeyAt(frontmatter, fmLine, unquotedColon)
	if path != "" {
		fmt.Fprintf(&msg, ErrFmtFrontmatterAtKey, path)
		if unquotedColon {
			msg.WriteString(": ")
			fmt.Fprintf(&msg, ErrFmtFrontmatterUnquotedColon, leaf, leaf)
			if keyLine != fmLine {
				fmt.Fprintf(&msg, ErrFmtFrontmatterIndentHint, docLine)
			}
		}
	}

	err := cuserr.WrapStdError(cause, ErrCodeConfig, msg.String()).
		WithMetadata(MetaKeyLine, strconv.Itoa(docLine)).
		WithMetadata(MetaKeyFrontmatterLine, strconv.Itoa(fmLine))
	if path != "" {
		err = err.WithMetadata(MetaKeyFrontmatterKey, path)
	}
	return err
}

// yamlErrorLine returns the first line yaml.v3 names in err and the message after it.
func yamlErrorLine(err error) (int, string, bool) {
	if err == nil {
		return 0, "", false
	}
	text := strings.TrimPrefix(err.Error(), yamlErrorPrefix)
	var te *yaml.TypeError
	if errors.As(err, &te) {
		if len(te.Errors) == 0 {
			return 0, "", false
		}
		text = te.Errors[0]
	}
	m := yamlErrorLineRe.FindStringSubmatch(text)
	if m == nil {
		return 0, "", false
	}
	n, convErr := strconv.Atoi(m[1])
	if convErr != nil || n < 1 {
		return 0, "", false
	}
	return n, m[2], true
}

// frontmatterLine is one scanned line of frontmatter text.
type frontmatterLine struct {
	key       string // unquoted key, "" when the line is not a key line
	keyIndent int    // column of the key (after indentation and sequence dashes)
	rest      string // the text after `key:`
	lead      int    // leading spaces
	dash      bool   // the line is a sequence item
	skip      bool   // blank or comment-only
}

func scanFrontmatterLine(s string) frontmatterLine {
	trimmed := strings.TrimLeft(s, " ")
	l := frontmatterLine{lead: len(s) - len(trimmed)}
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		l.skip = true
		return l
	}
	l.dash = strings.HasPrefix(trimmed, "-") && (len(trimmed) == 1 || trimmed[1] == ' ')
	m := frontmatterKeyLineRe.FindStringSubmatch(s)
	if m == nil {
		return l
	}
	l.key = unquoteFrontmatterKey(m[3])
	l.keyIndent = len(m[1]) + len(m[2])
	l.rest = m[4]
	return l
}

// unquoteFrontmatterKey strips a quoted key's quotes for display.
func unquoteFrontmatterKey(k string) string {
	if len(k) >= 2 {
		switch {
		case k[0] == '"' && k[len(k)-1] == '"':
			if u, err := strconv.Unquote(k); err == nil {
				return u
			}
		case k[0] == '\'' && k[len(k)-1] == '\'':
			return strings.ReplaceAll(k[1:len(k)-1], "''", "'")
		}
	}
	return k
}

// restHasValueIndicator reports whether a key line's value itself carries a ": " (or
// ends in ":"), i.e. whether the offending indicator is on the key's own line.
func restHasValueIndicator(rest string) bool {
	rest = strings.TrimRight(rest, " \t")
	return strings.Contains(rest, ": ") || strings.HasSuffix(rest, ":")
}

// frontmatterKeyAt finds the key a yaml error at frontmatter line `line` belongs to.
// It returns the dotted path, the leaf key, and the (1-based) line the leaf key is on.
//
// The line's own key is the answer unless the error is "mapping values" and the key's
// value carries no further ": " — then the ": " that failed IS this line's, so the line
// is the continuation of a plain scalar that began on a less-indented key above, and
// that key is the one whose value needs quoting.
func frontmatterKeyAt(frontmatter string, line int, unquotedColon bool) (path, leaf string, keyLine int) {
	lines := strings.Split(strings.ReplaceAll(frontmatter, "\r\n", "\n"), "\n")
	if line < 1 || line > len(lines) {
		return "", "", 0
	}
	scanned := make([]frontmatterLine, line)
	for i := 0; i < line; i++ {
		scanned[i] = scanFrontmatterLine(lines[i])
	}

	at := line - 1
	cur := scanned[at]
	keyIdx := -1
	if cur.key != "" && (!unquotedColon || restHasValueIndicator(cur.rest)) {
		keyIdx = at
	} else if !cur.skip {
		// The owner is the nearest key above that is less indented than this line; a
		// sequence item may sit at its parent key's own indentation.
		for i := at - 1; i >= 0; i-- {
			l := scanned[i]
			if l.key == "" {
				continue
			}
			if l.keyIndent < cur.lead || (cur.dash && l.keyIndent <= cur.lead) {
				keyIdx = i
				break
			}
		}
	}
	if keyIdx < 0 {
		return "", "", 0
	}

	parts := []string{scanned[keyIdx].key}
	indent := scanned[keyIdx].keyIndent
	for i := keyIdx - 1; i >= 0 && indent > 0; i-- {
		l := scanned[i]
		if l.key != "" && l.keyIndent < indent {
			parts = append([]string{l.key}, parts...)
			indent = l.keyIndent
		}
	}
	return strings.Join(parts, "."), scanned[keyIdx].key, keyIdx + 1
}
