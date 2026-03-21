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

// HasTools returns true if the ToolsConfig has at least one function or MCP server defined.
func (tc *ToolsConfig) HasTools() bool {
	return tc != nil && (len(tc.Functions) > 0 || len(tc.MCPServers) > 0)
}

// ToOpenAITool returns an OpenAI-compatible tool definition map for this function.
// Format: {"type": "function", "function": {"name": ..., "description": ..., "parameters": ...}}
func (f *FunctionDef) ToOpenAITool() map[string]any {
	if f == nil {
		return nil
	}
	fn := map[string]any{
		ToolKeyName: f.Name,
	}
	if f.Description != "" {
		fn[ToolKeyDescription] = f.Description
	}
	if f.Parameters != nil {
		fn[ToolKeyParameters] = f.Parameters
	}
	return map[string]any{
		ToolKeyType:     ToolKeyFunction,
		ToolKeyFunction: fn,
	}
}

// ToAnthropicTool returns an Anthropic-compatible tool definition map.
// Format: {"name": ..., "description": ..., "input_schema": ...}
func (f *FunctionDef) ToAnthropicTool() map[string]any {
	if f == nil {
		return nil
	}
	result := map[string]any{
		ToolKeyName: f.Name,
	}
	if f.Description != "" {
		result[ToolKeyDescription] = f.Description
	}
	if f.Parameters != nil {
		result[ToolKeyInputSchema] = f.Parameters
	}
	return result
}

// ToGeminiTool returns a Gemini-compatible tool function declaration map.
// Format: {"name": ..., "description": ..., "parameters": ...}
func (f *FunctionDef) ToGeminiTool() map[string]any {
	if f == nil {
		return nil
	}
	result := map[string]any{
		ToolKeyName: f.Name,
	}
	if f.Description != "" {
		result[ToolKeyDescription] = f.Description
	}
	if f.Parameters != nil {
		result[ToolKeyParameters] = f.Parameters
	}
	return result
}

// ToMCPTool returns an MCP-compatible tool definition map.
// Format: {"name": ..., "description": ..., "inputSchema": ...}
func (f *FunctionDef) ToMCPTool() map[string]any {
	if f == nil {
		return nil
	}
	result := map[string]any{
		ToolKeyName: f.Name,
	}
	if f.Description != "" {
		result[ToolKeyDescription] = f.Description
	}
	if f.Parameters != nil {
		result[ToolKeyInputSchemaCamel] = f.Parameters
	}
	return result
}

// ToMistralTool returns a Mistral-compatible tool definition map.
// Mistral uses OpenAI-compatible format.
func (f *FunctionDef) ToMistralTool() map[string]any {
	return f.ToOpenAITool()
}

// ToCohereTool returns a Cohere-compatible tool definition map.
// Format: {"name": ..., "description": ..., "parameter_definitions": {...}}
// Cohere uses a flat parameter_definitions format where each parameter
// includes its type, description, and required status.
func (f *FunctionDef) ToCohereTool() map[string]any {
	if f == nil {
		return nil
	}
	result := map[string]any{
		ToolKeyName: f.Name,
	}
	if f.Description != "" {
		result[ToolKeyDescription] = f.Description
	}
	if f.Parameters != nil {
		result[ToolKeyParameterDefinitions] = cohereParameterDefs(f.Parameters)
	}
	return result
}

// cohereParameterDefs converts a JSON Schema parameters object to Cohere's
// flat parameter_definitions format.
func cohereParameterDefs(params map[string]any) map[string]any {
	props, _ := params[SchemaKeyProperties].(map[string]any)
	if props == nil {
		return nil
	}

	// Build required set.
	// Two type paths: YAML unmarshal produces []any, JSON unmarshal produces []string.
	requiredSet := make(map[string]bool)
	if reqList, ok := params[ToolKeyRequired]; ok {
		if reqSlice, ok := reqList.([]any); ok {
			for _, r := range reqSlice {
				if s, ok := r.(string); ok {
					requiredSet[s] = true
				}
			}
		}
		if reqStrSlice, ok := reqList.([]string); ok {
			for _, s := range reqStrSlice {
				requiredSet[s] = true
			}
		}
	}

	defs := make(map[string]any, len(props))
	for name, schema := range props {
		def := make(map[string]any)
		if schemaMap, ok := schema.(map[string]any); ok {
			if t, ok := schemaMap[ToolKeyType]; ok {
				def[ToolKeyType] = t
			}
			if d, ok := schemaMap[ToolKeyDescription]; ok {
				def[ToolKeyDescription] = d
			}
		}
		def[ToolKeyRequired] = requiredSet[name]
		defs[name] = def
	}
	return defs
}

// ToOpenAITools returns all functions as OpenAI-compatible tool definitions.
func (tc *ToolsConfig) ToOpenAITools() []map[string]any {
	return toolsConfigToList(tc, (*FunctionDef).ToOpenAITool)
}

// ToAnthropicTools returns all functions as Anthropic-compatible tool definitions.
func (tc *ToolsConfig) ToAnthropicTools() []map[string]any {
	return toolsConfigToList(tc, (*FunctionDef).ToAnthropicTool)
}

