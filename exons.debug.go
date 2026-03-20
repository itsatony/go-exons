package exons

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/itsatony/go-exons/internal"
)

// DryRun output format constants.
const (
	dryRunHeader             = "=== Dry Run Result ===\n"
	dryRunValidLabel         = "Valid: %v\n"
	dryRunVariablesHeader    = "\nVariables (%d):\n"
	dryRunResolversHeader    = "\nResolvers (%d):\n"
	dryRunIncludesHeader     = "\nIncludes (%d):\n"
	dryRunConditionalsHeader = "\nConditionals (%d):\n"
	dryRunLoopsHeader        = "\nLoops (%d):\n"
	dryRunMissingVarsHeader  = "\nMissing Variables (%d):\n"
	dryRunUnusedVarsHeader   = "\nUnused Variables (%d):\n"
	dryRunErrorsHeader       = "\nErrors (%d):\n"
	dryRunWarningsHeader     = "\nWarnings (%d):\n"
	dryRunOutputHeader       = "\n=== Placeholder Output ===\n"
	dryRunListPrefix         = "  - "
	dryRunNewline            = "\n"
)

// DryRun variable status labels.
const (
	dryRunStatusFound      = "found"
	dryRunStatusMissing    = "MISSING"
	dryRunStatusNotFound   = "NOT FOUND"
	dryRunStatusDefault    = "not found (default: %q)"
	dryRunStatusVarLine    = "  - %s [line %d]: %s\n"
	dryRunStatusSuggestion = "    Did you mean: %s?\n"
	dryRunStatusResSummary = "  - %s [line %d]\n"
	dryRunStatusIncLine    = "  - %s [line %d]: %s\n"
	dryRunStatusCondLine   = "  - %s [line %d]\n"
	dryRunStatusLoopFound  = "source found"
	dryRunStatusLoopMiss   = "source NOT FOUND"
	dryRunStatusLoopLine   = "  - for %s in %s [line %d]: %s\n"
)

// DryRun warning format strings.
const (
	dryRunWarnInclude = "line %d: included template '%s' not found"
	dryRunWarnLoopSrc = "line %d: loop source '%s' not found in data"
)

// Placeholder format strings for dry-run output.
const (
	placeholderVar         = "{{%s}}"
	placeholderInclude     = "{{include:%s}}"
	placeholderTag         = "{{%s}}"
	placeholderIfOpen      = "{{if:%s}}"
	placeholderElseIf      = "{{elseif:%s}}"
	placeholderElse        = "{{else}}"
	placeholderIfClose     = "{{/if}}"
	placeholderForOpen     = "{{for:%s in %s}}"
	placeholderForClose    = "{{/for}}"
	placeholderSwitchOpen  = "{{switch:%s}}"
	placeholderCaseVal     = "{{case:%s}}"
	placeholderCaseEval    = "{{case eval:%s}}"
	placeholderDefault     = "{{default}}"
	placeholderSwitchClose = "{{/switch}}"
)

// Explain output format constants.
const (
	explainHeader       = "=== Template Explanation ===\n"
	explainASTHeader    = "\n--- AST Structure ---\n"
	explainVarsHeader   = "\n--- Variable Accesses ---\n"
	explainTimingHeader = "\n--- Timing ---\n"
	explainErrorHeader  = "\n--- Error ---\n  %v\n"
	explainOutputHeader = "\n--- Output ---\n"
	explainTimingTotal  = "  Total: %v\n"
	explainTimingExec   = "  Execution: %v\n"
	explainVarNotFound  = "NOT FOUND"
	explainVarDefault   = "not found, using default: %q"
	explainVarValue     = "= %v"
	explainVarLine      = "  [line %d] %s: %s\n"
)

// AST format constants.
const (
	astIndentUnit = "  "
	astRootLabel  = "%sRoot\n"
	astTextLabel  = "%sText: %q\n"
	astTagLabel   = "%sTag: %s"
	astAttrsFmt   = " [%s]"
	astAttrPair   = "%s=%q"
	astLineFmt    = " (line %d)\n"
	astCondFmt    = "%sConditional: %s (line %d)\n"
	astThenLabel  = "%s  Then:\n"
	astElseLabel  = "%s  Else:\n"
	astElseIfFmt  = "%s  ElseIf: %s\n"
	astForFmt     = "%sFor: %s in %s"
	astForIndex   = " (index: %s)"
	astForLimit   = " (limit: %d)"
	astSwitchFmt  = "%sSwitch: %s (line %d)\n"
	astCaseVal    = "%s  Case: %s\n"
	astCaseEval   = "%s  Case eval: %s\n"
	astDefaultLbl = "%s  Default:\n"
)

