//go:build !integration

package workflow

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.9.9", "2.0.0", -1},
		{"3.11", "3.9", 1},
		{"3.9", "3.11", -1},
		{"24", "20", 1},
		{"20", "24", -1},
	}

	for _, tt := range tests {
		t.Run(tt.v1+"_vs_"+tt.v2, func(t *testing.T) {
			result := compareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareVersions(%s, %s) = %d, expected %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestIsValidVersionTag(t *testing.T) {
	tests := []struct {
		tag   string
		valid bool
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
		{"abc123def456789012345678901234567890abcd", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			result := isValidVersionTag(tt.tag)
			if result != tt.valid {
				t.Errorf("isValidVersionTag(%q) = %v, want %v", tt.tag, result, tt.valid)
			}
		})
	}
}

func TestIsSemverCompatible(t *testing.T) {
	tests := []struct {
		pinVersion       string
		requestedVersion string
		expected         bool
	}{
		{"v5.0.0", "v5", true},
		{"v5.1.0", "v5.0.0", true},
		{"v6.0.0", "v5", false},
		{"v4.6.2", "v4", true},
		{"v4.6.2", "v5", false},
		{"v10.2.3", "v10", true},
	}

	for _, tt := range tests {
		t.Run(tt.pinVersion+"_"+tt.requestedVersion, func(t *testing.T) {
			result := isSemverCompatible(tt.pinVersion, tt.requestedVersion)
			if result != tt.expected {
				t.Errorf("isSemverCompatible(%q, %q) = %v, want %v",
					tt.pinVersion, tt.requestedVersion, result, tt.expected)
			}
		})
	}
}
