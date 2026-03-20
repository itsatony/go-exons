package exons

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// AgentDryRunIssue represents a single issue found during dry run validation.
type AgentDryRunIssue struct {
	// Category is the issue category (one of AgentDryRunCategory* constants).
	Category string
	// Message is the human-readable issue description.
	Message string
	// Location identifies where the issue was found (e.g. "spec", "skill:web-search", "message[0]").
	Location string
	// Err is the underlying error, if any.
	Err error
}

// AgentDryRunResult holds all results of an agent dry run.
// It collects all issues found during validation without stopping at the first failure.
type AgentDryRunResult struct {
	// Issues is the list of all issues found during the dry run.
	Issues []AgentDryRunIssue
	// SkillsResolved is the number of skills successfully resolved.
	SkillsResolved int
	// ToolsDefined is the number of tool functions and MCP servers defined.
	ToolsDefined int
	// MessageCount is the number of messages defined in the spec.
	MessageCount int
}

// OK returns true if the dry run found no issues.
func (r *AgentDryRunResult) OK() bool {
	return len(r.Issues) == 0
}

// HasErrors returns true if the dry run found any issues.
func (r *AgentDryRunResult) HasErrors() bool {
	return len(r.Issues) > 0
}

// String returns a formatted summary of the dry run result.
func (r *AgentDryRunResult) String() string {
	if r.OK() {
		return fmt.Sprintf(AgentDryRunSummaryOK, r.SkillsResolved, r.ToolsDefined, r.MessageCount)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(AgentDryRunSummaryIssues, len(r.Issues)))
	for _, issue := range r.Issues {
		sb.WriteString(fmt.Sprintf(AgentDryRunIssueFormat, issue.Category, issue.Location, issue.Message))
	}
	return sb.String()
}

// AgentDryRun validates an agent spec by checking all references, templates, and credentials
// without actually producing output. It collects ALL issues rather than stopping at the first.
//
// Validation steps:
//  1. Validate spec via ValidateAsAgent
//  2. Resolve each skill via SpecResolver (skipped if no resolver in opts)
//  3. Validate credential refs via ValidateCredentialRefs
//  4. Parse each message template through engine (parse-only, only if content contains {~)
//  5. Parse body through engine (parse-only, only if body contains {~)
//  6. Count tools from spec.Tools
func (s *Spec) AgentDryRun(ctx context.Context, opts *CompileOptions) *AgentDryRunResult {
	result := &AgentDryRunResult{}

	// Nil spec guard
	if s == nil {
		result.Issues = append(result.Issues, AgentDryRunIssue{
			Category: AgentDryRunCategoryValidation,
			Message:  ErrMsgAgentDryRunNilSpec,
			Location: DryRunLocationSpec,
		})
		return result
	}

	// Step 1: Validate spec as agent
	if err := s.ValidateAsAgent(); err != nil {
		result.Issues = append(result.Issues, AgentDryRunIssue{
			Category: AgentDryRunCategoryValidation,
			Message:  err.Error(),
			Location: DryRunLocationSpec,
			Err:      err,
		})
	}

	// Step 2: Resolve each skill via SpecResolver
	if opts != nil && opts.Resolver != nil && len(s.Skills) > 0 {
		for _, skill := range s.Skills {
			_, _, err := opts.Resolver.ResolveSpec(ctx, skill.Slug, RefVersionLatest)
			if err != nil {
				result.Issues = append(result.Issues, AgentDryRunIssue{
					Category: AgentDryRunCategoryResolver,
					Message:  err.Error(),
					Location: DryRunLocationSkillPrefix + skill.Slug,
					Err:      err,
				})
			} else {
				result.SkillsResolved++
			}
		}
	}

	// Step 3: Validate credential refs
	if err := s.ValidateCredentialRefs(); err != nil {
		result.Issues = append(result.Issues, AgentDryRunIssue{
			Category: AgentDryRunCategoryCredential,
			Message:  err.Error(),
			Location: DryRunLocationCredentials,
			Err:      err,
		})
	}

	// Step 4: Parse each message template through engine (parse-only)
	engine := compileEngine(opts)
	for i, mt := range s.Messages {
		if strings.Contains(mt.Content, DefaultOpenDelim) {
			_, err := engine.Parse(mt.Content)
			if err != nil {
				location := DryRunLocationMessagePrefix + strconv.Itoa(i) + DryRunLocationMessageSuffix
				result.Issues = append(result.Issues, AgentDryRunIssue{
					Category: AgentDryRunCategoryTemplate,
					Message:  err.Error(),
					Location: location,
					Err:      err,
				})
			}
		}
	}
	result.MessageCount = len(s.Messages)

	// Step 5: Parse body through engine (parse-only)
	if s.Body != "" && strings.Contains(s.Body, DefaultOpenDelim) {
		_, err := engine.Parse(s.Body)
		if err != nil {
			result.Issues = append(result.Issues, AgentDryRunIssue{
				Category: AgentDryRunCategoryTemplate,
				Message:  err.Error(),
				Location: DryRunLocationBody,
				Err:      err,
			})
		}
	}

	// Step 6: Count tools
	if s.Tools != nil {
		result.ToolsDefined = len(s.Tools.Functions) + len(s.Tools.MCPServers)
	}

	return result
}