// AST text truncation constants.
const (
	astTextMaxLen     = 40
	astTextTruncation = "..."
	astNewlineEscape  = "\\n"
	astNewlineChar    = "\n"
)

// Levenshtein similarity threshold.
const (
	levenshteinMaxSuggestions = 3
	levenshteinThresholdExtra = 2
)

// DryRunResult contains the results of a dry-run execution.
// Dry-run validates the template structure without executing resolvers.
type DryRunResult struct {
	// Valid indicates if the template structure is valid
	Valid bool

	// Output is the template with placeholders for dynamic content
	Output string

	// Variables lists all variable references found in the template
	Variables []VariableReference

	// Resolvers lists all resolver invocations found in the template
	Resolvers []ResolverReference

	// Includes lists all template includes found
	Includes []IncludeReference

	// Conditionals lists all conditional blocks found
	Conditionals []ConditionalReference

	// Loops lists all loop blocks found
	Loops []LoopReference

	// Errors contains any structural errors found
	Errors []string

	// Warnings contains non-fatal issues
	Warnings []string

	// MissingVariables lists variables that are referenced but not in data
	MissingVariables []string

	// UnusedVariables lists variables in data that are not referenced
	UnusedVariables []string
}

// VariableReference represents a variable reference in a template.
type VariableReference struct {
	Name        string   // Variable path (e.g., "user.name")
	Default     string   // Default value if specified
	Line        int      // Source line number
	Column      int      // Source column number
	HasDefault  bool     // Whether a default was specified
	InData      bool     // Whether the variable exists in provided data
	Suggestions []string // Similar variable names if not found
}

// ResolverReference represents a resolver invocation in a template.
type ResolverReference struct {
	TagName    string            // Resolver tag name
	Attributes map[string]string // Attributes passed to resolver
	Line       int               // Source line number
	Column     int               // Source column number
	Registered bool              // Whether resolver is registered
}

// IncludeReference represents a template include in a template.
type IncludeReference struct {
	TemplateName string            // Name of included template
	Attributes   map[string]string // Additional attributes
	Line         int               // Source line number
	Column       int               // Source column number
	Exists       bool              // Whether template is registered
	Isolated     bool              // Whether isolate="true"
}

// ConditionalReference represents a conditional block in a template.
type ConditionalReference struct {
	Condition string // The eval expression
	Line      int    // Source line number
	Column    int    // Source column number
	HasElseIf bool   // Whether it has elseif branches
	HasElse   bool   // Whether it has an else branch
}

// LoopReference represents a loop block in a template.
type LoopReference struct {
	ItemVar  string // Loop item variable name
	IndexVar string // Loop index variable name
	Source   string // Source collection path
	Line     int    // Source line number
	Column   int    // Source column number
	Limit    int    // Loop limit if specified
	InData   bool   // Whether source exists in data
}

// ExplainResult contains detailed execution explanation.
type ExplainResult struct {
	// AST is a human-readable representation of the parsed AST
	AST string

	// Steps contains the execution steps in order
	Steps []ExecutionStep

	// Variables shows all variable accesses during execution
	Variables []VariableAccess

	// Resolvers shows all resolver invocations
	Resolvers []ResolverInvocation

	// Timing contains execution timing information
	Timing ExecutionTiming

	// Output is the final rendered output
	Output string

	// Error is set if execution failed
	Error error
}

// ExecutionStep represents a single step in template execution.
type ExecutionStep struct {
	StepNumber  int           // Step number (1-based)
	Type        string        // Step type (text, variable, resolver, conditional, loop, include)
	Description string        // Human-readable description
	Input       string        // Input to this step
	Output      string        // Output from this step
	Duration    time.Duration // Time taken for this step
	Line        int           // Source line number
	Column      int           // Source column number
}

