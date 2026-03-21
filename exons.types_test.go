package exons

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// FunctionDef.ToOpenAITool
// ---------------------------------------------------------------------------

func TestFunctionDef_ToOpenAITool(t *testing.T) {
	tests := []struct {
		name     string
		fn       *FunctionDef
		wantNil  bool
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name:    "nil receiver returns nil",
			fn:      nil,
			wantNil: true,
		},
		{
			name: "minimal function with name only",
			fn:   &FunctionDef{Name: "get_weather"},
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, ToolKeyFunction, result[ToolKeyType])
				fn, ok := result[ToolKeyFunction].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "get_weather", fn[ToolKeyName])
				assert.Nil(t, fn[ToolKeyDescription])
				assert.Nil(t, fn[ToolKeyParameters])
			},
		},
		{
			name: "full function with name, description, and parameters",
			fn: &FunctionDef{
				Name:        "search",
				Description: "Search the web",
				Parameters: map[string]any{
					ToolKeyType: SchemaTypeObject,
					SchemaKeyProperties: map[string]any{
						"query": map[string]any{
							ToolKeyType:        SchemaTypeString,
							ToolKeyDescription: "The search query",
						},
					},
					ToolKeyRequired: []any{"query"},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, ToolKeyFunction, result[ToolKeyType])
				fn, ok := result[ToolKeyFunction].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "search", fn[ToolKeyName])
				assert.Equal(t, "Search the web", fn[ToolKeyDescription])
				params, ok := fn[ToolKeyParameters].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, SchemaTypeObject, params[ToolKeyType])
				props, ok := params[SchemaKeyProperties].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, props, "query")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.fn.ToOpenAITool()
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			tc.validate(t, result)
		})
	}
}

// ---------------------------------------------------------------------------
// FunctionDef.ToAnthropicTool
// ---------------------------------------------------------------------------

func TestFunctionDef_ToAnthropicTool(t *testing.T) {
	tests := []struct {
		name     string
		fn       *FunctionDef
		wantNil  bool
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name:    "nil receiver returns nil",
			fn:      nil,
			wantNil: true,
		},
		{
			name: "minimal function with name only",
			fn:   &FunctionDef{Name: "read_file"},
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "read_file", result[ToolKeyName])
				assert.Nil(t, result[ToolKeyDescription])
				assert.Nil(t, result[ToolKeyInputSchema])
				// Must NOT have OpenAI-style "type"/"function" keys
				assert.Nil(t, result[ToolKeyType])
				assert.Nil(t, result[ToolKeyFunction])
			},
		},
		{
			name: "full function with parameters maps to input_schema",
			fn: &FunctionDef{
				Name:        "create_file",
				Description: "Create a new file",
				Parameters: map[string]any{
					ToolKeyType: SchemaTypeObject,
					SchemaKeyProperties: map[string]any{
						"path": map[string]any{
							ToolKeyType:        SchemaTypeString,
							ToolKeyDescription: "File path",
						},
						"content": map[string]any{
							ToolKeyType:        SchemaTypeString,
							ToolKeyDescription: "File content",
						},
					},
					ToolKeyRequired: []any{"path", "content"},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "create_file", result[ToolKeyName])
				assert.Equal(t, "Create a new file", result[ToolKeyDescription])
				schema, ok := result[ToolKeyInputSchema].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, SchemaTypeObject, schema[ToolKeyType])
				props, ok := schema[SchemaKeyProperties].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, props, "path")
				assert.Contains(t, props, "content")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.fn.ToAnthropicTool()
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			tc.validate(t, result)
		})
	}
}

// ---------------------------------------------------------------------------
// FunctionDef.ToGeminiTool
// ---------------------------------------------------------------------------

