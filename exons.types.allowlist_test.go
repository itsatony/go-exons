package exons

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// go-exons#3 — an allow-list's EMPTY form ("no tools") must survive every
// re-emission, and its ABSENT form ("no narrowing") must stay absent. Each arm
// varies one thing — absent / empty / populated — and asserts the value that
// comes back, through the path consumers actually use (ExportFull → Parse)
// as well as the bare encoders.

const allowlistDoc = `---
name: allowlist-probe
description: probe
type: agent
execution:
  provider: anthropic
  model: claude-sonnet-4-6
tools:
%s
---
Hello.
`

func allowlistRoundTrip(t *testing.T, toolsYAML string) *ToolsConfig {
	t.Helper()
	spec, err := Parse([]byte(fmt.Sprintf(allowlistDoc, toolsYAML)))
	require.NoError(t, err)
	out, err := spec.ExportFull()
	require.NoError(t, err)
	again, err := Parse(out)
	require.NoError(t, err, "re-import of:\n%s", out)
	return again.Tools
}

func TestAllowListSurvivesExportRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name      string
		tools     string
		wantAllow []string // nil = absent
	}{
		{"populated", "  allow: [a, b]\n  functions:\n    - name: a\n", []string{"a", "b"}},
		{"EMPTY means no tools, and must not become absent", "  allow: []\n  functions:\n    - name: a\n", []string{}},
		{"absent stays absent", "  functions:\n    - name: a\n", nil},
		{"an allow-ONLY block is kept (it narrows runtime-added tools)", "  allow: [web_search]\n", []string{"web_search"}},
		{"an allow-only EMPTY block is kept", "  allow: []\n", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := allowlistRoundTrip(t, tc.tools)
			require.NotNil(t, got, "the tools block was dropped by the export")
			if tc.wantAllow == nil {
				assert.Nil(t, got.Allow)
				return
			}
			require.NotNil(t, got.Allow, "a present allow list came back absent — that is every tool")
			assert.Equal(t, tc.wantAllow, got.Allow)
		})
	}
}

func TestMCPServerToolsSurviveExportRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name      string
		tools     string
		wantTools []string
	}{
		{"populated", "      tools: [search]\n", []string{"search"}},
		{"EMPTY means none of the server's tools", "      tools: []\n", []string{}},
		{"absent stays absent", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := allowlistRoundTrip(t, "  mcp_servers:\n    - name: s\n      url: https://mcp.example.com/mcp\n      transport: sse\n"+tc.tools)
			require.Len(t, got.MCPServers, 1)
			assert.Equal(t, "sse", got.MCPServers[0].Transport, "transport is kept too")
			if tc.wantTools == nil {
				assert.Nil(t, got.MCPServers[0].Tools)
				return
			}
			require.NotNil(t, got.MCPServers[0].Tools)
			assert.Equal(t, tc.wantTools, got.MCPServers[0].Tools)
		})
	}
}

// The bare encoders, both directions, both formats: a host that stores the
// struct as JSON (a registry column) is on this path, not on ExportFull.
func TestAllowListEncodersKeepNilAndEmptyApart(t *testing.T) {
	for _, tc := range []struct {
		name  string
		allow []string
	}{
		{"nil", nil},
		{"empty", []string{}},
		{"populated", []string{"x"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := &ToolsConfig{Allow: tc.allow, MCPServers: []*MCPServer{{Name: "s", URL: "u", Tools: tc.allow}}}

			js, err := json.Marshal(in)
			require.NoError(t, err)
			var fromJSON ToolsConfig
			require.NoError(t, json.Unmarshal(js, &fromJSON))

			ys, err := yaml.Marshal(in)
			require.NoError(t, err)
			var fromYAML ToolsConfig
			require.NoError(t, yaml.Unmarshal(ys, &fromYAML))

			for label, got := range map[string]ToolsConfig{"json " + string(js): fromJSON, "yaml " + string(ys): fromYAML} {
				assert.Equal(t, tc.allow == nil, got.Allow == nil, "%s: allow nil-ness", label)
				assert.Equal(t, tc.allow == nil, got.MCPServers[0].Tools == nil, "%s: server tools nil-ness", label)
				assert.Len(t, got.Allow, len(tc.allow), label)
			}
		})
	}
}

func TestToolsConfigIsZero(t *testing.T) {
	yes := true
	assert.True(t, (*ToolsConfig)(nil).IsZero())
	assert.True(t, (&ToolsConfig{}).IsZero())
	for name, tc := range map[string]*ToolsConfig{
		"empty allow":         {Allow: []string{}},
		"tool_choice":         {ToolChoice: "none"},
		"parallel_tool_calls": {ParallelToolCalls: &yes},
		"functions":           {Functions: []*FunctionDef{{Name: "f"}}},
	} {
		assert.False(t, tc.IsZero(), name)
	}
}