// VariableAccess records a variable access during execution.
type VariableAccess struct {
	Path    string // Variable path accessed
	Value   any    // Value retrieved (or nil if not found)
	Found   bool   // Whether the variable was found
	Default string // Default value used (if any)
	Line    int    // Source line number
	Column  int    // Source column number
}

// ResolverInvocation records a resolver invocation during execution.
type ResolverInvocation struct {
	TagName    string            // Resolver tag name
	Attributes map[string]string // Attributes passed
	Output     string            // Output produced
	Error      error             // Error if any
	Duration   time.Duration     // Time taken
	Line       int               // Source line number
	Column     int               // Source column number
}

// ExecutionTiming contains timing information for execution.
type ExecutionTiming struct {
	Total        time.Duration // Total execution time
	Parsing      time.Duration // Time spent parsing
	Execution    time.Duration // Time spent executing
	ResolverTime time.Duration // Total time in resolvers
	VariableTime time.Duration // Total time resolving variables
}

// DryRun performs a dry-run of the template without executing resolvers.
// It validates the template structure and reports all dynamic elements.
func (t *Template) DryRun(ctx context.Context, data map[string]any) *DryRunResult {
	result := &DryRunResult{
		Valid:            true,
		Variables:        make([]VariableReference, 0),
		Resolvers:        make([]ResolverReference, 0),
		Includes:         make([]IncludeReference, 0),
		Conditionals:     make([]ConditionalReference, 0),
		Loops:            make([]LoopReference, 0),
		Errors:           make([]string, 0),
		Warnings:         make([]string, 0),
		MissingVariables: make([]string, 0),
		UnusedVariables:  make([]string, 0),
	}

	// Track which data keys are used
	usedKeys := make(map[string]bool)

	// Collect available keys for suggestions
	availableKeys := collectAllKeys(data, "")

	// Walk the AST and collect references
	t.walkASTForDryRun(t.ast, data, result, usedKeys, availableKeys)

	// Find missing variables
	missingSet := make(map[string]bool)
	for _, v := range result.Variables {
		if !v.InData && !v.HasDefault {
			missingSet[v.Name] = true
		}
	}
	for name := range missingSet {
		result.MissingVariables = append(result.MissingVariables, name)
	}
	sort.Strings(result.MissingVariables)

	// Find unused variables
	for _, key := range availableKeys {
		if !usedKeys[key] {
			// Only report top-level unused keys
			if !strings.Contains(key, PathSeparator) {
				result.UnusedVariables = append(result.UnusedVariables, key)
			}
		}
	}
	sort.Strings(result.UnusedVariables)

	// Generate placeholder output
	result.Output = t.generatePlaceholderOutput(t.ast, data)

	// Set valid based on errors
	if len(result.Errors) > 0 {
		result.Valid = false
	}

	return result
}

// walkASTForDryRun recursively walks the AST to collect dry-run information.
func (t *Template) walkASTForDryRun(node any, data map[string]any, result *DryRunResult, usedKeys map[string]bool, availableKeys []string) {
	switch n := node.(type) {
	case *internal.RootNode:
		for _, child := range n.Children {
			t.walkASTForDryRun(child, data, result, usedKeys, availableKeys)
		}

	case *internal.TagNode:
		t.processTagNodeForDryRun(n, data, result, usedKeys, availableKeys)

	case *internal.ConditionalNode:
		t.processConditionalNodeForDryRun(n, data, result, usedKeys, availableKeys)

	case *internal.ForNode:
		t.processForNodeForDryRun(n, data, result, usedKeys, availableKeys)

	case *internal.SwitchNode:
		t.processSwitchNodeForDryRun(n, data, result, usedKeys, availableKeys)
	}
}

