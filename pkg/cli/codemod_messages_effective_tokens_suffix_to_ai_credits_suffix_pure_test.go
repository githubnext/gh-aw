package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMigrateMessagesEffectiveTokensSuffixToAICreditsSuffix covers the pure
// line-based rewrite that migrates safe-outputs.messages placeholders from
// {effective_tokens_suffix} to {ai_credits_suffix}.
func TestMigrateMessagesEffectiveTokensSuffixToAICreditsSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		lines        []string
		wantModified bool
		wantLines    []string
	}{
		{
			name:         "no safe-outputs block",
			lines:        []string{"on: push", "permissions: {}"},
			wantModified: false,
			wantLines:    []string{"on: push", "permissions: {}"},
		},
		{
			name: "no messages block under safe-outputs",
			lines: []string{
				"safe-outputs:",
				"  create-issue:",
			},
			wantModified: false,
			wantLines: []string{
				"safe-outputs:",
				"  create-issue:",
			},
		},
		{
			name: "replaces placeholder in simple scalar message",
			lines: []string{
				"safe-outputs:",
				"  messages:",
				"    footer: \"Cost {effective_tokens_suffix}\"",
			},
			wantModified: true,
			wantLines: []string{
				"safe-outputs:",
				"  messages:",
				"    footer: \"Cost {ai_credits_suffix}\"",
			},
		},
		{
			name: "no placeholder present leaves lines untouched",
			lines: []string{
				"safe-outputs:",
				"  messages:",
				"    footer: \"Cost: $0\"",
			},
			wantModified: false,
			wantLines: []string{
				"safe-outputs:",
				"  messages:",
				"    footer: \"Cost: $0\"",
			},
		},
		{
			name: "replaces placeholder inside block scalar",
			lines: []string{
				"safe-outputs:",
				"  messages:",
				"    footer: |",
				"      Cost {effective_tokens_suffix}",
				"      more {effective_tokens_suffix} text",
				"    other: value",
			},
			wantModified: true,
			wantLines: []string{
				"safe-outputs:",
				"  messages:",
				"    footer: |",
				"      Cost {ai_credits_suffix}",
				"      more {ai_credits_suffix} text",
				"    other: value",
			},
		},
		{
			name: "does not touch keys outside messages block",
			lines: []string{
				"safe-outputs:",
				"  create-issue:",
				"    title-prefix: \"{effective_tokens_suffix}\"",
				"  messages:",
				"    footer: \"{effective_tokens_suffix}\"",
			},
			wantModified: true,
			wantLines: []string{
				"safe-outputs:",
				"  create-issue:",
				"    title-prefix: \"{effective_tokens_suffix}\"",
				"  messages:",
				"    footer: \"{ai_credits_suffix}\"",
			},
		},
		{
			name: "stops applying once outside safe-outputs block",
			lines: []string{
				"safe-outputs:",
				"  messages:",
				"    footer: \"{effective_tokens_suffix}\"",
				"on: push",
				"env:",
				"  X: \"{effective_tokens_suffix}\"",
			},
			wantModified: true,
			wantLines: []string{
				"safe-outputs:",
				"  messages:",
				"    footer: \"{ai_credits_suffix}\"",
				"on: push",
				"env:",
				"  X: \"{effective_tokens_suffix}\"",
			},
		},
		{
			name:         "empty input",
			lines:        []string{},
			wantModified: false,
			wantLines:    []string{},
		},
		{
			name: "comments are not treated as keys",
			lines: []string{
				"safe-outputs:",
				"  messages:",
				"    # a comment mentioning {effective_tokens_suffix}",
				"    footer: \"{effective_tokens_suffix}\"",
			},
			wantModified: true,
			wantLines: []string{
				"safe-outputs:",
				"  messages:",
				"    # a comment mentioning {effective_tokens_suffix}",
				"    footer: \"{ai_credits_suffix}\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotLines, gotModified := migrateMessagesEffectiveTokensSuffixToAICreditsSuffix(tt.lines)
			assert.Equal(t, tt.wantModified, gotModified, "modified flag mismatch")
			assert.Equal(t, tt.wantLines, gotLines, "resulting lines mismatch")
		})
	}
}

// TestMigrateMessagesEffectiveTokensSuffixToAICreditsSuffixPurity verifies the
// function neither mutates its input slice nor produces different output
// across repeated calls with identical input (purity/determinism check).
func TestMigrateMessagesEffectiveTokensSuffixToAICreditsSuffixPurity(t *testing.T) {
	t.Parallel()

	original := []string{
		"safe-outputs:",
		"  messages:",
		"    footer: \"{effective_tokens_suffix}\"",
	}
	inputCopy := make([]string, len(original))
	copy(inputCopy, original)

	result1, modified1 := migrateMessagesEffectiveTokensSuffixToAICreditsSuffix(inputCopy)
	assert.Equal(t, original, inputCopy, "input lines were mutated")

	result2, modified2 := migrateMessagesEffectiveTokensSuffixToAICreditsSuffix(inputCopy)
	assert.Equal(t, result1, result2, "results differ across repeated calls with identical input")
	assert.Equal(t, modified1, modified2)
}
