//go:build !integration

package workflow

import (
	"reflect"
	"sort"
	"testing"
)

func TestGitHubAppManifestPermissionKey(t *testing.T) {
	tests := []struct {
		name    string
		scope   PermissionScope
		wantKey string
		wantOK  bool
	}{
		{name: "hyphenated scope normalizes to underscore", scope: PermissionPullRequests, wantKey: "pull_requests", wantOK: true},
		{name: "security-events normalizes to underscore", scope: PermissionSecurityEvents, wantKey: "security_events", wantOK: true},
		{name: "single word scope unchanged", scope: PermissionContents, wantKey: "contents", wantOK: true},
		{name: "id-token has no manifest equivalent", scope: PermissionIdToken, wantOK: false},
		{name: "attestations has no manifest equivalent", scope: PermissionAttestations, wantOK: false},
		{name: "models has no manifest equivalent", scope: PermissionModels, wantOK: false},
		{name: "copilot-requests has no manifest equivalent", scope: PermissionCopilotRequests, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, ok := GitHubAppManifestPermissionKey(tt.scope)
			if ok != tt.wantOK {
				t.Fatalf("GitHubAppManifestPermissionKey(%q) ok = %v, want %v", tt.scope, ok, tt.wantOK)
			}
			if ok && key != tt.wantKey {
				t.Fatalf("GitHubAppManifestPermissionKey(%q) key = %q, want %q", tt.scope, key, tt.wantKey)
			}
		})
	}
}

func TestComputeGitHubAppManifestPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions any
		safeOutputs *SafeOutputsConfig
		want        map[string]string
	}{
		{
			name:        "nil permissions and no safe-outputs yields nil",
			permissions: nil,
			safeOutputs: nil,
			want:        nil,
		},
		{
			name: "top-level permissions normalize hyphenated keys to manifest keys",
			permissions: map[string]any{
				"pull-requests":   "write",
				"security-events": "read",
			},
			want: map[string]string{"pull_requests": "write", "security_events": "read"},
		},
		{
			name:        "none-level permissions are omitted",
			permissions: map[string]any{"issues": "none"},
			want:        nil,
		},
		{
			name:        "scopes without a manifest equivalent are dropped",
			permissions: map[string]any{"id-token": "write", "contents": "read"},
			want:        map[string]string{"contents": "read"},
		},
		{
			name:        "safe-outputs derive write permissions even when top-level is read-only",
			permissions: map[string]any{"issues": "read"},
			safeOutputs: SafeOutputsConfigFromKeys([]string{"create-issue"}),
			want:        map[string]string{"issues": "write"},
		},
		{
			name: "app-only scopes require explicit declaration, not read-all/write-all shorthand",
			permissions: map[string]any{
				"permissions": "write-all",
			},
		},
		{
			name:        "app-only scope explicitly declared is included",
			permissions: map[string]any{"administration": "write"},
			want:        map[string]string{"administration": "write"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeGitHubAppManifestPermissions(tt.permissions, tt.safeOutputs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ComputeGitHubAppManifestPermissions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeGitHubAppManifestPermissionsWriteAllShorthandExcludesAppOnlyScopes(t *testing.T) {
	got := ComputeGitHubAppManifestPermissions("write-all", nil)
	if _, ok := got["administration"]; ok {
		t.Fatalf("write-all shorthand must not implicitly grant GitHub App-only scopes, got %v", got)
	}
	if got["contents"] != "write" {
		t.Fatalf("write-all shorthand should still grant standard scopes, got %v", got)
	}
}

func TestComputeGitHubAppManifestWebhookPermissions(t *testing.T) {
	tests := []struct {
		name string
		on   any
		want map[string]string
	}{
		{
			name: "selected Projects and security events",
			on:   map[string]any{"gateway-dispatch": map[string]any{"projects-items": map[string]any{"project_id": "PVT_example"}, "code-scanning-alert": map[string]any{}, "dependabot-alerts": map[string]any{}}},
			want: map[string]string{
				"organization_projects": "read",
				"security_events":       "read",
				"vulnerability_alerts":  "read",
			},
		},
		{
			name: "only Dependabot alert trigger",
			on:   map[string]any{"gateway-dispatch": map[string]any{"dependabot-alerts": map[string]any{}}},
			want: map[string]string{"vulnerability_alerts": "read"},
		},
		{
			name: "only code scanning alert trigger",
			on:   map[string]any{"gateway-dispatch": map[string]any{"code-scanning-alert": map[string]any{}}},
			want: map[string]string{"security_events": "read"},
		},
		{
			name: "multi-repository alerts explicitly select both triggers",
			on:   map[string]any{"gateway-dispatch": map[string]any{"repos": []any{"bar/baz", "foo/*"}, "code-scanning-alert": map[string]any{}, "dependabot-alerts": map[string]any{}}},
			want: map[string]string{"security_events": "read", "vulnerability_alerts": "read"},
		},
		{
			name: "removed secret scanning source grants no permissions",
			on:   map[string]any{"gateway-dispatch": map[string]any{"dependabot-alerts": map[string]any{}, "secret-scanning-alert": map[string]any{}}},
			want: nil,
		},
		{
			name: "native events need no webhook permissions",
			on:   map[string]any{"issues": nil, "workflow_dispatch": nil},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeGitHubAppManifestWebhookPermissions(tt.on); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ComputeGitHubAppManifestWebhookPermissions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeGitHubAppWebhookEvents(t *testing.T) {
	tests := []struct {
		name string
		on   any
		want []string
	}{
		{name: "nil on value", on: nil, want: nil},
		{name: "string trigger", on: "issues", want: []string{"issues"}},
		{name: "list of triggers", on: []any{"issues", "pull_request"}, want: []string{"issues", "pull_request"}},
		{
			name: "map excludes non-webhook triggers",
			on: map[string]any{
				"issues":              map[string]any{"types": []any{"opened"}},
				"schedule":            []any{map[string]any{"cron": "0 0 * * *"}},
				"workflow_dispatch":   nil,
				"repository_dispatch": nil,
				"workflow_call":       nil,
			},
			want: []string{"issues"},
		},
		{
			name: "gh-aw compiler-only keys excluded",
			on: map[string]any{
				"issues":         nil,
				"reaction":       "eyes",
				"status-comment": true,
			},
			want: []string{"issues"},
		},
		{
			name: "first-class Projects v2 and security triggers use wire event names",
			on:   map[string]any{"gateway-dispatch": map[string]any{"projects-items": map[string]any{"project_id": "PVT_example", "types": []any{"edited"}}, "code-scanning-alert": map[string]any{}, "dependabot-alerts": map[string]any{}}},
			want: []string{
				"code_scanning_alert", "dependabot_alert", "projects_v2_item",
			},
		},
		{
			name: "removed secret scanning source yields no webhook events",
			on:   map[string]any{"gateway-dispatch": map[string]any{"secret-scanning-alert": map[string]any{}}},
			want: nil,
		},
		{
			name: "exclusive gateway rejects native enrollment",
			on: map[string]any{"issues": nil, "gateway-dispatch": map[string]any{
				"projects-items": map[string]any{"project_id": "PVT_example"},
			}},
			want: nil,
		},
		{
			name: "pull_request_target maps to pull_request",
			on:   map[string]any{"pull_request_target": nil},
			want: []string{"pull_request"},
		},
		{
			name: "slash command shorthand expands to underlying webhook events",
			on:   "/my-bot",
			want: []string{"issue_comment", "issues", "pull_request", "pull_request_review_comment"},
		},
		{
			name: "slash_command key expands to underlying webhook events",
			on:   map[string]any{"slash_command": map[string]any{"name": "my-bot"}},
			want: []string{"issue_comment", "issues", "pull_request", "pull_request_review_comment"},
		},
		{
			name: "label_command key expands to underlying webhook events",
			on:   map[string]any{"label_command": map[string]any{"name": "my-label"}},
			want: []string{"discussion", "issues", "pull_request"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeGitHubAppWebhookEvents(tt.on)
			sort.Strings(got)
			want := append([]string(nil), tt.want...)
			sort.Strings(want)
			if len(got) == 0 && len(want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("NormalizeGitHubAppWebhookEvents(%v) = %v, want %v", tt.on, got, want)
			}
		})
	}
}