func TestFunctionDef_ToGeminiTool(t *testing.T) {
	tests := []struct {
		name     string
		fn       *FunctionDef
		wantNil  bool
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name:    "nil receiver returns nil",
			fn:      nil,
			wantNil: true,
		},
		{
			name: "minimal function with name only",
			fn:   &FunctionDef{Name: "lookup"},
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "lookup", result[ToolKeyName])
				assert.Nil(t, result[ToolKeyDescription])
				assert.Nil(t, result[ToolKeyParameters])
				// Must NOT have OpenAI-style "type"/"function" wrapper
				assert.Nil(t, result[ToolKeyType])
				assert.Nil(t, result[ToolKeyFunction])
			},
		},
		{
			name: "full function uses parameters key directly",
			fn: &FunctionDef{
				Name:        "translate",
				Description: "Translate text between languages",
				Parameters: map[string]any{
					ToolKeyType: SchemaTypeObject,
					SchemaKeyProperties: map[string]any{
						"text": map[string]any{ToolKeyType: SchemaTypeString},
						"lang": map[string]any{ToolKeyType: SchemaTypeString},
					},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "translate", result[ToolKeyName])
				assert.Equal(t, "Translate text between languages", result[ToolKeyDescription])
				params, ok := result[ToolKeyParameters].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, SchemaTypeObject, params[ToolKeyType])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.fn.ToGeminiTool()
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			tc.validate(t, result)
		})
	}
}

// ---------------------------------------------------------------------------
// FunctionDef.ToMCPTool
// ---------------------------------------------------------------------------

func TestFunctionDef_ToMCPTool(t *testing.T) {
	tests := []struct {
		name     string
		fn       *FunctionDef
		wantNil  bool
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name:    "nil receiver returns nil",
			fn:      nil,
			wantNil: true,
		},
		{
			name: "minimal function with name only",
			fn:   &FunctionDef{Name: "ping"},
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "ping", result[ToolKeyName])
				assert.Nil(t, result[ToolKeyDescription])
				assert.Nil(t, result[ToolKeyInputSchemaCamel])
				// Must NOT have other schema keys
				assert.Nil(t, result[ToolKeyInputSchema])
				assert.Nil(t, result[ToolKeyParameters])
			},
		},
		{
			name: "full function uses inputSchema (camelCase)",
			fn: &FunctionDef{
				Name:        "fetch_url",
				Description: "Fetch a URL",
				Parameters: map[string]any{
					ToolKeyType: SchemaTypeObject,
					SchemaKeyProperties: map[string]any{
						"url": map[string]any{ToolKeyType: SchemaTypeString},
					},
					ToolKeyRequired: []any{"url"},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "fetch_url", result[ToolKeyName])
				assert.Equal(t, "Fetch a URL", result[ToolKeyDescription])
				schema, ok := result[ToolKeyInputSchemaCamel].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, SchemaTypeObject, schema[ToolKeyType])
				// Verify it uses camelCase key, not snake_case
				assert.Nil(t, result[ToolKeyInputSchema])
				assert.Nil(t, result[ToolKeyParameters])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.fn.ToMCPTool()
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			tc.validate(t, result)
		})
	}
}

// ---------------------------------------------------------------------------
// FunctionDef.ToMistralTool (delegates to ToOpenAITool)
// ---------------------------------------------------------------------------

func TestFunctionDef_ToMistralTool(t *testing.T) {
	tests := []struct {
		name     string
		fn       *FunctionDef
		wantNil  bool
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name:    "nil receiver returns nil",
			fn:      nil,
			wantNil: true,
		},
		{
			name: "minimal function matches OpenAI format",
			fn:   &FunctionDef{Name: "classify"},
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, ToolKeyFunction, result[ToolKeyType])
				fn, ok := result[ToolKeyFunction].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "classify", fn[ToolKeyName])
			},
		},
		{
			name: "full function identical to OpenAI output",
			fn: &FunctionDef{
				Name:        "summarize",
				Description: "Summarize text",
				Parameters: map[string]any{
					ToolKeyType: SchemaTypeObject,
					SchemaKeyProperties: map[string]any{
						"text": map[string]any{ToolKeyType: SchemaTypeString},
					},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				// Mistral uses OpenAI format — verify structure
				assert.Equal(t, ToolKeyFunction, result[ToolKeyType])
				fn, ok := result[ToolKeyFunction].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "summarize", fn[ToolKeyName])
				assert.Equal(t, "Summarize text", fn[ToolKeyDescription])
				assert.NotNil(t, fn[ToolKeyParameters])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.fn.ToMistralTool()
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			tc.validate(t, result)
		})
	}
}

