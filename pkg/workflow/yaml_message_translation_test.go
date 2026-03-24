//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
)

// TestTranslateYAMLMessage tests that raw goccy/go-yaml error messages are translated to plain English.
// It exercises parser.TranslateYAMLMessage, the shared translation function used by both packages.
func TestTranslateYAMLMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNot []string // substrings that must NOT appear in output
		wantAny []string // ALL of these must appear in output
	}{
		{
			name:    "non-map value translated to user-friendly message",
			input:   "non-map value is specified",
			wantNot: []string{"non-map value is specified"},
			wantAny: []string{"key: value", "colon"},
		},
		{
			name:    "mapping values not allowed translated",
			input:   "mapping values are not allowed in this context",
			wantNot: []string{"mapping values are not allowed"},
			wantAny: []string{"indentation"},
		},
		{
			name:    "did not find expected translated",
			input:   "did not find expected key",
			wantNot: []string{"did not find expected key"},
			wantAny: []string{"indentation"},
		},
		{
			name:    "unrecognized message returned unchanged",
			input:   "found unknown escape character 'z'",
			wantNot: []string{},
			wantAny: []string{"found unknown escape character 'z'"},
		},
		{
			name:    "empty message returned unchanged",
			input:   "",
			wantNot: []string{},
			wantAny: []string{""},
		},
		{
			name:    "partially matching message translated",
			input:   "[3:1] non-map value is specified as a key\n   2 | foo: bar\n>  3 | baz qux\n       ^",
			wantNot: []string{"non-map value is specified"},
			wantAny: []string{"key: value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.TranslateYAMLMessage(tt.input)

			for _, unwanted := range tt.wantNot {
				assert.NotContains(t, result, unwanted,
					"Result should not contain %q\nResult: %s", unwanted, result)
			}

			for _, wanted := range tt.wantAny {
				if wanted == "" {
					continue
				}
				assert.Contains(t, result, wanted,
					"Result should contain %q\nResult: %s", wanted, result)
			}
		})
	}
}
