package exons

import "encoding/json"

// ⛔ A TOOL ALLOW-LIST'S EMPTY FORM IS A NARROWING, AND `omitempty` DELETED IT.
//
// `tools.allow: []` and `mcp_servers[].tools: []` mean "no tools". Absent means
// "no narrowing". Parse kept the difference (yaml decodes `[]` to a non-nil
// empty slice) but both tags carried `omitempty`, and encoding/json and
// yaml.v3 treat a nil and an empty slice alike under it. So any document that
// was parsed and re-emitted — a registry export, a copy, a hydrate — turned
// "no tools" into "every tool". go-vaibstract v1.235.0 enforces both lists,
// which is what made the loss a fail-OPEN rather than a cosmetic one
// (go-exons#3).
//
// The marshallers below emit the lists through a pointer, whose `omitempty`
// omits only a nil pointer. A nil list stays absent; an empty list is written
// as `[]`. Decoding needs no hook: yaml and json already produce the right
// value for `[]`, and `null` decodes to nil, which is the absent meaning.

type toolsConfigWire struct {
	Functions         []*FunctionDef `yaml:"functions,omitempty" json:"functions,omitempty"`
	MCPServers        []*MCPServer   `yaml:"mcp_servers,omitempty" json:"mcp_servers,omitempty"`
	ToolChoice        string         `yaml:"tool_choice,omitempty" json:"tool_choice,omitempty"`
	ParallelToolCalls *bool          `yaml:"parallel_tool_calls,omitempty" json:"parallel_tool_calls,omitempty"`
	Allow             *[]string      `yaml:"allow,omitempty" json:"allow,omitempty"`
}

type mcpServerWire struct {
	Name      string    `yaml:"name" json:"name"`
	URL       string    `yaml:"url" json:"url"`
	Transport string    `yaml:"transport,omitempty" json:"transport,omitempty"`
	Tools     *[]string `yaml:"tools,omitempty" json:"tools,omitempty"`
}

// presentList is a pointer to l when l is present (non-nil, possibly empty),
// nil when it is absent.
func presentList(l []string) *[]string {
	if l == nil {
		return nil
	}
	return &l
}

func (tc ToolsConfig) wire() toolsConfigWire {
	return toolsConfigWire{
		Functions:         tc.Functions,
		MCPServers:        tc.MCPServers,
		ToolChoice:        tc.ToolChoice,
		ParallelToolCalls: tc.ParallelToolCalls,
		Allow:             presentList(tc.Allow),
	}
}

// MarshalYAML keeps an empty `allow` list — see the note at the top of this file.
func (tc ToolsConfig) MarshalYAML() (any, error) { return tc.wire(), nil }

// MarshalJSON keeps an empty `allow` list — see the note at the top of this file.
func (tc ToolsConfig) MarshalJSON() ([]byte, error) { return json.Marshal(tc.wire()) }

func (m MCPServer) wire() mcpServerWire {
	return mcpServerWire{Name: m.Name, URL: m.URL, Transport: m.Transport, Tools: presentList(m.Tools)}
}

// MarshalYAML keeps an empty `tools` list — see the note at the top of this file.
func (m MCPServer) MarshalYAML() (any, error) { return m.wire(), nil }

// MarshalJSON keeps an empty `tools` list — see the note at the top of this file.
func (m MCPServer) MarshalJSON() ([]byte, error) { return json.Marshal(m.wire()) }

// IsZero reports whether the block declares nothing at all. A block that only
// narrows (`allow:` with no functions or servers), or only sets tool_choice /
// parallel_tool_calls, is NOT zero: it constrains the tools a runtime adds
// itself, and an export must keep it. HasTools answers a different question —
// "are there tools to render a catalog from" — and must not be used to decide
// whether to emit the block.
func (tc *ToolsConfig) IsZero() bool {
	return tc == nil || (len(tc.Functions) == 0 && len(tc.MCPServers) == 0 &&
		tc.ToolChoice == "" && tc.ParallelToolCalls == nil && tc.Allow == nil)
}
