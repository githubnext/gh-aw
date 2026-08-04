//go:build !js && !wasm

package parser

import "testing"

// TestParseWorkflowSpecPartsHostAllowlist ensures that host-prefixed
// workflowspecs (host/owner/repo/path[@ref]) are only accepted when the host
// is a recognized GitHub or GitHub Enterprise host. This prevents SSRF and
// conditional credential leakage (e.g. GH_ENTERPRISE_TOKEN) to arbitrary
// attacker-controlled hosts embedded in a nested import.
func TestParseWorkflowSpecPartsHostAllowlist(t *testing.T) {
	tests := []struct {
		name      string
		spec      string
		wantHost  string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "github.com host is allowed",
			spec:      "github.com/owner/repo/path/to/file.md",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "ghe.com host is allowed",
			spec:      "myorg.ghe.com/owner/repo/path/to/file.md",
			wantHost:  "myorg.ghe.com",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "raw.githubusercontent.com host is normalized to github.com",
			spec:      "raw.githubusercontent.com/owner/repo/path/to/file.md",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "repository name containing dots is allowed",
			spec:      "github.com/github/.github/path/to/file.md",
			wantHost:  "github.com",
			wantOwner: "github",
			wantRepo:  ".github",
		},
		{
			name:    "attacker host is rejected",
			spec:    "evil.com/owner/repo/path/to/file.md",
			wantErr: true,
		},
		{
			name:    "attacker host with valid-looking owner/repo is rejected",
			spec:    "evil.com/github/gh-aw/path/to/file.md",
			wantErr: true,
		},
		{
			name:    "invalid owner in host-prefixed spec is rejected",
			spec:    "github.com/-bad-/repo/path/to/file.md",
			wantErr: true,
		},
		{
			name:      "no host prefix defaults to empty host",
			spec:      "owner/repo/path/to/file.md",
			wantHost:  "",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, owner, repo, _, _, err := parseWorkflowSpecParts(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for spec %q, got none (host=%q owner=%q repo=%q)", tt.spec, host, owner, repo)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for spec %q: %v", tt.spec, err)
			}
			if host != tt.wantHost || owner != tt.wantOwner || repo != tt.wantRepo {
				t.Fatalf("parseWorkflowSpecParts(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.spec, host, owner, repo, tt.wantHost, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}
