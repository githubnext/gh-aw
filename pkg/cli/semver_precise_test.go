//go:build !integration

// This file tests the parseVersion wrapper in pkg/cli and the IsPreciseVersion/IsNewer
// methods it exposes from pkg/semverutil. For full semverutil parsing-logic coverage
// see pkg/semverutil/semverutil_test.go.
package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPreciseVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		wantNil  bool
		expected bool
	}{
		{
			name:     "major only - not precise",
			version:  "v6",
			expected: false,
		},
		{
			name:     "major.minor - not precise",
			version:  "v6.0",
			expected: false,
		},
		{
			name:     "major.minor.patch - precise",
			version:  "v6.0.0",
			expected: true,
		},
		{
			name:     "major.minor.patch non-zero - precise",
			version:  "v6.0.1",
			expected: true,
		},
		{
			name:     "full version - precise",
			version:  "v6.1.2",
			expected: true,
		},
		{
			name:     "without v prefix - precise",
			version:  "6.0.0",
			expected: true,
		},
		{
			name:     "single digit major - not precise",
			version:  "v1",
			expected: false,
		},
		{
			name:     "three component version - precise",
			version:  "v1.2.3",
			expected: true,
		},
		{
			name:    "invalid version string - returns nil",
			version: "not-a-version",
			wantNil: true,
		},
		{
			name:    "empty string - returns nil",
			version: "",
			wantNil: true,
		},
		{
			name:     "pre-release on precise core - still precise",
			version:  "v1.2.3-rc.1",
			expected: true,
		},
		{
			name:    "pre-release on imprecise core - invalid semver, returns nil",
			version: "v1.2-rc.1",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := parseVersion(tt.version)
			if tt.wantNil {
				require.Nil(t, v, "parseVersion(%q) should return nil", tt.version)
				return
			}
			require.NotNil(t, v, "parseVersion(%q) should not return nil", tt.version)
			assert.Equal(t, tt.expected, v.IsPreciseVersion(), "IsPreciseVersion() for %q", tt.version)
		})
	}
}

func TestPreciseVersionPreference(t *testing.T) {
	// Tests that when comparing equivalent versions, imprecise tags (major-only or
	// major.minor) are not considered precise, while full three-component versions are.
	// This follows GitHub Actions convention of distinguishing major version pins
	// (e.g. v8) from precise pins (e.g. v8.0.0).
	tests := []struct {
		name      string
		imprecise string
		precise   string
	}{
		{name: "major vs major.minor.patch", imprecise: "v6", precise: "v6.0.0"},
		{name: "major.minor vs major.minor.patch", imprecise: "v6.0", precise: "v6.0.0"},
		{name: "no-prefix major.minor vs major.minor.patch", imprecise: "6.1", precise: "v6.1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vImprecise := parseVersion(tt.imprecise)
			vPrecise := parseVersion(tt.precise)
			require.NotNil(t, vImprecise, "parseVersion(%q) should not return nil", tt.imprecise)
			require.NotNil(t, vPrecise, "parseVersion(%q) should not return nil", tt.precise)

			// Both versions should share the same major.minor.patch
			assert.Equal(t, vPrecise.Major, vImprecise.Major, "Major should match for %q and %q", tt.imprecise, tt.precise)
			assert.Equal(t, vPrecise.Minor, vImprecise.Minor, "Minor should match for %q and %q", tt.imprecise, tt.precise)
			assert.Equal(t, vPrecise.Patch, vImprecise.Patch, "Patch should match for %q and %q", tt.imprecise, tt.precise)

			assert.True(t, vPrecise.IsPreciseVersion(), "%q should be precise", tt.precise)
			assert.False(t, vImprecise.IsPreciseVersion(), "%q should not be precise", tt.imprecise)

			// Neither should be considered newer than the other
			assert.False(t, vImprecise.IsNewer(vPrecise), "%q should not be newer than %q", tt.imprecise, tt.precise)
			assert.False(t, vPrecise.IsNewer(vImprecise), "%q should not be newer than %q", tt.precise, tt.imprecise)
		})
	}
}
