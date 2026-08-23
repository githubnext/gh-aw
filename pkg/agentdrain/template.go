package agentdrain

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var templateLog = logger.New("agentdrain:template")

// Tokenize splits a log line on whitespace and returns the individual tokens.
func Tokenize(line string) []string {
	return strings.Fields(line)
}

// computeSimilarity returns the fraction of positions where tokens a and b
// match exactly, considering only positions that are not paramToken in a.
// Returns 0 when the slices have different lengths.
func computeSimilarity(a, b []string, paramToken string) float64 {
	if len(a) != len(b) {
		if templateLog.Enabled() {
			templateLog.Printf("Similarity: length mismatch (%d vs %d), returning 0", len(a), len(b))
		}
		return 0
	}
	nonParam := 0
	matches := 0
	for i, tok := range a {
		if tok == paramToken {
			continue
		}
		nonParam++
		if tok == b[i] {
			matches++
		}
	}
	if nonParam == 0 {
		// All positions are wildcards – treat as a perfect structural match.
		return 1.0
	}
	sim := float64(matches) / float64(nonParam)
	if templateLog.Enabled() {
		templateLog.Printf("Similarity: matches=%d/%d non-param positions, score=%.3f", matches, nonParam, sim)
	}
	return sim
}

// mergeTemplate produces a new template by replacing positions where the two
// token slices differ with paramToken. Positions where either token already is
// paramToken also become paramToken.
func mergeTemplate(existing, incoming []string, paramToken string) []string {
	if len(existing) != len(incoming) {
		return existing
	}
	merged := make([]string, len(existing))
	for i, tok := range existing {
		if tok == paramToken || incoming[i] == paramToken || tok != incoming[i] {
			merged[i] = paramToken
		} else {
			merged[i] = tok
		}
	}
	return merged
}

// extractParams returns the token values at positions where the template has paramToken.
func extractParams(tokens []string, template []string, paramToken string) []string {
	var params []string
	for i, tok := range template {
		if tok == paramToken && i < len(tokens) {
			params = append(params, tokens[i])
		}
	}
	return params
}