// processTagNodeForDryRun processes a tag node for dry-run.
func (t *Template) processTagNodeForDryRun(n *internal.TagNode, data map[string]any, result *DryRunResult, usedKeys map[string]bool, availableKeys []string) {
	attrs := n.Attributes.Map()
	pos := n.Pos()
	line := pos.Line
	col := pos.Column

	switch n.Name {
	case TagNameVar:
		varName, _ := n.Attributes.Get(AttrName)
		defaultVal := n.Attributes.GetDefault(AttrDefault, "")
		hasDefault := n.Attributes.Has(AttrDefault)

		// Check if variable exists in data
		inData := hasPath(data, varName)
		if inData {
			markKeyUsed(usedKeys, varName)
		}

		// Find suggestions if not found
		var suggestions []string
		if !inData && !hasDefault {
			suggestions = findSimilarStrings(varName, availableKeys, levenshteinMaxSuggestions)
		}

		result.Variables = append(result.Variables, VariableReference{
			Name:        varName,
			Default:     defaultVal,
			Line:        line,
			Column:      col,
			HasDefault:  hasDefault,
			InData:      inData,
			Suggestions: suggestions,
		})

	case TagNameInclude:
		tmplName, _ := n.Attributes.Get(AttrTemplate)
		isolated := n.Attributes.GetDefault(AttrIsolate, "") == AttrValueTrue

		// Check if template exists
		exists := false
		if t.engine != nil {
			exists = t.engine.HasTemplate(tmplName)
		}

		result.Includes = append(result.Includes, IncludeReference{
			TemplateName: tmplName,
			Attributes:   attrs,
			Line:         line,
			Column:       col,
			Exists:       exists,
			Isolated:     isolated,
		})

		if !exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf(dryRunWarnInclude, line, tmplName))
		}

	case TagNameRaw, TagNameComment:
		// No action needed for raw/comment

	default:
		// Custom resolver
		result.Resolvers = append(result.Resolvers, ResolverReference{
			TagName:    n.Name,
			Attributes: attrs,
			Line:       line,
			Column:     col,
			Registered: true, // Assume registered since it parsed
		})
	}
}

// processConditionalNodeForDryRun processes a conditional node for dry-run.
func (t *Template) processConditionalNodeForDryRun(n *internal.ConditionalNode, data map[string]any, result *DryRunResult, usedKeys map[string]bool, availableKeys []string) {
	pos := n.Pos()

	// Count branches
	hasElseIf := false
	hasElse := false
	firstCondition := ""

	for i, branch := range n.Branches {
		if i == 0 {
			firstCondition = branch.Condition
		} else if branch.IsElse {
			hasElse = true
		} else {
			hasElseIf = true
		}
	}

	result.Conditionals = append(result.Conditionals, ConditionalReference{
		Condition: firstCondition,
		Line:      pos.Line,
		Column:    pos.Column,
		HasElseIf: hasElseIf,
		HasElse:   hasElse,
	})

	// Walk all branches
	for _, branch := range n.Branches {
		for _, child := range branch.Children {
			t.walkASTForDryRun(child, data, result, usedKeys, availableKeys)
		}
	}
}

// processForNodeForDryRun processes a for node for dry-run.
func (t *Template) processForNodeForDryRun(n *internal.ForNode, data map[string]any, result *DryRunResult, usedKeys map[string]bool, availableKeys []string) {
	pos := n.Pos()
	inData := hasPath(data, n.Source)
	if inData {
		markKeyUsed(usedKeys, n.Source)
	}

	result.Loops = append(result.Loops, LoopReference{
		ItemVar:  n.ItemVar,
		IndexVar: n.IndexVar,
		Source:   n.Source,
		Line:     pos.Line,
		Column:   pos.Column,
		Limit:    n.Limit,
		InData:   inData,
	})

	if !inData {
		result.Warnings = append(result.Warnings, fmt.Sprintf(dryRunWarnLoopSrc, pos.Line, n.Source))
	}

	// Walk body (with loop variables added conceptually)
	for _, child := range n.Children {
		t.walkASTForDryRun(child, data, result, usedKeys, availableKeys)
	}
}

// processSwitchNodeForDryRun processes a switch node for dry-run.
func (t *Template) processSwitchNodeForDryRun(n *internal.SwitchNode, data map[string]any, result *DryRunResult, usedKeys map[string]bool, availableKeys []string) {
	// Walk all cases
	for _, c := range n.Cases {
		for _, child := range c.Children {
			t.walkASTForDryRun(child, data, result, usedKeys, availableKeys)
		}
	}
	if n.Default != nil {
		for _, child := range n.Default.Children {
			t.walkASTForDryRun(child, data, result, usedKeys, availableKeys)
		}
	}
}

