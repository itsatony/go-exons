package exons

import "strconv"

// MaxToolAllowEntries bounds tools.allow and every mcp_servers[].tools list
// (v0.40.0). A runtime's whole catalogue is far below it (atlas's per-turn
// capability ceiling is 512), so a longer list is a paste, not a narrowing.
const MaxToolAllowEntries = 512

// Validate checks the tool allow-lists: tools.allow and each
// mcp_servers[].tools entry must be non-empty, unique within its list, and the
// list at most MaxToolAllowEntries long. Names are compared VERBATIM — never
// trimmed or case-folded — because a runtime matches them against tool names
// exactly, and a validator that normalised would bless a name nothing matches.
//
// ⚠ BEHAVIOUR CHANGE in v0.40.0: before it, nothing validated these lists. A
// duplicate was harmless to a set-based runtime but an empty string or an
// oversized list was accepted into every store. nil and [] are both still
// valid and still mean different things (absent = no narrowing, [] = none).
// Safe on a nil receiver.
func (tc *ToolsConfig) Validate() error {
	if tc == nil {
		return nil
	}
	if err := validateToolNameList(tc.Allow, SpecFieldTools+"."+ToolsFieldAllow); err != nil {
		return err
	}
	for i, srv := range tc.MCPServers {
		if srv == nil {
			continue
		}
		field := SpecFieldTools + "." + ToolsFieldMCPServers + "[" + strconv.Itoa(i) + "]." + ToolsFieldMCPServerTools
		if err := validateToolNameList(srv.Tools, field); err != nil {
			return err
		}
	}
	return nil
}

// validateToolNameList is the one rule both allow-lists share.
func validateToolNameList(names []string, field string) error {
	if len(names) > MaxToolAllowEntries {
		return NewSpecValidationError(ErrMsgToolAllowTooMany, field)
	}
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		if n == "" {
			return NewSpecValidationError(ErrMsgToolAllowEntryEmpty, field)
		}
		if _, dup := seen[n]; dup {
			return NewSpecValidationError(ErrMsgToolAllowEntryDup, field+"="+n)
		}
		seen[n] = struct{}{}
	}
	return nil
}
