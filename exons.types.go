package exons

// SkillRef references a skill for agent composition.
type SkillRef struct {
	Slug       string `yaml:"slug" json:"slug"`
	Injection  string `yaml:"injection,omitempty" json:"injection,omitempty"`
	Credential string `yaml:"credential,omitempty" json:"credential,omitempty"`
}

// ToolsConfig defines tool availability for an agent.
type ToolsConfig struct {
	Functions         []*FunctionDef `yaml:"functions,omitempty" json:"functions,omitempty"`
	MCPServers        []*MCPServer   `yaml:"mcp_servers,omitempty" json:"mcp_servers,omitempty"`
	ToolChoice        string         `yaml:"tool_choice,omitempty" json:"tool_choice,omitempty"`
	ParallelToolCalls *bool          `yaml:"parallel_tool_calls,omitempty" json:"parallel_tool_calls,omitempty"`
	Allow             []string       `yaml:"allow,omitempty" json:"allow,omitempty"`
}

// FunctionDef defines a tool function.
type FunctionDef struct {
	Name        string         `yaml:"name" json:"name"`
	Description string         `yaml:"description,omitempty" json:"description,omitempty"`
	Parameters  map[string]any `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

// MCPServer references an MCP server.
type MCPServer struct {
	Name string `yaml:"name" json:"name"`
	URL  string `yaml:"url" json:"url"`
}

// ConstraintsConfig defines agent behavioral and operational constraints.
type ConstraintsConfig struct {
	Behavioral  []string                `yaml:"behavioral,omitempty" json:"behavioral,omitempty"`
	Safety      []string                `yaml:"safety,omitempty" json:"safety,omitempty"`
	Operational *OperationalConstraints `yaml:"operational,omitempty" json:"operational,omitempty"`
}

// OperationalConstraints defines hard limits on agent execution.
type OperationalConstraints struct {
	MaxTurns         *int     `yaml:"max_turns,omitempty" json:"max_turns,omitempty"`
	MaxTokensPerTurn *int     `yaml:"max_tokens_per_turn,omitempty" json:"max_tokens_per_turn,omitempty"`
	AllowedDomains   []string `yaml:"allowed_domains,omitempty" json:"allowed_domains,omitempty"`
	BlockedDomains   []string `yaml:"blocked_domains,omitempty" json:"blocked_domains,omitempty"`
	TimeoutSeconds   *int     `yaml:"timeout_seconds,omitempty" json:"timeout_seconds,omitempty"`
	MaxToolCalls     *int     `yaml:"max_tool_calls,omitempty" json:"max_tool_calls,omitempty"`
}

// CredentialRef declares a credential for provider authentication.
// go-exons stores but does NOT resolve credentials.
type CredentialRef struct {
	Provider string   `yaml:"provider,omitempty" json:"provider,omitempty"`
	Label    string   `yaml:"label,omitempty" json:"label,omitempty"`
	Ref      string   `yaml:"ref,omitempty" json:"ref,omitempty"`
	Scopes   []string `yaml:"scopes,omitempty" json:"scopes,omitempty"`
}