// generatePlaceholderOutput generates output with placeholders for dynamic content.
func (t *Template) generatePlaceholderOutput(node any, data map[string]any) string {
	var sb strings.Builder
	t.generatePlaceholders(node, data, &sb)
	return sb.String()
}

// generatePlaceholders recursively generates placeholder output.
func (t *Template) generatePlaceholders(node any, data map[string]any, sb *strings.Builder) {
	switch n := node.(type) {
	case *internal.RootNode:
		for _, child := range n.Children {
			t.generatePlaceholders(child, data, sb)
		}

	case *internal.TextNode:
		sb.WriteString(n.Content)

	case *internal.TagNode:
		switch n.Name {
		case TagNameVar:
			varName, _ := n.Attributes.Get(AttrName)
			defaultVal := n.Attributes.GetDefault(AttrDefault, "")

			// Try to get actual value
			if val, ok := getPath(data, varName); ok {
				sb.WriteString(fmt.Sprintf("%v", val))
			} else if defaultVal != "" {
				sb.WriteString(defaultVal)
			} else {
				sb.WriteString(fmt.Sprintf(placeholderVar, varName))
			}

		case TagNameInclude:
			tmplName, _ := n.Attributes.Get(AttrTemplate)
			sb.WriteString(fmt.Sprintf(placeholderInclude, tmplName))

		case TagNameRaw:
			sb.WriteString(n.RawContent)

		case TagNameComment:
			// Comments produce no output

		default:
			sb.WriteString(fmt.Sprintf(placeholderTag, n.Name))
		}

	case *internal.ConditionalNode:
		if len(n.Branches) > 0 {
			sb.WriteString(fmt.Sprintf(placeholderIfOpen, n.Branches[0].Condition))
			for _, child := range n.Branches[0].Children {
				t.generatePlaceholders(child, data, sb)
			}
			for i := 1; i < len(n.Branches); i++ {
				branch := n.Branches[i]
				if branch.IsElse {
					sb.WriteString(placeholderElse)
				} else {
					sb.WriteString(fmt.Sprintf(placeholderElseIf, branch.Condition))
				}
				for _, child := range branch.Children {
					t.generatePlaceholders(child, data, sb)
				}
			}
			sb.WriteString(placeholderIfClose)
		}

	case *internal.ForNode:
		sb.WriteString(fmt.Sprintf(placeholderForOpen, n.ItemVar, n.Source))
		for _, child := range n.Children {
			t.generatePlaceholders(child, data, sb)
		}
		sb.WriteString(placeholderForClose)

	case *internal.SwitchNode:
		sb.WriteString(fmt.Sprintf(placeholderSwitchOpen, n.Expression))
		for _, c := range n.Cases {
			if c.Value != "" {
				sb.WriteString(fmt.Sprintf(placeholderCaseVal, c.Value))
			} else {
				sb.WriteString(fmt.Sprintf(placeholderCaseEval, c.Eval))
			}
			for _, child := range c.Children {
				t.generatePlaceholders(child, data, sb)
			}
		}
		if n.Default != nil {
			sb.WriteString(placeholderDefault)
			for _, child := range n.Default.Children {
				t.generatePlaceholders(child, data, sb)
			}
		}
		sb.WriteString(placeholderSwitchClose)
	}
}

// Explain provides detailed execution explanation for debugging.
func (t *Template) Explain(ctx context.Context, data map[string]any) *ExplainResult {
	result := &ExplainResult{
		Steps:     make([]ExecutionStep, 0),
		Variables: make([]VariableAccess, 0),
		Resolvers: make([]ResolverInvocation, 0),
	}

	startTime := time.Now()

	// Generate AST representation
	result.AST = t.formatAST(t.ast, 0)

	// Execute with tracking
	execCtx := NewContextWithStrategy(data, t.config.errorStrategy)
	if t.engine != nil {
		execCtx = execCtx.WithEngine(t.engine)
	}

	execStart := time.Now()
	output, err := t.executor.Execute(ctx, t.ast, execCtx)
	execDuration := time.Since(execStart)

	result.Output = output
	result.Error = err
	result.Timing = ExecutionTiming{
		Total:     time.Since(startTime),
		Execution: execDuration,
	}

	// Add variable accesses from context keys
	t.collectVariableAccesses(t.ast, data, result)

	return result
}

