//go:build !integration

package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWorkflowStatuses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		pattern      string
		repoOverride string
		verbose      bool
		// We can't easily test the actual results without mocking gh CLI,
		// so we just verify the function doesn't panic
	}{
		{
			name:         "simple pattern",
			pattern:      "test-workflow",
			repoOverride: "",
			verbose:      false,
		},
		{
			name:         "with repo override",
			pattern:      "daily-status",
			repoOverride: "owner/repo",
			verbose:      false,
		},
		{
			name:         "verbose mode",
			pattern:      "workflow-name",
			repoOverride: "owner/repo",
			verbose:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// This function calls gh CLI, so it will likely error in tests
			// We just verify it doesn't panic
			statuses, err := findWorkflowsByFilenamePattern(tt.pattern, tt.repoOverride, tt.verbose)

			// Either succeeds or fails gracefully, but shouldn't panic
			// Note: statuses may be nil even when err is nil (no workflows found)
			if err != nil {
				// Error is acceptable in test environment without gh CLI setup
				require.Error(t, err, "Expected error without gh CLI")
			}
			// If no error and statuses exist, verify they have expected structure
			if err == nil && statuses != nil {
				for _, status := range statuses {
					assert.NotEmpty(t, status.Workflow, "Workflow name should not be empty")
				}
			}
		})
	}
}

func TestCheckStatusAndOfferRun_ContextCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	cfg := &AddInteractiveConfig{Verbose: true}
	err := cfg.checkStatusAndOfferRun(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}
}

func TestShouldOfferAddedWorkflowRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  AddInteractiveConfig
		want bool
	}{
		{
			name: "no add result",
			cfg:  AddInteractiveConfig{},
			want: false,
		},
		{
			name: "workflow dispatch unavailable",
			cfg: AddInteractiveConfig{
				addResult: &AddWorkflowsResult{HasWorkflowDispatch: false},
			},
			want: false,
		},
		{
			name: "workflow dispatch available",
			cfg: AddInteractiveConfig{
				addResult: &AddWorkflowsResult{HasWorkflowDispatch: true},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.cfg.shouldOfferAddedWorkflowRun())
		})
	}
}