// TestFunctionDef_ToMistralTool_MatchesOpenAI verifies that ToMistralTool
// produces identical output to ToOpenAITool for the same FunctionDef.
func TestFunctionDef_ToMistralTool_MatchesOpenAI(t *testing.T) {
	fn := &FunctionDef{
		Name:        "summarize",
		Description: "Summarize text",
		Parameters: map[string]any{
			ToolKeyType: SchemaTypeObject,
			SchemaKeyProperties: map[string]any{
				"text": map[string]any{ToolKeyType: SchemaTypeString},
			},
		},
	}
	openaiResult := fn.ToOpenAITool()
	mistralResult := fn.ToMistralTool()
	assert.Equal(t, openaiResult, mistralResult)
}

// ---------------------------------------------------------------------------
// FunctionDef.ToCohereTool
// ---------------------------------------------------------------------------

func TestFunctionDef_ToCohereTool(t *testing.T) {
	tests := []struct {
		name     string
		fn       *FunctionDef
		wantNil  bool
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name:    "nil receiver returns nil",
			fn:      nil,
			wantNil: true,
		},
		{
			name: "minimal function with name only",
			fn:   &FunctionDef{Name: "echo"},
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "echo", result[ToolKeyName])
				assert.Nil(t, result[ToolKeyDescription])
				// No parameters → no parameter_definitions
				assert.Nil(t, result[ToolKeyParameterDefinitions])
			},
		},
		{
			name: "function with parameters produces parameter_definitions",
			fn: &FunctionDef{
				Name:        "search_db",
				Description: "Search the database",
				Parameters: map[string]any{
					ToolKeyType: SchemaTypeObject,
					SchemaKeyProperties: map[string]any{
						"query": map[string]any{
							ToolKeyType:        SchemaTypeString,
							ToolKeyDescription: "Search query",
						},
						"limit": map[string]any{
							ToolKeyType:        SchemaTypeNumber,
							ToolKeyDescription: "Max results",
						},
					},
					ToolKeyRequired: []any{"query"},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "search_db", result[ToolKeyName])
				assert.Equal(t, "Search the database", result[ToolKeyDescription])
				defs, ok := result[ToolKeyParameterDefinitions].(map[string]any)
				require.True(t, ok)

				// Check "query" parameter
				queryDef, ok := defs["query"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, SchemaTypeString, queryDef[ToolKeyType])
				assert.Equal(t, "Search query", queryDef[ToolKeyDescription])
				assert.Equal(t, true, queryDef[ToolKeyRequired])

				// Check "limit" parameter
				limitDef, ok := defs["limit"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, SchemaTypeNumber, limitDef[ToolKeyType])
				assert.Equal(t, "Max results", limitDef[ToolKeyDescription])
				assert.Equal(t, false, limitDef[ToolKeyRequired])
			},
		},
		{
			name: "parameters with no properties returns nil parameter_definitions",
			fn: &FunctionDef{
				Name: "empty_params",
				Parameters: map[string]any{
					ToolKeyType: SchemaTypeObject,
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "empty_params", result[ToolKeyName])
				// cohereParameterDefs returns nil when no properties
				assert.Nil(t, result[ToolKeyParameterDefinitions])
			},
		},
		{
			name: "required field as []string (JSON unmarshal path)",
			fn: &FunctionDef{
				Name: "json_required",
				Parameters: map[string]any{
					ToolKeyType: SchemaTypeObject,
					SchemaKeyProperties: map[string]any{
						"id": map[string]any{
							ToolKeyType: SchemaTypeString,
						},
						"name": map[string]any{
							ToolKeyType: SchemaTypeString,
						},
					},
					ToolKeyRequired: []string{"id", "name"},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				defs, ok := result[ToolKeyParameterDefinitions].(map[string]any)
				require.True(t, ok)

				idDef, ok := defs["id"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, true, idDef[ToolKeyRequired])

				nameDef, ok := defs["name"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, true, nameDef[ToolKeyRequired])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.fn.ToCohereTool()
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			tc.validate(t, result)
		})
	}
}