// formatAST formats the AST as a human-readable string.
func (t *Template) formatAST(node any, depth int) string {
	indent := strings.Repeat(astIndentUnit, depth)
	var sb strings.Builder

	switch n := node.(type) {
	case *internal.RootNode:
		sb.WriteString(fmt.Sprintf(astRootLabel, indent))
		for _, child := range n.Children {
			sb.WriteString(t.formatAST(child, depth+1))
		}

	case *internal.TextNode:
		content := n.Content
		if len(content) > astTextMaxLen {
			content = content[:astTextMaxLen] + astTextTruncation
		}
		content = strings.ReplaceAll(content, astNewlineChar, astNewlineEscape)
		sb.WriteString(fmt.Sprintf(astTextLabel, indent, content))

	case *internal.TagNode:
		sb.WriteString(fmt.Sprintf(astTagLabel, indent, n.Name))
		if len(n.Attributes.Keys()) > 0 {
			attrs := make([]string, 0)
			for _, k := range n.Attributes.Keys() {
				v, _ := n.Attributes.Get(k)
				attrs = append(attrs, fmt.Sprintf(astAttrPair, k, v))
			}
			sb.WriteString(fmt.Sprintf(astAttrsFmt, strings.Join(attrs, ", ")))
		}
		pos := n.Pos()
		sb.WriteString(fmt.Sprintf(astLineFmt, pos.Line))

	case *internal.ConditionalNode:
		pos := n.Pos()
		condition := ""
		if len(n.Branches) > 0 {
			condition = n.Branches[0].Condition
		}
		sb.WriteString(fmt.Sprintf(astCondFmt, indent, condition, pos.Line))
		for i, branch := range n.Branches {
			if i == 0 {
				sb.WriteString(fmt.Sprintf(astThenLabel, indent))
			} else if branch.IsElse {
				sb.WriteString(fmt.Sprintf(astElseLabel, indent))
			} else {
				sb.WriteString(fmt.Sprintf(astElseIfFmt, indent, branch.Condition))
			}
			for _, child := range branch.Children {
				sb.WriteString(t.formatAST(child, depth+2))
			}
		}

	case *internal.ForNode:
		pos := n.Pos()
		sb.WriteString(fmt.Sprintf(astForFmt, indent, n.ItemVar, n.Source))
		if n.IndexVar != "" {
			sb.WriteString(fmt.Sprintf(astForIndex, n.IndexVar))
		}
		if n.Limit > 0 {
			sb.WriteString(fmt.Sprintf(astForLimit, n.Limit))
		}
		sb.WriteString(fmt.Sprintf(astLineFmt, pos.Line))
		for _, child := range n.Children {
			sb.WriteString(t.formatAST(child, depth+1))
		}

	case *internal.SwitchNode:
		pos := n.Pos()
		sb.WriteString(fmt.Sprintf(astSwitchFmt, indent, n.Expression, pos.Line))
		for _, c := range n.Cases {
			if c.Value != "" {
				sb.WriteString(fmt.Sprintf(astCaseVal, indent, c.Value))
			} else {
				sb.WriteString(fmt.Sprintf(astCaseEval, indent, c.Eval))
			}
			for _, child := range c.Children {
				sb.WriteString(t.formatAST(child, depth+2))
			}
		}
		if n.Default != nil {
			sb.WriteString(fmt.Sprintf(astDefaultLbl, indent))
			for _, child := range n.Default.Children {
				sb.WriteString(t.formatAST(child, depth+2))
			}
		}
	}

	return sb.String()
}

