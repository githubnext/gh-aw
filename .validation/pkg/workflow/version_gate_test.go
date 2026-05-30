//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		name           string
		versionToCheck string
		defaultVersion string
		minVersion     string
		want           bool
	}{
		{
			name:           "empty version uses default above minimum",
			versionToCheck: "",
			defaultVersion: "1.2.0",
			minVersion:     "1.1.0",
			want:           true,
		},
		{
			name:           "empty version uses default below minimum",
			versionToCheck: "",
			defaultVersion: "1.0.0",
			minVersion:     "1.1.0",
			want:           false,
		},
		{
			name:           "explicit version at minimum",
			versionToCheck: "1.1.0",
			defaultVersion: "9.9.9",
			minVersion:     "1.1.0",
			want:           true,
		},
		{
			name:           "latest is always supported",
			versionToCheck: "latest",
			defaultVersion: "1.0.0",
			minVersion:     "9.9.9",
			want:           true,
		},
		{
			name:           "LATEST is always supported",
			versionToCheck: "LATEST",
			defaultVersion: "1.0.0",
			minVersion:     "9.9.9",
			want:           true,
		},
		{
			name:           "non semver returns false",
			versionToCheck: "main",
			defaultVersion: "1.2.0",
			minVersion:     "1.1.0",
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := versionAtLeast(tt.versionToCheck, tt.defaultVersion, tt.minVersion)
			assert.Equal(t, tt.want, got, "versionAtLeast result")
		})
	}
}
