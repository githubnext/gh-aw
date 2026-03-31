//go:build !integration

package semverutil

import "testing"

func TestEnsureVPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{"1", "v1"},
		{"v1", "v1"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := EnsureVPrefix(tt.input)
			if got != tt.want {
				t.Errorf("EnsureVPrefix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsActionVersionTag(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"v0", true},
		{"v1", true},
		{"v10", true},
		{"v1.0", true},
		{"v1.2", true},
		{"v1.0.0", true},
		{"v1.2.3", true},
		{"v10.20.30", true},
		// invalid
		{"v1.0.0.0", false},
		{"1.0.0", false},
		{"v", false},
		{"", false},
		{"latest", false},
		{"v1.0.0-beta", false}, // prerelease not accepted
		{"abc123def456789012345678901234567890abcd", false},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := IsActionVersionTag(tt.tag)
			if got != tt.want {
				t.Errorf("IsActionVersionTag(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		ref  string
		want bool
	}{
		{"v1.0.0", true},
		{"1.0.0", true},
		{"v1.0", true},
		{"v1", true},
		{"v1.0.0-beta", true},
		{"v1.0.0+20230101", true},
		{"main", false},
		{"feature-branch", false},
		{"abc123def456", false},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got := IsValid(tt.ref)
			if got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.ref, got, tt.want)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMajor int
		wantMinor int
		wantPatch int
		wantPre   string
		wantNil   bool
	}{
		{
			name:      "full version with v",
			input:     "v1.2.3",
			wantMajor: 1,
			wantMinor: 2,
			wantPatch: 3,
		},
		{
			name:      "full version without v",
			input:     "1.2.3",
			wantMajor: 1,
			wantMinor: 2,
			wantPatch: 3,
		},
		{
			name:      "version with prerelease",
			input:     "v1.2.3-beta.1",
			wantMajor: 1,
			wantMinor: 2,
			wantPatch: 3,
			wantPre:   "beta.1",
		},
		{
			name:      "two-part version",
			input:     "v1.2",
			wantMajor: 1,
			wantMinor: 2,
		},
		{
			name:      "one-part version",
			input:     "v1",
			wantMajor: 1,
		},
		{
			name:    "invalid version",
			input:   "not-a-version",
			wantNil: true,
		},
		{
			name:    "branch name",
			input:   "main",
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseVersion(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ParseVersion(%q) = %+v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Errorf("ParseVersion(%q) = nil, want non-nil", tt.input)
				return
			}
			if got.Major != tt.wantMajor {
				t.Errorf("ParseVersion(%q).Major = %d, want %d", tt.input, got.Major, tt.wantMajor)
			}
			if got.Minor != tt.wantMinor {
				t.Errorf("ParseVersion(%q).Minor = %d, want %d", tt.input, got.Minor, tt.wantMinor)
			}
			if got.Patch != tt.wantPatch {
				t.Errorf("ParseVersion(%q).Patch = %d, want %d", tt.input, got.Patch, tt.wantPatch)
			}
			if got.Pre != tt.wantPre {
				t.Errorf("ParseVersion(%q).Pre = %q, want %q", tt.input, got.Pre, tt.wantPre)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		v1   string
		v2   string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"2.0.0", "1.9.9", 1},
		{"24", "20", 1},
		{"20", "24", -1},
	}
	for _, tt := range tests {
		t.Run(tt.v1+"_vs_"+tt.v2, func(t *testing.T) {
			got := Compare(tt.v1, tt.v2)
			if got != tt.want {
				t.Errorf("Compare(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestIsCompatible(t *testing.T) {
	tests := []struct {
		pin       string
		requested string
		want      bool
	}{
		{"v5.0.0", "v5", true},
		{"v5.1.0", "v5.0.0", true},
		{"v6.0.0", "v5", false},
		{"v4.6.2", "v4", true},
		{"v4.6.2", "v5", false},
		{"v10.2.3", "v10", true},
	}
	for _, tt := range tests {
		t.Run(tt.pin+"_"+tt.requested, func(t *testing.T) {
			got := IsCompatible(tt.pin, tt.requested)
			if got != tt.want {
				t.Errorf("IsCompatible(%q, %q) = %v, want %v", tt.pin, tt.requested, got, tt.want)
			}
		})
	}
}
