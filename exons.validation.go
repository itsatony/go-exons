package exons

import (
	"strings"

	"github.com/itsatony/go-exons/internal"
)

// Validation error/warning message constants are declared in exons.errors.go.
// Position is declared in exons.errors.go.

// ValidationResult contains the results of template validation.
type ValidationResult struct {
	issues []ValidationIssue
}

// ValidationIssue represents a single validation finding.
type ValidationIssue struct {
	Severity ValidationSeverity
	Message  string
	Position Position
	TagName  string
}

// Issues returns all validation issues found.
func (r *ValidationResult) Issues() []ValidationIssue {
	return r.issues
}

// Errors returns only issues with error severity.
func (r *ValidationResult) Errors() []ValidationIssue {
	var errors []ValidationIssue
	for _, issue := range r.issues {
		if issue.Severity == SeverityError {
			errors = append(errors, issue)
		}
	}
	return errors
}

// Warnings returns only issues with warning severity.
func (r *ValidationResult) Warnings() []ValidationIssue {
	var warnings []ValidationIssue
	for _, issue := range r.issues {
		if issue.Severity == SeverityWarning {
			warnings = append(warnings, issue)
		}
	}
	return warnings
}

// HasErrors returns true if there are any error-severity issues.
func (r *ValidationResult) HasErrors() bool {
	for _, issue := range r.issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if there are any warning-severity issues.
func (r *ValidationResult) HasWarnings() bool {
	for _, issue := range r.issues {
		if issue.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

// IsValid returns true if there are no error-severity issues.
func (r *ValidationResult) IsValid() bool {
	return !r.HasErrors()
}

// Validate parses and validates a template without executing it.
// It returns validation results containing any issues found.
// Parse errors are returned as validation errors with SeverityError.
func (e *Engine) Validate(source string) (*ValidationResult, error) {
	result := &ValidationResult{
		issues: make([]ValidationIssue, 0),
	}

	// Markdown fence lints run before tokenization so they surface even when
	// the template fails to parse.
	if e.config.markdownFences {
		e.validateMarkdownFences(source, result)
	}

	// Create lexer with configured delimiters
	lexerConfig := internal.LexerConfig{
		OpenDelim:      e.config.openDelim,
		CloseDelim:     e.config.closeDelim,
		MarkdownFences: e.config.markdownFences,
	}
	lexer := internal.NewLexerWithConfig(source, lexerConfig, e.logger)

	// Tokenize
	tokens, tokenErr := lexer.Tokenize()
	if tokenErr != nil {
		result.issues = append(result.issues, ValidationIssue{
			Severity: SeverityError,
			Message:  ErrMsgParseFailed + ": " + tokenErr.Error(),
			Position: Position{},
		})
		return result, nil //nolint:nilerr // intentional: collect parse errors as validation issues
	}

	// Parse with source for validation
	parser := internal.NewParserWithSource(tokens, source, lexerConfig, e.logger)
	ast, parseErr := parser.Parse()
	if parseErr != nil {
		result.issues = append(result.issues, ValidationIssue{
			Severity: SeverityError,
			Message:  ErrMsgParseFailed + ": " + parseErr.Error(),
			Position: Position{},
		})
		return result, nil //nolint:nilerr // intentional: collect parse errors as validation issues
	}

	// Validate AST nodes
	e.validateNodes(ast.Children, result)

	return result, nil
}

// validateMarkdownFences lints markdown code fences (WithMarkdownFences mode):
// tag-like syntax inside an inert fence will not render, and an unclosed
// fence silently inerts everything to the end of input.
func (e *Engine) validateMarkdownFences(source string, result *ValidationResult) {
	for _, region := range internal.ScanMarkdownFences(source) {
		pos := Position{
			Offset: region.OpenPos.Offset,
			Line:   region.OpenPos.Line,
			Column: region.OpenPos.Column,
		}
		if !region.Live && strings.Contains(source[region.BodyStart:region.End], e.config.openDelim) {
			result.issues = append(result.issues, ValidationIssue{
				Severity: SeverityWarning,
				Message:  WarnMsgTagLikeInInertFence,
				Position: pos,
			})
		}
		if region.Unclosed {
			result.issues = append(result.issues, ValidationIssue{
				Severity: SeverityWarning,
				Message:  WarnMsgUnclosedMarkdownFence,
				Position: pos,
			})
		}
	}
}

// validateNodes recursively validates a slice of AST nodes.
func (e *Engine) validateNodes(nodes []internal.Node, result *ValidationResult) {
	for _, node := range nodes {
		e.validateNode(node, result)
	}
}

// validateNode validates a single AST node.
func (e *Engine) validateNode(node internal.Node, result *ValidationResult) {
	switch n := node.(type) {
	case *internal.TextNode:
		// Text nodes are always valid
		return

	case *internal.TagNode:
		e.validateTagNode(n, result)

	case *internal.ConditionalNode:
		e.validateConditionalNode(n, result)

	case *internal.ForNode:
		e.validateForNode(n, result)

	case *internal.SwitchNode:
		e.validateSwitchNode(n, result)
	}
}

// validateTagNode validates a tag node.
func (e *Engine) validateTagNode(tag *internal.TagNode, result *ValidationResult) {
	// Skip raw blocks - they're always valid
	if tag.IsRaw() {
		return
	}

	// Check if tag has a registered resolver
	if !e.registry.Has(tag.Name) {
		result.issues = append(result.issues, ValidationIssue{
			Severity: SeverityWarning,
			Message:  ErrMsgUnknownTagInTemplate,
			Position: e.internalPosToPublic(tag.Pos()),
			TagName:  tag.Name,
		})
	} else {
		// Validate using the resolver's Validate method
		resolver, _ := e.registry.Get(tag.Name)
		if err := resolver.Validate(tag.Attributes); err != nil {
			result.issues = append(result.issues, ValidationIssue{
				Severity: SeverityError,
				Message:  err.Error(),
				Position: e.internalPosToPublic(tag.Pos()),
				TagName:  tag.Name,
			})
		}
	}

	e.validateOnErrorAttr(tag.Attributes, tag.Pos(), tag.Name, result)

	// Validate exons.include references
	if tag.Name == TagNameInclude {
		if templateName, hasTemplate := tag.Attributes.Get(AttrTemplate); hasTemplate {
			if !e.HasTemplate(templateName) {
				result.issues = append(result.issues, ValidationIssue{
					Severity: SeverityWarning,
					Message:  ErrMsgMissingIncludeTarget,
					Position: e.internalPosToPublic(tag.Pos()),
					TagName:  tag.Name,
				})
			}
		}
	}

	// Validate children recursively
	if len(tag.Children) > 0 {
		e.validateNodes(tag.Children, result)
	}
}

// validateOnErrorAttr reports an onerror= whose value is not one of the declared strategies.
//
// It is shared by every attribute-bearing shape — the tag, the three block constructs, and the
// individual elseif/case branches — because as of v0.24.0 all of them HONOUR onerror=, and a lint
// that checks it on only some of them is worse than one that checks it nowhere: an author reads
// silence as approval. The trap is specific and quiet: getErrorStrategy returns as soon as the key
// is PRESENT, and ParseErrorStrategy maps anything unrecognised to throw. So onerror="remov" on a
// for-loop does not fall back to the context's lenient default — it hard-fails, under a renderer
// configured never to hard-fail, and the typo is the only evidence.
func (e *Engine) validateOnErrorAttr(attrs internal.Attributes, pos internal.Position, tagName string, result *ValidationResult) {
	onErrorStr, hasOnError := attrs.Get(AttrOnError)
	if !hasOnError || IsValidErrorStrategy(onErrorStr) {
		return
	}
	result.issues = append(result.issues, ValidationIssue{
		Severity: SeverityError,
		Message:  ErrMsgInvalidOnErrorAttr,
		Position: e.internalPosToPublic(pos),
		TagName:  tagName,
	})
}

// validateConditionalNode validates a conditional node.
func (e *Engine) validateConditionalNode(cond *internal.ConditionalNode, result *ValidationResult) {
	e.validateOnErrorAttr(cond.Attributes, cond.Pos(), TagNameIf, result)
	for _, branch := range cond.Branches {
		// A branch's own attributes, at the branch's own position — an elseif's typo must not be
		// reported at the line of the opening exons.if.
		e.validateOnErrorAttr(branch.Attributes, branch.Pos, TagNameIf, result)
		// Validate branch children recursively
		e.validateNodes(branch.Children, result)
	}
}

// validateForNode validates a for loop node.
func (e *Engine) validateForNode(forNode *internal.ForNode, result *ValidationResult) {
	e.validateOnErrorAttr(forNode.Attributes, forNode.Pos(), TagNameFor, result)

	// Check required item variable
	if forNode.ItemVar == "" {
		result.issues = append(result.issues, ValidationIssue{
			Severity: SeverityError,
			Message:  ErrMsgForMissingItem,
			Position: e.internalPosToPublic(forNode.Pos()),
			TagName:  TagNameFor,
		})
	}

	// Check required source path
	if forNode.Source == "" {
		result.issues = append(result.issues, ValidationIssue{
			Severity: SeverityError,
			Message:  ErrMsgForMissingIn,
			Position: e.internalPosToPublic(forNode.Pos()),
			TagName:  TagNameFor,
		})
	}

	// Check for negative limit
	if forNode.Limit < 0 {
		result.issues = append(result.issues, ValidationIssue{
			Severity: SeverityError,
			Message:  ErrMsgForInvalidLimit,
			Position: e.internalPosToPublic(forNode.Pos()),
			TagName:  TagNameFor,
		})
	}

	// Validate children recursively
	e.validateNodes(forNode.Children, result)
}

// validateSwitchNode validates a switch/case node.
func (e *Engine) validateSwitchNode(switchNode *internal.SwitchNode, result *ValidationResult) {
	e.validateOnErrorAttr(switchNode.Attributes, switchNode.Pos(), TagNameSwitch, result)

	// Check required expression
	if switchNode.Expression == "" {
		result.issues = append(result.issues, ValidationIssue{
			Severity: SeverityError,
			Message:  ErrMsgSwitchMissingEval,
			Position: e.internalPosToPublic(switchNode.Pos()),
			TagName:  TagNameSwitch,
		})
	}

	// Validate each case
	for _, caseNode := range switchNode.Cases {
		e.validateOnErrorAttr(caseNode.Attributes, caseNode.Pos, TagNameCase, result)

		// Check that case has either value or eval
		if caseNode.Value == "" && caseNode.Eval == "" {
			result.issues = append(result.issues, ValidationIssue{
				Severity: SeverityError,
				Message:  ErrMsgSwitchMissingValue,
				Position: e.internalPosToPublic(caseNode.Pos),
				TagName:  TagNameCase,
			})
		}

		// Validate case children recursively
		e.validateNodes(caseNode.Children, result)
	}

	// Validate default case if present
	if switchNode.Default != nil {
		e.validateNodes(switchNode.Default.Children, result)
	}
}

// internalPosToPublic converts internal Position to public Position.
func (e *Engine) internalPosToPublic(pos internal.Position) Position {
	return Position{
		Offset: pos.Offset,
		Line:   pos.Line,
		Column: pos.Column,
	}
}
