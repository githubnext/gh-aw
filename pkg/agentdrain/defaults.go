package agentdrain

import (
	"bytes"
	_ "embed"

	"github.com/github/gh-aw/pkg/logger"
)

var defaultsLog = logger.New("agentdrain:defaults")

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
		defaultsLog.Print("No default weights embedded; skipping load")
		return nil
	}
	// A bare "{}" file means no weights have been trained yet.
	if string(bytes.TrimSpace(defaultWeightsJSON)) == "{}" {
		defaultsLog.Print("Default weights file is empty ({}); skipping load")
		return nil
	}
	defaultsLog.Printf("Loading embedded default weights: bytes=%d", len(defaultWeightsJSON))
	return c.LoadWeightsJSON(defaultWeightsJSON)
}
