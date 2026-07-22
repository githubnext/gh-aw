//go:build !integration && !js && !wasm

package console

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmAction(t *testing.T) {
	// Note: This test can't fully test the interactive behavior without mocking
	// the terminal input, but we can verify the function signature and basic setup

	t.Run("function signature", func(t *testing.T) {
		// This test just verifies the function exists and has the right signature
		// Actual interactive testing would require a mock terminal
		_ = ConfirmAction
	})
}

func TestShowTextConfirm(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantResult  bool
		wantErr     bool
		errContains string
	}{
		{name: "yes", input: "y\n", wantResult: true},
		{name: "YES uppercase", input: "YES\n", wantResult: true},
		{name: "yes full", input: "yes\n", wantResult: true},
		{name: "1 for affirmative", input: "1\n", wantResult: true},
		{name: "no", input: "n\n", wantResult: false},
		{name: "NO uppercase", input: "NO\n", wantResult: false},
		{name: "no full", input: "no\n", wantResult: false},
		{name: "2 for negative", input: "2\n", wantResult: false},
		{name: "invalid input", input: "maybe\n", wantErr: true, errContains: "invalid input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := showTextConfirm("Delete all workflows?", "Yes, delete", "Cancel", strings.NewReader(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.ErrorContains(t, err, tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
		})
	}
}