// ToGeminiTools returns all functions as Gemini-compatible tool definitions.
func (tc *ToolsConfig) ToGeminiTools() []map[string]any {
	return toolsConfigToList(tc, (*FunctionDef).ToGeminiTool)
}

// ToMCPTools returns all functions as MCP-compatible tool definitions.
func (tc *ToolsConfig) ToMCPTools() []map[string]any {
	return toolsConfigToList(tc, (*FunctionDef).ToMCPTool)
}

// ToMistralTools returns all functions as Mistral-compatible tool definitions.
func (tc *ToolsConfig) ToMistralTools() []map[string]any {
	return toolsConfigToList(tc, (*FunctionDef).ToMistralTool)
}

// ToCohereTools returns all functions as Cohere-compatible tool definitions.
func (tc *ToolsConfig) ToCohereTools() []map[string]any {
	return toolsConfigToList(tc, (*FunctionDef).ToCohereTool)
}

// toolsConfigToList applies a conversion function to all FunctionDefs in a ToolsConfig.
func toolsConfigToList(tc *ToolsConfig, convert func(*FunctionDef) map[string]any) []map[string]any {
	if tc == nil || len(tc.Functions) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(tc.Functions))
	for _, f := range tc.Functions {
		if tool := convert(f); tool != nil {
			result = append(result, tool)
		}
	}
	return result
}

// Clone creates a deep copy of the ToolsConfig.
func (tc *ToolsConfig) Clone() *ToolsConfig {
	if tc == nil {
		return nil
	}
	clone := *tc

	if tc.Functions != nil {
		clone.Functions = make([]*FunctionDef, len(tc.Functions))
		for i, f := range tc.Functions {
			fCopy := *f
			if f.Parameters != nil {
				fCopy.Parameters = deepCopyMap(f.Parameters)
			}
			clone.Functions[i] = &fCopy
		}
	}

	if tc.MCPServers != nil {
		clone.MCPServers = make([]*MCPServer, len(tc.MCPServers))
		for i, m := range tc.MCPServers {
			mCopy := *m
			clone.MCPServers[i] = &mCopy
		}
	}

	if tc.Allow != nil {
		clone.Allow = make([]string, len(tc.Allow))
		copy(clone.Allow, tc.Allow)
	}

	if tc.ParallelToolCalls != nil {
		t := *tc.ParallelToolCalls
		clone.ParallelToolCalls = &t
	}

	return &clone
}

// Clone creates a deep copy of the ConstraintsConfig.
func (cc *ConstraintsConfig) Clone() *ConstraintsConfig {
	if cc == nil {
		return nil
	}
	clone := *cc

	if cc.Behavioral != nil {
		clone.Behavioral = make([]string, len(cc.Behavioral))
		copy(clone.Behavioral, cc.Behavioral)
	}

	if cc.Safety != nil {
		clone.Safety = make([]string, len(cc.Safety))
		copy(clone.Safety, cc.Safety)
	}

	if cc.Operational != nil {
		clone.Operational = cc.Operational.Clone()
	}

	return &clone
}

// Clone creates a deep copy of the OperationalConstraints.
func (oc *OperationalConstraints) Clone() *OperationalConstraints {
	if oc == nil {
		return nil
	}
	clone := *oc

	if oc.MaxTurns != nil {
		t := *oc.MaxTurns
		clone.MaxTurns = &t
	}
	if oc.MaxTokensPerTurn != nil {
		t := *oc.MaxTokensPerTurn
		clone.MaxTokensPerTurn = &t
	}
	if oc.AllowedDomains != nil {
		clone.AllowedDomains = make([]string, len(oc.AllowedDomains))
		copy(clone.AllowedDomains, oc.AllowedDomains)
	}
	if oc.BlockedDomains != nil {
		clone.BlockedDomains = make([]string, len(oc.BlockedDomains))
		copy(clone.BlockedDomains, oc.BlockedDomains)
	}
	if oc.TimeoutSeconds != nil {
		t := *oc.TimeoutSeconds
		clone.TimeoutSeconds = &t
	}
	if oc.MaxToolCalls != nil {
		t := *oc.MaxToolCalls
		clone.MaxToolCalls = &t
	}

	return &clone
}

// CredentialRef declares a credential for provider authentication.
// go-exons stores but does NOT resolve credentials.
type CredentialRef struct {
	Provider string   `yaml:"provider,omitempty" json:"provider,omitempty"`
	Label    string   `yaml:"label,omitempty" json:"label,omitempty"`
	Ref      string   `yaml:"ref,omitempty" json:"ref,omitempty"`
	Scopes   []string `yaml:"scopes,omitempty" json:"scopes,omitempty"`
}

// Validate checks that the CredentialRef has the minimum required fields.
func (cr *CredentialRef) Validate() error {
	if cr == nil {
		return nil
	}
	if cr.Provider == "" {
		return NewSpecValidationError(ErrMsgCredentialMissingProvider, cr.Label)
	}
	return nil
}

// Clone creates a deep copy of the CredentialRef.
func (cr *CredentialRef) Clone() *CredentialRef {
	if cr == nil {
		return nil
	}
	clone := *cr
	if cr.Scopes != nil {
		clone.Scopes = make([]string, len(cr.Scopes))
		copy(clone.Scopes, cr.Scopes)
	}
	return &clone
}
