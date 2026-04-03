package agentdrain

import (
	_ "embed"
)

//go:embed data/default_weights.json
var defaultWeightsJSON []byte

// LoadDefaultWeights restores all stage miners from the embedded default weights file
// (pkg/agentdrain/data/default_weights.json).  When the file is empty or contains
// only an empty JSON object the call is a no-op and returns nil.
//
// Update the default weights by running:
//
//	gh aw logs --train --output <dir>
//
// and copying the resulting drain3_weights.json to pkg/agentdrain/data/default_weights.json,
// then rebuilding the binary.
func (c *Coordinator) LoadDefaultWeights() error {
	if len(defaultWeightsJSON) == 0 {
		return nil
	}
	// A bare "{}" file means no weights have been trained yet.
	trimmed := trimWhitespace(defaultWeightsJSON)
	if string(trimmed) == "{}" {
		return nil
	}
	return c.LoadWeightsJSON(defaultWeightsJSON)
}

// trimWhitespace strips leading and trailing ASCII whitespace from a byte slice.
func trimWhitespace(b []byte) []byte {
	start := 0
	for start < len(b) && isSpace(b[start]) {
		start++
	}
	end := len(b)
	for end > start && isSpace(b[end-1]) {
		end--
	}
	return b[start:end]
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