// ---------------------------------------------------------------------------
// ToolsConfig batch methods: ToOpenAITools, ToAnthropicTools, etc.
// ---------------------------------------------------------------------------

func TestToolsConfig_ToOpenAITools(t *testing.T) {
	tests := []struct {
		name     string
		tc       *ToolsConfig
		wantNil  bool
		wantLen  int
		validate func(t *testing.T, result []map[string]any)
	}{
		{
			name:    "nil ToolsConfig returns nil",
			tc:      nil,
			wantNil: true,
		},
		{
			name:    "empty Functions slice returns nil",
			tc:      &ToolsConfig{Functions: []*FunctionDef{}},
			wantNil: true,
		},
		{
			name: "multiple functions produce correct count",
			tc: &ToolsConfig{
				Functions: []*FunctionDef{
					{Name: "fn_a", Description: "Function A"},
					{Name: "fn_b", Description: "Function B"},
				},
			},
			wantLen: 2,
			validate: func(t *testing.T, result []map[string]any) {
				for _, tool := range result {
					assert.Equal(t, ToolKeyFunction, tool[ToolKeyType])
					fn, ok := tool[ToolKeyFunction].(map[string]any)
					require.True(t, ok)
					assert.NotEmpty(t, fn[ToolKeyName])
				}
			},
		},
		{
			name: "nil functions in slice are skipped",
			tc: &ToolsConfig{
				Functions: []*FunctionDef{
					{Name: "fn_valid"},
					nil,
					{Name: "fn_also_valid"},
				},
			},
			wantLen: 2,
			validate: func(t *testing.T, result []map[string]any) {
				fn0, ok := result[0][ToolKeyFunction].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "fn_valid", fn0[ToolKeyName])
				fn1, ok := result[1][ToolKeyFunction].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "fn_also_valid", fn1[ToolKeyName])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.tc.ToOpenAITools()
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Len(t, result, tc.wantLen)
			if tc.validate != nil {
				tc.validate(t, result)
			}
		})
	}
}

func TestToolsConfig_ToAnthropicTools(t *testing.T) {
	tests := []struct {
		name     string
		tc       *ToolsConfig
		wantNil  bool
		wantLen  int
		validate func(t *testing.T, result []map[string]any)
	}{
		{
			name:    "nil ToolsConfig returns nil",
			tc:      nil,
			wantNil: true,
		},
		{
			name:    "empty Functions returns nil",
			tc:      &ToolsConfig{Functions: []*FunctionDef{}},
			wantNil: true,
		},
		{
			name: "multiple functions use Anthropic format",
			tc: &ToolsConfig{
				Functions: []*FunctionDef{
					{
						Name:        "tool_a",
						Description: "Tool A",
						Parameters:  map[string]any{ToolKeyType: SchemaTypeObject},
					},
					{Name: "tool_b"},
				},
			},
			wantLen: 2,
			validate: func(t *testing.T, result []map[string]any) {
				// First tool should have input_schema
				assert.Equal(t, "tool_a", result[0][ToolKeyName])
				assert.NotNil(t, result[0][ToolKeyInputSchema])
				// Second tool should NOT have input_schema
				assert.Equal(t, "tool_b", result[1][ToolKeyName])
				assert.Nil(t, result[1][ToolKeyInputSchema])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.tc.ToAnthropicTools()
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Len(t, result, tc.wantLen)
			if tc.validate != nil {
				tc.validate(t, result)
			}
		})
	}
}

func TestToolsConfig_ToGeminiTools(t *testing.T) {
	tests := []struct {
		name    string
		tc      *ToolsConfig
		wantNil bool
		wantLen int
	}{
		{
			name:    "nil ToolsConfig returns nil",
			tc:      nil,
			wantNil: true,
		},
		{
			name:    "empty Functions returns nil",
			tc:      &ToolsConfig{Functions: []*FunctionDef{}},
			wantNil: true,
		},
		{
			name: "single function",
			tc: &ToolsConfig{
				Functions: []*FunctionDef{
					{Name: "gemini_tool", Description: "A Gemini tool"},
				},
			},
			wantLen: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.tc.ToGeminiTools()
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Len(t, result, tc.wantLen)
			// Gemini tools have name directly at top level, not nested
			assert.Equal(t, "gemini_tool", result[0][ToolKeyName])
		})
	}
}