// collectVariableAccesses collects variable accesses from the AST.
func (t *Template) collectVariableAccesses(node any, data map[string]any, result *ExplainResult) {
	switch n := node.(type) {
	case *internal.RootNode:
		for _, child := range n.Children {
			t.collectVariableAccesses(child, data, result)
		}

	case *internal.TagNode:
		if n.Name == TagNameVar {
			varName, _ := n.Attributes.Get(AttrName)
			defaultVal := n.Attributes.GetDefault(AttrDefault, "")
			pos := n.Pos()

			val, found := getPath(data, varName)
			result.Variables = append(result.Variables, VariableAccess{
				Path:    varName,
				Value:   val,
				Found:   found,
				Default: defaultVal,
				Line:    pos.Line,
				Column:  pos.Column,
			})
		}

	case *internal.ConditionalNode:
		for _, branch := range n.Branches {
			for _, child := range branch.Children {
				t.collectVariableAccesses(child, data, result)
			}
		}

	case *internal.ForNode:
		for _, child := range n.Children {
			t.collectVariableAccesses(child, data, result)
		}

	case *internal.SwitchNode:
		for _, c := range n.Cases {
			for _, child := range c.Children {
				t.collectVariableAccesses(child, data, result)
			}
		}
		if n.Default != nil {
			for _, child := range n.Default.Children {
				t.collectVariableAccesses(child, data, result)
			}
		}
	}
}

// Helper functions

// hasPath checks if a path exists in data.
func hasPath(data map[string]any, path string) bool {
	_, ok := getPath(data, path)
	return ok
}

// getPath retrieves a value by dot-notation path.
func getPath(data map[string]any, path string) (any, bool) {
	if path == "" || data == nil {
		return nil, false
	}

	parts := strings.Split(path, PathSeparator)
	var current any = data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return nil, false
			}
			current = val
		case map[string]string:
			val, ok := v[part]
			if !ok {
				return nil, false
			}
			current = val
		default:
			return nil, false
		}
	}

	return current, true
}

// collectAllKeys collects all keys from nested maps with dot notation.
func collectAllKeys(data map[string]any, prefix string) []string {
	var keys []string
	for k, v := range data {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + PathSeparator + k
		}
		keys = append(keys, fullKey)

		// Recurse into nested maps
		if nested, ok := v.(map[string]any); ok {
			keys = append(keys, collectAllKeys(nested, fullKey)...)
		}
	}
	return keys
}

// markKeyUsed marks a key and its parent keys as used.
func markKeyUsed(usedKeys map[string]bool, path string) {
	usedKeys[path] = true
	// Also mark parent paths
	parts := strings.Split(path, PathSeparator)
	for i := 1; i < len(parts); i++ {
		parentPath := strings.Join(parts[:i], PathSeparator)
		usedKeys[parentPath] = true
	}
}

// findSimilarStrings finds strings similar to target using Levenshtein distance.
func findSimilarStrings(target string, candidates []string, maxResults int) []string {
	type scored struct {
		str   string
		score int
	}

	var scoredCandidates []scored
	for _, c := range candidates {
		dist := levenshteinDistance(strings.ToLower(target), strings.ToLower(c))
		// Only include if reasonably similar (distance < half the length)
		if dist <= len(target)/2+levenshteinThresholdExtra {
			scoredCandidates = append(scoredCandidates, scored{c, dist})
		}
	}

	// Sort by distance
	sort.Slice(scoredCandidates, func(i, j int) bool {
		return scoredCandidates[i].score < scoredCandidates[j].score
	})

	// Return top results
	results := make([]string, 0, maxResults)
	for i := 0; i < len(scoredCandidates) && i < maxResults; i++ {
		results = append(results, scoredCandidates[i].str)
	}
	return results
}

