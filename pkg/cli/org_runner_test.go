//go:build !integration

package cli

import (
	"context"
	"strings"
	"testing"
)

func TestRunCommandForOrgValidationErrorsAreActionable(t *testing.T) {
	tests := []struct {
		name        string
		callbacks   orgRunCallbacks
		createPR    bool
		createIssue bool
		want        string
	}{
		{
			name: "missing search callback",
			want: "Expected orgRunCallbacks.SearchFn",
		},
		{
			name:      "missing report callback",
			callbacks: orgRunCallbacks{SearchFn: func(context.Context, string, bool) ([]string, error) { return nil, nil }},
			want:      "Expected orgRunCallbacks.ReportFn",
		},
		{
			name:     "missing apply callback",
			createPR: true,
			callbacks: orgRunCallbacks{
				SearchFn: func(context.Context, string, bool) ([]string, error) { return nil, nil },
				ReportFn: func([]orgRepoPreview, bool) {},
			},
			want: "Expected orgRunCallbacks.ApplyFn",
		},
		{
			name:        "missing issue callback",
			createIssue: true,
			callbacks: orgRunCallbacks{
				SearchFn: func(context.Context, string, bool) ([]string, error) { return nil, nil },
				ReportFn: func([]orgRepoPreview, bool) {},
			},
			want: "Expected orgRunCallbacks.IssueFn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCommandForOrg(context.Background(), "octo-org", nil, tt.callbacks, tt.createPR, tt.createIssue, false)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), "Example:") {
				t.Errorf("Expected actionable error containing %q and Example:, got %q", tt.want, err.Error())
			}
			if strings.Contains(err.Error(), "orgRunCallbacks{") {
				t.Errorf("Expected error not to expose raw struct literal syntax, got %q", err.Error())
			}
		})
	}
}

func TestRunCommandForOrgRequiresYesForCreateOperationsInCI(t *testing.T) {
	origIsRunningInCI := isRunningInCIFn
	isRunningInCIFn = func() bool { return true }
	t.Cleanup(func() { isRunningInCIFn = origIsRunningInCI })

	callbacks := orgRunCallbacks{
		SearchFn: func(context.Context, string, bool) ([]string, error) { return nil, nil },
		ReportFn: func([]orgRepoPreview, bool) {},
		ApplyFn:  func(context.Context, orgRepoPreview, bool) error { return nil },
	}

	err := runCommandForOrg(context.Background(), "octo-org", nil, callbacks, true, false, false)
	if err == nil {
		t.Fatal("expected CI confirmation error, got nil")
	}
	if !strings.Contains(err.Error(), "Expected --yes") || !strings.Contains(err.Error(), "Example: gh aw update --org octo-org --create-pull-request --yes") {
		t.Errorf("Expected actionable CI confirmation error, got %q", err.Error())
	}
}