func TestToolsConfig_ToMCPTools(t *testing.T) {
	tests := []struct {
		name    string
		tc      *ToolsConfig
		wantNil bool
		wantLen int
	}{
		{
			name:    "nil ToolsConfig returns nil",
			tc:      nil,
			wantNil: true,
		},
		{
			name:    "empty Functions returns nil",
			tc:      &ToolsConfig{Functions: []*FunctionDef{}},
			wantNil: true,
		},
		{
			name: "single function uses inputSchema key",
			tc: &ToolsConfig{
				Functions: []*FunctionDef{
					{
						Name:       "mcp_tool",
						Parameters: map[string]any{ToolKeyType: SchemaTypeObject},
					},
				},
			},
			wantLen: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.tc.ToMCPTools()
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Len(t, result, tc.wantLen)
			assert.Equal(t, "mcp_tool", result[0][ToolKeyName])
			assert.NotNil(t, result[0][ToolKeyInputSchemaCamel])
		})
	}
}

func TestToolsConfig_ToMistralTools(t *testing.T) {
	tests := []struct {
		name    string
		tc      *ToolsConfig
		wantNil bool
		wantLen int
	}{
		{
			name:    "nil ToolsConfig returns nil",
			tc:      nil,
			wantNil: true,
		},
		{
			name:    "empty Functions returns nil",
			tc:      &ToolsConfig{Functions: []*FunctionDef{}},
			wantNil: true,
		},
		{
			name: "uses OpenAI format",
			tc: &ToolsConfig{
				Functions: []*FunctionDef{
					{Name: "mistral_tool", Description: "A Mistral tool"},
				},
			},
			wantLen: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.tc.ToMistralTools()
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Len(t, result, tc.wantLen)
			// Mistral uses OpenAI format
			assert.Equal(t, ToolKeyFunction, result[0][ToolKeyType])
		})
	}
}

func TestToolsConfig_ToCohereTools(t *testing.T) {
	tests := []struct {
		name     string
		tc       *ToolsConfig
		wantNil  bool
		wantLen  int
		validate func(t *testing.T, result []map[string]any)
	}{
		{
			name:    "nil ToolsConfig returns nil",
			tc:      nil,
			wantNil: true,
		},
		{
			name:    "empty Functions returns nil",
			tc:      &ToolsConfig{Functions: []*FunctionDef{}},
			wantNil: true,
		},
		{
			name: "multiple functions with flattened parameter_definitions",
			tc: &ToolsConfig{
				Functions: []*FunctionDef{
					{
						Name:        "cohere_a",
						Description: "Tool A",
						Parameters: map[string]any{
							ToolKeyType: SchemaTypeObject,
							SchemaKeyProperties: map[string]any{
								"x": map[string]any{ToolKeyType: SchemaTypeString},
							},
							ToolKeyRequired: []any{"x"},
						},
					},
					{Name: "cohere_b"},
				},
			},
			wantLen: 2,
			validate: func(t *testing.T, result []map[string]any) {
				assert.Equal(t, "cohere_a", result[0][ToolKeyName])
				defs, ok := result[0][ToolKeyParameterDefinitions].(map[string]any)
				require.True(t, ok)
				xDef, ok := defs["x"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, true, xDef[ToolKeyRequired])

				assert.Equal(t, "cohere_b", result[1][ToolKeyName])
				assert.Nil(t, result[1][ToolKeyParameterDefinitions])
			},
		},
		{
			name: "nil functions in slice are skipped",
			tc: &ToolsConfig{
				Functions: []*FunctionDef{
					nil,
					{Name: "cohere_valid"},
					nil,
				},
			},
			wantLen: 1,
			validate: func(t *testing.T, result []map[string]any) {
				assert.Equal(t, "cohere_valid", result[0][ToolKeyName])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.tc.ToCohereTools()
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Len(t, result, tc.wantLen)
			if tc.validate != nil {
				tc.validate(t, result)
			}
		})
	}
}