// levenshteinDistance calculates the edit distance between two strings.
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create matrix
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(a); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}

	// Fill in the rest
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = minOfThree(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

// minOfThree returns the minimum of three integers.
func minOfThree(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// String returns a human-readable summary of the dry-run result.
func (r *DryRunResult) String() string {
	var sb strings.Builder

	sb.WriteString(dryRunHeader)
	sb.WriteString(fmt.Sprintf(dryRunValidLabel, r.Valid))

	if len(r.Variables) > 0 {
		sb.WriteString(fmt.Sprintf(dryRunVariablesHeader, len(r.Variables)))
		for _, v := range r.Variables {
			status := dryRunStatusFound
			if !v.InData {
				if v.HasDefault {
					status = fmt.Sprintf(dryRunStatusDefault, v.Default)
				} else {
					status = dryRunStatusMissing
				}
			}
			sb.WriteString(fmt.Sprintf(dryRunStatusVarLine, v.Name, v.Line, status))
			if len(v.Suggestions) > 0 {
				sb.WriteString(fmt.Sprintf(dryRunStatusSuggestion, strings.Join(v.Suggestions, ", ")))
			}
		}
	}

	if len(r.Resolvers) > 0 {
		sb.WriteString(fmt.Sprintf(dryRunResolversHeader, len(r.Resolvers)))
		for _, res := range r.Resolvers {
			sb.WriteString(fmt.Sprintf(dryRunStatusResSummary, res.TagName, res.Line))
		}
	}

	if len(r.Includes) > 0 {
		sb.WriteString(fmt.Sprintf(dryRunIncludesHeader, len(r.Includes)))
		for _, inc := range r.Includes {
			status := dryRunStatusFound
			if !inc.Exists {
				status = dryRunStatusNotFound
			}
			sb.WriteString(fmt.Sprintf(dryRunStatusIncLine, inc.TemplateName, inc.Line, status))
		}
	}

	if len(r.Conditionals) > 0 {
		sb.WriteString(fmt.Sprintf(dryRunConditionalsHeader, len(r.Conditionals)))
		for _, cond := range r.Conditionals {
			sb.WriteString(fmt.Sprintf(dryRunStatusCondLine, cond.Condition, cond.Line))
		}
	}

	if len(r.Loops) > 0 {
		sb.WriteString(fmt.Sprintf(dryRunLoopsHeader, len(r.Loops)))
		for _, loop := range r.Loops {
			status := dryRunStatusLoopFound
			if !loop.InData {
				status = dryRunStatusLoopMiss
			}
			sb.WriteString(fmt.Sprintf(dryRunStatusLoopLine, loop.ItemVar, loop.Source, loop.Line, status))
		}
	}

	if len(r.MissingVariables) > 0 {
		sb.WriteString(fmt.Sprintf(dryRunMissingVarsHeader, len(r.MissingVariables)))
		for _, v := range r.MissingVariables {
			sb.WriteString(dryRunListPrefix + v + dryRunNewline)
		}
	}

	if len(r.UnusedVariables) > 0 {
		sb.WriteString(fmt.Sprintf(dryRunUnusedVarsHeader, len(r.UnusedVariables)))
		for _, v := range r.UnusedVariables {
			sb.WriteString(dryRunListPrefix + v + dryRunNewline)
		}
	}

	if len(r.Errors) > 0 {
		sb.WriteString(fmt.Sprintf(dryRunErrorsHeader, len(r.Errors)))
		for _, e := range r.Errors {
			sb.WriteString(dryRunListPrefix + e + dryRunNewline)
		}
	}

	if len(r.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf(dryRunWarningsHeader, len(r.Warnings)))
		for _, w := range r.Warnings {
			sb.WriteString(dryRunListPrefix + w + dryRunNewline)
		}
	}

	sb.WriteString(dryRunOutputHeader)
	sb.WriteString(r.Output)
	sb.WriteString(dryRunNewline)

	return sb.String()
}

// String returns a human-readable summary of the explain result.
func (r *ExplainResult) String() string {
	var sb strings.Builder

	sb.WriteString(explainHeader)

	sb.WriteString(explainASTHeader)
	sb.WriteString(r.AST)

	if len(r.Variables) > 0 {
		sb.WriteString(explainVarsHeader)
		for _, v := range r.Variables {
			var status string
			if !v.Found {
				if v.Default != "" {
					status = fmt.Sprintf(explainVarDefault, v.Default)
				} else {
					status = explainVarNotFound
				}
			} else {
				status = fmt.Sprintf(explainVarValue, v.Value)
			}
			sb.WriteString(fmt.Sprintf(explainVarLine, v.Line, v.Path, status))
		}
	}

	sb.WriteString(explainTimingHeader)
	sb.WriteString(fmt.Sprintf(explainTimingTotal, r.Timing.Total))
	sb.WriteString(fmt.Sprintf(explainTimingExec, r.Timing.Execution))

	if r.Error != nil {
		sb.WriteString(fmt.Sprintf(explainErrorHeader, r.Error))
	}

	sb.WriteString(explainOutputHeader)
	sb.WriteString(r.Output)
	sb.WriteString(dryRunNewline)

	return sb.String()
}
