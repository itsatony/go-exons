package a2a

import "encoding/json"

// jsonIndent matches root JSONIndentDefault. Duplicated here because a2a/ cannot import root.
const jsonIndent = "  "

// ToJSON serializes the Agent Card to compact JSON.
func (c *AgentCard) ToJSON() ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

// ToJSONPretty serializes the Agent Card to indented JSON.
func (c *AgentCard) ToJSONPretty() ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	return json.MarshalIndent(c, "", jsonIndent)
}
